package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"

	"github.com/ContainerSolutions/flux"
)

type getConfigOpts struct {
	*rootOpts
	fingerprint string
	output      string
}

func newGetConfig(parent *rootOpts) *getConfigOpts {
	return &getConfigOpts{rootOpts: parent}
}

func (opts *getConfigOpts) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-config",
		Short: "display configuration values for an instance",
		Example: makeExample(
			"fluxctl config --output=yaml",
		),
		RunE: opts.RunE,
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "yaml", `The format to output ("yaml" or "json")`)
	cmd.Flags().StringVar(&opts.fingerprint, "fingerprint", "", `Show a fingerprint of the public key, using the hash given ("md5" or "sha256")`)
	return cmd
}

func (opts *getConfigOpts) RunE(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errorWantedNoArgs
	}

	var marshal func(interface{}) ([]byte, error)
	switch opts.output {
	case "yaml":
		marshal = yaml.Marshal
	case "json":
		marshal = func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		}
	default:
		return errors.New("unknown output format " + opts.output)
	}

	config, err := opts.API.GetConfig(noInstanceID)

	if err != nil {
		return err
	}

	if opts.fingerprint != "" && config.Git.Key != "" {
		pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(config.Git.Key))
		if err != nil {
			config.Git.Key = "unable to parse public key"
		} else {
			switch opts.fingerprint {
			case "md5":
				hash := md5.Sum(pk.Marshal())
				fingerprint := ""
				for i, b := range hash {
					fingerprint = fmt.Sprintf("%s%0.2x", fingerprint, b)
					if i < len(hash)-1 {
						fingerprint = fingerprint + ":"
					}
				}
				config.Git.Key = fingerprint
			case "sha256":
				hash := sha256.Sum256(pk.Marshal())
				config.Git.Key = strings.TrimRight(base64.StdEncoding.EncodeToString(hash[:]), "=")
			}
		}
	}

	// Since we always want to output whatever we got, use UnsafeInstanceConfig
	bytes, err := marshal(flux.UnsafeInstanceConfig(config))
	if err != nil {
		return errors.Wrap(err, "marshalling to output format "+opts.output)
	}
	os.Stdout.Write(bytes)
	return nil
}
