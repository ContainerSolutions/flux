package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"k8s.io/client-go/1.5/rest"

	"github.com/ContainerSolutions/flux"
	transport "github.com/ContainerSolutions/flux/http"
	"github.com/ContainerSolutions/flux/platform"
	"github.com/ContainerSolutions/flux/platform/docker"
	"github.com/ContainerSolutions/flux/platform/kubernetes"
)

var version string

func main() {
	// Flag domain.
	fs := pflag.NewFlagSet("default", pflag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION\n")
		fmt.Fprintf(os.Stderr, "  fluxd is the agent of flux.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		fs.PrintDefaults()
	}
	// This mirrors how kubectl extracts information from the environment.
	var (
		listenAddr        = fs.StringP("listen", "l", ":3031", "Listen address where /metrics will be served")
		fluxsvcAddress    = fs.String("fluxsvc-address", "wss://cloud.weave.works/api/flux", "Address of the fluxsvc to connect to.")
		token             = fs.String("token", "", "Token to use to authenticate with flux service")
		kubernetesKubectl = fs.String("kubernetes-kubectl", "", "Optional, explicit path to kubectl tool")
		versionFlag       = fs.Bool("version", false, "Get version number")
	)
	fs.Parse(os.Args)

	if version == "" {
		version = "unversioned"
	}
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	// Logger component.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}

	// Platform component.
	// When adding a new platform, don't just bash it in. Create a Platform
	// or Cluster interface in package platform, and have kubernetes.Cluster
	// and your new platform implement that interface.
	var fluxPlatform platform.Platform
	{
		logger := log.NewContext(logger).With("component", "platform")

		restClientConfig, err := rest.InClusterConfig()
		if err != nil {
			logger.Log("err", err)
		} else {
			restClientConfig.QPS = 50.0
			restClientConfig.Burst = 100

			logger.Log("host", restClientConfig.Host)

			kubectl := *kubernetesKubectl
			if kubectl == "" {
				kubectl, err = exec.LookPath("kubectl")
			} else {
				_, err = os.Stat(kubectl)
			}
			if err != nil {
				logger.Log("err", err)
			} else {
				logger.Log("kubectl", kubectl)

				kubectlApplier := kubernetes.NewKubectl(kubectl, restClientConfig)
				cluster, _ := kubernetes.NewCluster(restClientConfig, kubectlApplier, version, logger)
				if err != nil {
					logger.Log("err", err)
				} else {

					if services, err := cluster.AllServices("", nil); err != nil {
						logger.Log("services", err)
					} else {
						logger.Log("services", len(services))
					}

					logger.Log("info", "kubernetes platform")
					fluxPlatform = cluster
				}
			}
		}

		swarm, err := docker.NewSwarm(logger)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		logger.Log("info", "swarm platform")
		fluxPlatform = swarm
	}

	// Connect to fluxsvc
	daemonLogger := log.NewContext(logger).With("component", "client")
	daemon, err := transport.NewDaemon(
		&http.Client{Timeout: 10 * time.Second},
		fmt.Sprintf("fluxd/%v", version),
		flux.Token(*token),
		transport.NewRouter(),
		*fluxsvcAddress,
		fluxPlatform,
		daemonLogger,
	)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}
	defer daemon.Close()

	// Mechanical components.
	errc := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// HTTP transport component, for metrics
	go func() {
		logger.Log("addr", *listenAddr)
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		errc <- http.ListenAndServe(*listenAddr, mux)
	}()

	// Go!
	logger.Log("exiting", <-errc)
}
