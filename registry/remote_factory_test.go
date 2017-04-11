package registry

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/kit/log"

	"github.com/ContainerSolutions/flux"
)

// Note: This actually goes off to docker hub to find the Image.
// It will fail if there is not internet connection
func TestRemoteFactory_CreateForDockerHub(t *testing.T) {
	// No credentials required for public Image
	fact := NewRemoteClientFactory(Credentials{}, log.NewNopLogger(), nil, time.Second)
	img, err := flux.ParseImage("alpine:latest", nil)
	testRepository = RepositoryFromImage(img)
	if err != nil {
		t.Fatal(err)
	}
	r, err := fact.CreateFor(testRepository.Host())
	if err != nil {
		t.Fatal(err)
	}
	res, err := r.Manifest(testRepository, img.Tag)
	if err != nil {
		t.Fatal(err)
	}
	expected := "index.docker.io/library/alpine:latest"
	if res.FullID() != expected {
		t.Fatal("Expected %q. Got %q", expected, res.FullID())
	}
}

func TestRemoteFactory_InvalidHost(t *testing.T) {
	fact := NewRemoteClientFactory(Credentials{}, log.NewNopLogger(), nil, time.Second)
	img, err := flux.ParseImage("invalid.host/library/alpine:latest", nil)
	if err != nil {
		t.Fatal(err)
	}
	testRepository = RepositoryFromImage(img)
	r, err := fact.CreateFor(testRepository.Host())
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Manifest(testRepository, img.Tag)
	if err == nil {
		t.Fatal("Expected error due to invalid host but got none.")
	}
}

func TestRemoteFactory_CredentialsFromConfig(t *testing.T) {
	user := "user"
	pass := "pass"
	host := "host"
	conf := flux.UnsafeInstanceConfig{
		Registry: flux.RegistryConfig{
			Auths: map[string]flux.Auth{
				host: {
					Auth: base64.StdEncoding.EncodeToString([]byte(user + ":" + pass)),
				},
			},
		},
	}
	creds, err := CredentialsFromConfig(conf)
	if err != nil {
		t.Fatal(err)
	}
	c := creds.credsFor(host)
	if user != c.username {
		t.Fatalf("Expected %q, got %q.", user, c.username)
	}
	if pass != c.password {
		t.Fatalf("Expected %q, got %q.", pass, c.password)
	}
	if len(creds.Hosts()) != 1 || host != creds.Hosts()[0] {
		t.Fatalf("Expected %q, got %q.", host, creds.Hosts()[0])
	}
}

func TestRemoteFactory_CredentialsFromConfigDecodeError(t *testing.T) {
	host := "host"
	conf := flux.UnsafeInstanceConfig{
		Registry: flux.RegistryConfig{
			Auths: map[string]flux.Auth{
				host: {
					Auth: "shouldnotbe:plaintext",
				},
			},
		},
	}
	_, err := CredentialsFromConfig(conf)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestRemoteFactory_CredentialsFromConfigHTTPSHosts(t *testing.T) {
	user := "user"
	pass := "pass"
	host := "host"
	httpsHost := fmt.Sprintf("https://%s/v1/", host)
	conf := flux.UnsafeInstanceConfig{
		Registry: flux.RegistryConfig{
			Auths: map[string]flux.Auth{
				httpsHost: {
					Auth: base64.StdEncoding.EncodeToString([]byte(user + ":" + pass)),
				},
			},
		},
	}
	creds, err := CredentialsFromConfig(conf)
	if err != nil {
		t.Fatal(err)
	}
	c := creds.credsFor(host)
	if user != c.username {
		t.Fatalf("Expected %q, got %q.", user, c.username)
	}
	if pass != c.password {
		t.Fatalf("Expected %q, got %q.", pass, c.password)
	}
	if len(creds.Hosts()) != 1 || httpsHost != creds.Hosts()[0] {
		t.Fatalf("Expected %q, got %q.", httpsHost, creds.Hosts()[0])
	}
}
