package docker

import (
	"fmt"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ContainerSolutions/flux/platform"

	"golang.org/x/net/context"
)

type Swarm struct {
	client *client.Client
	logger log.Logger
}

func NewSwarm(logger log.Logger) (*Swarm, error) {
	cli, err := client.NewEnvClient()

	if err != nil {
		panic(err)
	}

	c := &Swarm{
		client: cli,
		logger: logger,
	}

	return c, nil
}

func (c *Swarm) Apply(defs []platform.ServiceDefinition) error {
	stack_name := "default_swarm"
	bin, err := findBinary("docker")

	if err != nil {
		c.logger.Log(err)
	}

	for _, svc := range defs {
		tmpfile, err := ioutil.TempFile("", "temp")
		if err != nil {
			c.logger.Log(err)
		}

		defer os.Remove(tmpfile.Name()) //clean up

		if _, err := tmpfile.Write(svc.NewDefinition); err != nil {
			c.logger.Log(err)
		}
		if err := tmpfile.Close(); err != nil {
			c.logger.Log(err)
		}

		cmd := exec.Command(bin, "deploy", "-c", tmpfile.Name(), stack_name)
		err = cmd.Run()
	}

	return err
}

func (c *Swarm) Sync(spec platform.SyncDef) error {
	return nil
}

func (c *Swarm) Ping() error {
	return nil
}

func (c *Swarm) Export() ([]byte, error) {
	return nil, nil
}

func (c *Swarm) Version() (string, error) {
	ctx := context.Background()
	version, err := c.client.ServerVersion(ctx)
	return version.Version, err
}

func findBinary(name string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	localBin := filepath.Join(cwd, name)
	if _, err := os.Stat(localBin); err == nil {
		return localBin, nil
	}
	if pathBin, err := exec.LookPath(name); err == nil {
		return pathBin, nil
	}
	return "", fmt.Errorf("%s not found", name)
}
