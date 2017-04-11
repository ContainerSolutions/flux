package instance

import (
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/ContainerSolutions/flux"
	"github.com/ContainerSolutions/flux/git"
	"github.com/ContainerSolutions/flux/history"
	"github.com/ContainerSolutions/flux/platform"
	"github.com/ContainerSolutions/flux/registry"
)

// StandaloneInstancer is the instancer for standalone mode
type StandaloneInstancer struct {
	Instance    flux.InstanceID
	Connecter   platform.Connecter
	Registry    registry.Registry
	Config      Configurer
	GitRepo     git.Repo
	EventReader history.EventReader
	EventWriter history.EventWriter
	BaseLogger  log.Logger
}

func (s StandaloneInstancer) Get(inst flux.InstanceID) (*Instance, error) {
	if inst != s.Instance {
		return nil, errors.New("cannot find instance with ID: " + string(inst))
	}
	platform, err := s.Connecter.Connect(inst)
	if err != nil {
		return nil, errors.New("cannot get platform for instance")
	}
	return New(
		platform,
		s.Registry,
		s.Config,
		s.GitRepo,
		log.NewContext(s.BaseLogger).With("instanceID", s.Instance),
		s.EventReader,
		s.EventWriter,
	), nil
}
