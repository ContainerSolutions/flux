package notifications

import (
	"github.com/ContainerSolutions/flux"
	"github.com/ContainerSolutions/flux/instance"
)

// Release performs post-release notifications for an instance
func Release(cfg instance.Config, r flux.Release, releaseError error) error {
	if r.Spec.Kind != flux.ReleaseKindExecute {
		return nil
	}

	// TODO: Use a config settings format which allows multiple notifiers to be
	// configured.
	var err error
	if cfg.Settings.Slack.HookURL != "" {
		err = slackNotifyRelease(cfg.Settings.Slack, r, releaseError)
	}
	return err
}
