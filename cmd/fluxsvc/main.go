package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"

	"github.com/ContainerSolutions/flux/automator"
	"github.com/ContainerSolutions/flux/db"
	"github.com/ContainerSolutions/flux/history"
	historysql "github.com/ContainerSolutions/flux/history/sql"
	transport "github.com/ContainerSolutions/flux/http"
	httpserver "github.com/ContainerSolutions/flux/http/server"
	"github.com/ContainerSolutions/flux/instance"
	instancedb "github.com/ContainerSolutions/flux/instance/sql"
	"github.com/ContainerSolutions/flux/jobs"
	"github.com/ContainerSolutions/flux/platform"
	"github.com/ContainerSolutions/flux/platform/rpc/nats"
	"github.com/ContainerSolutions/flux/registry"
	"github.com/ContainerSolutions/flux/release"
	"github.com/ContainerSolutions/flux/server"
)

const shutdownTimeout = 30 * time.Second

var version string

func main() {
	// Flag domain.
	fs := pflag.NewFlagSet("default", pflag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "DESCRIPTION\n")
		fmt.Fprintf(os.Stderr, "  fluxsvc is a deployment service.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		fs.PrintDefaults()
	}

	var (
		listenAddr            = fs.StringP("listen", "l", ":3030", "Listen address for Flux API clients")
		databaseSource        = fs.String("database-source", "file://fluxy.db", `Database source name; includes the DB driver as the scheme. The default is a temporary, file-based DB`)
		databaseMigrationsDir = fs.String("database-migrations", "./db/migrations", "Path to database migration scripts, which are in subdirectories named for each driver")
		natsURL               = fs.String("nats-url", "", `URL on which to connect to NATS, or empty to use the standalone message bus (e.g., "nats://user:pass@nats:4222")`)
		memcachedHostname     = fs.String("memcached-hostname", "", "Hostname for memcached service to use when caching chunks. If empty, no memcached will be used.")
		memcachedTimeout      = fs.Duration("memcached-timeout", 100*time.Millisecond, "Maximum time to wait before giving up on memcached requests.")
		memcachedService      = fs.String("memcached-service", "memcached", "SRV service used to discover memcache servers.")
		registryCacheExpiry   = fs.Duration("registry-cache-expiry", 20*time.Minute, "Duration to keep cached registry tag info. Must be < 1 month.")
		versionFlag           = fs.Bool("version", false, "Get version number")
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

	// Initialise database; we must fail if we can't do this, because
	// most things depend on it.
	var dbDriver string
	{
		var version uint64
		u, err := url.Parse(*databaseSource)
		if err == nil {
			version, err = db.Migrate(*databaseSource, *databaseMigrationsDir)
		}

		if err != nil {
			logger.Log("stage", "db init", "err", err)
			os.Exit(1)
		}
		dbDriver = db.DriverForScheme(u.Scheme)
		logger.Log("migrations", "success", "driver", dbDriver, "db-version", fmt.Sprintf("%d", version))
	}

	var messageBus platform.MessageBus
	{
		if *natsURL != "" {
			bus, err := nats.NewMessageBus(*natsURL, platform.BusMetricsImpl)
			if err != nil {
				logger.Log("component", "message bus", "err", err)
				os.Exit(1)
			}
			logger.Log("component", "message bus", "type", "NATS")
			messageBus = bus
		} else {
			messageBus = platform.NewStandaloneMessageBus(platform.BusMetricsImpl)
			logger.Log("component", "message bus", "type", "standalone")
		}
	}

	var historyDB history.DB
	{
		db, err := historysql.NewSQL(dbDriver, *databaseSource)
		if err != nil {
			logger.Log("component", "history", "err", err)
			os.Exit(1)
		}
		historyDB = history.InstrumentedDB(db)
	}

	// Configuration, i.e., whether services are automated or not.
	var instanceDB instance.DB
	{
		db, err := instancedb.New(dbDriver, *databaseSource)
		if err != nil {
			logger.Log("component", "config", "err", err)
			os.Exit(1)
		}
		instanceDB = instance.InstrumentedDB(db)
	}

	var memcacheClient registry.MemcacheClient
	if *memcachedHostname != "" {
		memcacheClient = registry.NewMemcacheClient(registry.MemcacheConfig{
			Host:           *memcachedHostname,
			Service:        *memcachedService,
			Timeout:        *memcachedTimeout,
			UpdateInterval: 1 * time.Minute,
			Logger:         log.NewContext(logger).With("component", "memcached"),
		})
		memcacheClient = registry.InstrumentMemcacheClient(memcacheClient)
		defer memcacheClient.Stop()
	}

	var instancer instance.Instancer
	{
		// Instancer, for the instancing of operations
		instancer = &instance.MultitenantInstancer{
			DB:                  instanceDB,
			Connecter:           messageBus,
			Logger:              logger,
			History:             historyDB,
			MemcacheClient:      memcacheClient,
			RegistryCacheExpiry: *registryCacheExpiry,
		}
	}

	// Job store.
	var jobStore jobs.JobStore
	{
		s, err := jobs.NewDatabaseStore(dbDriver, *databaseSource, time.Hour)
		if err != nil {
			logger.Log("component", "release job store", "err", err)
			os.Exit(1)
		}
		jobStore = jobs.InstrumentedJobStore(s)
	}

	// Automator component.
	var auto *automator.Automator
	{
		var err error
		auto, err = automator.New(automator.Config{
			Jobs:       jobStore,
			InstanceDB: instanceDB,
			Instancer:  instancer,
			Logger:     log.NewContext(logger).With("component", "automator"),
		})
		if err == nil {
			logger.Log("automator", "enabled")
		} else {
			// Service can handle a nil automator pointer.
			logger.Log("automator", "disabled", "reason", err)
		}
	}

	go auto.Start(log.NewContext(logger).With("component", "automator"))

	// Job workers.
	//
	// Doing one worker (and one queue) for each job type for now. This way slow
	// release jobs can't interfere with slow automated service jobs, or vice
	// versa. This is probably not optimal. Really all jobs should be quick and
	// recoverable.
	for _, queue := range []string{
		jobs.DefaultQueue,
		jobs.ReleaseJob,
		jobs.AutomatedInstanceJob,
	} {
		logger := log.NewContext(logger).With("component", "worker", "queues", fmt.Sprint([]string{queue}))
		worker := jobs.NewWorker(jobStore, logger, []string{queue})
		worker.Register(jobs.AutomatedInstanceJob, auto)
		worker.Register(jobs.ReleaseJob, release.NewReleaser(instancer))

		defer func() {
			logger.Log("stopping", "true")
			if err := worker.Stop(shutdownTimeout); err != nil {
				logger.Log("err", err)
			}
		}()
		go worker.Work()

	}

	// Job GC cleaner
	{
		cleaner := jobs.NewCleaner(jobStore, logger)
		cleanTicker := time.NewTicker(15 * time.Second)
		defer cleanTicker.Stop()
		go cleaner.Clean(cleanTicker.C)
	}

	// The server.
	server := server.New(version, instancer, instanceDB, messageBus, jobStore, logger)

	// Mechanical components.
	errc := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// HTTP transport component.
	go func() {
		logger.Log("addr", *listenAddr)
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		handler := httpserver.NewHandler(server, transport.NewRouter(), logger)
		mux.Handle("/", handler)
		mux.Handle("/api/flux/", http.StripPrefix("/api/flux", handler))
		errc <- http.ListenAndServe(*listenAddr, mux)
	}()

	logger.Log("exiting", <-errc)
}
