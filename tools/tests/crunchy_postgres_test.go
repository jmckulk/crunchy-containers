package tests

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	_ "github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/ory/dockertest"
)

var db *sql.DB
var db2 *sql.DB

//var pool *Pool

func init() {
	// Register 'pgx' as a driver
	stdlib.RegisterDriverConfig(&stdlib.DriverConfig{})
}

func TestMain(m *testing.M) {

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	network, err := pool.Client.CreateNetwork("test-on-start")
	//require.Nil(t, err)
	defer network.Close()

	envPrimary := []string{
		"PG_MODE=primary",
		"PG_PRIMARY_USER=postgres",
		"PG_PRIMARY_PASSWORD=yoursecurepassword",
		"PG_DATABASE=testdb",
		"PG_USER=yourusername",
		"PG_PASSWORD=yoursecurepassword",
		"PG_ROOT_PASSWORD=yoursecurepassword",
		"PG_PRIMARY_PORT=5432",
	}

	tag := "centos7-12.3-4.3.2"

	envReplica := []string{
		"PG_MODE=replica",
		"PG_PRIMARY_USER=postgres",
		"PG_PRIMARY_PASSWORD=yoursecurepassword",
		"PG_DATABASE=testdb",
		"PG_USER=yourusername",
		"PG_PASSWORD=yoursecurepassword",
		"PG_ROOT_PASSWORD=yoursecurepassword",
		"PG_PRIMARY_PORT=5432",
	}

	// pulls an image, creates a container based on it and runs it
	primaryContainer, err := pool.RunWithOptions(&RunOptions{
		Repository: "registry.developers.crunchydata.com/crunchydata/crunchy-postgres",
		Tag:        tag,
		Env:        envPrimary,
		Networks:   []*Network{network},
	})
	if err != nil {
		log.Fatalf("Could not start primaryContainer: %s", err)
	}
	//require.Nil(m, err)
	defer primaryContainer.Close()

	//err = primaryContainer.ConnectToNetwork(network)

	// pulls an image, creates a container based on it and runs it
	replicaContainer, err := pool.RunWithOptions(&RunOptions{
		Repository: "registry.developers.crunchydata.com/crunchydata/crunchy-postgres",
		Tag:        tag,
		Env:        envReplica,
		Networks:   []*Network{network},
	})
	//	primaryContainer, err := pool.RunWithOptions(optionsPrimary)
	if err != nil {
		log.Fatalf("Could not start replicaContainer: %s", err)
	}
	defer replicaContainer.Close()

	containerPortPrimary, _ := strconv.ParseUint(primaryContainer.GetPort("5432/tcp"), 10, 16)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		// db, err = sql.Open("pgx", fmt.Sprintf("yourusername:yoursecurepassword@(localhost:%s)/testdb", ))

		db, err = sql.Open("pgx",
			fmt.Sprintf("host=%s port=%d database=%s user=%s password=%s",
				"localhost", containerPortPrimary, "testdb",
				"yourusername", "yoursecurepassword",
			),
		)

		if err != nil {
			return err
		}

		return db.Ping()

	}); err != nil {
		log.Fatalf("Could not connect to docker1: %s", err)
	}

	/* 	// pulls an image, creates a container based on it and runs it
	   	replicaContainer, err := pool.RunWithOptions(optionsReplica)
	   	if err != nil {
	   		log.Fatalf("Could not start replicaContainer: %s", err)
	   	}

	   	err = replicaContainer.ConnectToNetwork(network)
	   	containerPortReplica, _ := strconv.ParseUint(replicaContainer.GetPort("5432/tcp"), 10, 16)

	   	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	   	if err := pool.Retry(func() error {
	   		var err error
	   		// db, err = sql.Open("pgx", fmt.Sprintf("yourusername:yoursecurepassword@(localhost:%s)/testdb", ))

	   		db2, err = sql.Open("pgx",
	   			fmt.Sprintf("host=%s port=%d database=%s user=%s password=%s",
	   				"localhost", containerPortReplica, "testdb",
	   				"yourusername", "yoursecurepassword",
	   			),
	   		)

	   		if err != nil {
	   			log.Fatalf("Could not connect to docker blah: %s", err)
	   			return err
	   		}

	   		return db2.Ping()
	   	}); err != nil {
	   		log.Fatalf("Could not connect to docker2: %s", err)
	   	} */

	logContainerInfo(primaryContainer)
	logContainerInfo(replicaContainer)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(primaryContainer); err != nil {
		log.Fatalf("Could not purge primaryContainer: %s", err)
	}

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(replicaContainer); err != nil {
		log.Fatalf("Could not purge replicaContainer: %s", err)
	}

	os.Exit(code)
}

// Simple helper function to give the user some information about the Docker containers we start
// up on their behalf.
func logContainerInfo(container *dockertest.Resource) {
	id := container.Container.ID
	image := container.Container.Name
	ports := container.Container.Config.ExposedPorts
	log.Printf("Started up %s (%s), listening on %s", image, id, ports)
}

func TestPostres(t *testing.T) {
	t.Log("Testing the 'crunchy-postgres' container...")
	var a int

	row := db.QueryRow("SELECT 1")

	if err := row.Scan(&a); err != nil {
		t.Fatalf("Could not scan: %s", err.Error())
	}

	if a != 1 {
		t.Fatalf("Expected 1, got: %d", a)
	}

	fmt.Println(a)
	fmt.Printf("%#v\n", a)

	extensions, err := AllExtensions()
	if err != nil {
		t.Fatal(err)
	}

	if len(extensions) < 1 {
		t.Fatalf("extensions less then 1, it shouldn't be: %d", len(extensions))
	}

	settings, err := Settings()
	if err != nil {
		t.Fatal(err)
	}

	for _, setting := range settings {
		if setting.Name == "log_timezone" && setting.Value != "UTC" {
			t.Fatalf("log_timezone isn't UTC, it should be: %s = %s", setting.Name, setting.Value)
		}
	}

	replicas, err := Replications()
	if err != nil {
		t.Fatal(err)
	}

	if len(replicas) < 1 {
		t.Fatalf("Replica count should be greater than 0: actual %d", len(replicas))
	}

	var sync bool
	for _, v := range replicas {
		if v.SyncState == "sync" {
			sync = true
		}
	}

	if sync {
		t.Fatalf("Sync replica detected, there shouldn't be.")
	}

}

// Extension is a data structure that holds information
// about extensions found in the database.
type Extension struct {
	DefaultVersion   string
	InstalledVersion string
	Name             string
}

func AllExtensions() ([]Extension, error) {
	statement := "SELECT name, default_version, installed_version " +
		" FROM pg_available_extensions"

	rows, err := db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	extensions := []Extension{}
	for rows.Next() {
		var name, defaultVersion, installedVersion sql.NullString
		if err := rows.Scan(&name, &defaultVersion, &installedVersion); err != nil {
			return nil, err
		}
		x := Extension{
			Name:             name.String,
			DefaultVersion:   defaultVersion.String,
			InstalledVersion: installedVersion.String,
		}
		//fmt.Println(x.Name)
		//fmt.Println(x.DefaultVersion)
		//fmt.Println(x.InstalledVersion)

		extensions = append(extensions, x)
	}
	return extensions, nil
}

// Setting is a data structure that holds the name and
// value of database settings.
type Setting struct {
	Name  string
	Value string
}

//
func Settings() ([]Setting, error) {
	statement := "SELECT name, setting FROM pg_settings"
	rows, err := db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []Setting{}
	for rows.Next() {
		var name, value sql.NullString
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		s := Setting{
			Name:  name.String,
			Value: value.String,
		}
		//fmt.Println(s.Name)
		//fmt.Println(s.Value)

		settings = append(settings, s)
	}
	return settings, nil
}

// Replication is a data structure that holds replication
// state queried from the primary database.
type Replication struct {
	Name      string
	State     string
	SyncState string
}

// Replication returns the state of replicas from the primary.
func Replications() ([]Replication, error) {
	statement := "SELECT application_name, state, " +
		"sync_state FROM pg_catalog.pg_stat_replication"

	rows, err := db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	replication := []Replication{}
	for rows.Next() {
		var name, state, syncState sql.NullString
		if err := rows.Scan(&name, &state, &syncState); err != nil {
			return nil, err
		}
		r := Replication{
			Name:      name.String,
			State:     state.String,
			SyncState: syncState.String,
		}
		replication = append(replication, r)
	}
	return replication, nil
}
