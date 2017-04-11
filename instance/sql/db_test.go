package sql

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/ContainerSolutions/flux"
	"github.com/ContainerSolutions/flux/db"
	"github.com/ContainerSolutions/flux/instance"
)

func newDB(t *testing.T) *DB {
	f, err := ioutil.TempFile("", "fluxy-testdb")
	if err != nil {
		t.Fatal(err)
	}
	dbsource := "file://" + f.Name()
	if _, err = db.Migrate(dbsource, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	db, err := New("ql", dbsource)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpdateOK(t *testing.T) {
	db := newDB(t)

	inst := flux.InstanceID("floaty-womble-abc123")
	service := flux.MakeServiceID("namespace", "service")
	services := map[flux.ServiceID]instance.ServiceConfig{}
	services[service] = instance.ServiceConfig{
		Automated: true,
		Locked:    true,
	}
	c := instance.Config{
		Services: services,
	}
	err := db.UpdateConfig(inst, func(_ instance.Config) (instance.Config, error) {
		return c, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	c1, err := db.GetConfig(inst)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := c1.Services[service]; !found {
		t.Fatalf("did not find instance config after setting")
	}
	if !c1.Services[service].Automated {
		t.Fatalf("expected service config %#v, got %#v", c.Services[service], c1.Services[service])
	}
	if !c1.Services[service].Locked {
		t.Fatalf("expected service config %#v, got %#v", c.Services[service], c1.Services[service])
	}
}

func TestUpdateRollback(t *testing.T) {
	inst := flux.InstanceID("floaty-womble-abc123")
	service := flux.MakeServiceID("namespace", "service")

	db := newDB(t)

	services := map[flux.ServiceID]instance.ServiceConfig{}
	services[service] = instance.ServiceConfig{
		Automated: true,
		Locked:    true,
	}
	c := instance.Config{
		Services: services,
	}

	err := db.UpdateConfig(inst, func(_ instance.Config) (instance.Config, error) {
		return c, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.UpdateConfig(inst, func(_ instance.Config) (instance.Config, error) {
		return instance.Config{}, errors.New("arbitrary fail")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	c1, err := db.GetConfig(inst)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := c1.Services[service]; !found {
		t.Fatalf("did not find instance config after setting")
	}
	if !c1.Services[service].Automated {
		t.Fatalf("expected service config %#v, got %#v", c.Services[service], c1.Services[service])
	}
	if !c1.Services[service].Locked {
		t.Fatalf("expected service config %#v, got %#v", c.Services[service], c1.Services[service])
	}
}
