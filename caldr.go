package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/mrusme/caldr/dav"
	"github.com/mrusme/caldr/store"
)

var username string
var password string
var endpoint string
var caldrDb string
var caldrTmpl string

var taskdLaunch bool
var taskdPort int
var taskdCertFile string
var taskdKeyFile string

var doRefresh bool
var outputJson bool

func setFlags() []string {
	flag.StringVar(
		&username,
		"carddav-username",
		os.Getenv("CARDDAV_USERNAME"),
		"CardDAV username (HTTP Basic Auth)",
	)
	flag.StringVar(
		&password,
		"carddav-password",
		os.Getenv("CARDDAV_PASSWORD"),
		"CardDAV password (HTTP Basic Auth)",
	)
	flag.StringVar(
		&endpoint,
		"carddav-endpoint",
		os.Getenv("CARDDAV_ENDPOINT"),
		"CardDAV endpoint (HTTP(S) URL)",
	)
	flag.StringVar(
		&caldrDb,
		"database",
		os.Getenv("CALDR_DB"),
		"Local icard database",
	)
	flag.StringVar(
		&caldrTmpl,
		"template",
		os.Getenv("CALDR_TEMPLATE"),
		"caldr template file",
	)

	flag.BoolVar(
		&taskdLaunch,
		"t",
		false,
		"Launch Taskd",
	)
	taskdPortEnv, _ := strconv.Atoi(os.Getenv("CALDR_TASKD_PORT"))
	flag.IntVar(
		&taskdPort,
		"taskd-port",
		taskdPortEnv,
		"Taskd port",
	)
	flag.StringVar(
		&taskdCertFile,
		"taskd-cert-file",
		os.Getenv("CALDR_TASKD_CERT_FILE"),
		"Taskd cert file",
	)
	flag.StringVar(
		&taskdKeyFile,
		"taskd-key-file",
		os.Getenv("CALDR_TASKD_KEY_FILE"),
		"Taskd key file",
	)

	flag.BoolVar(
		&doRefresh,
		"r",
		false,
		"Refresh local icard database",
	)
	flag.BoolVar(
		&outputJson,
		"j",
		false,
		"Output JSON",
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"usage: %s [flags] [query]\n\n",
			os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(),
			"Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(),
			"\nQuery:\n")
		fmt.Fprintf(flag.CommandLine.Output(),
			"  today\t\t\tShow today's entries\n")
		fmt.Fprintf(flag.CommandLine.Output(),
			"  tomorrow\t\tShow tomorrow's entries\n")
		fmt.Fprintf(flag.CommandLine.Output(),
			"  in 3 days\t\tShow entries in 3 days\n")
		fmt.Fprintf(flag.CommandLine.Output(),
			"  in 2 months\t\tShow entries in 2 months\n")
		fmt.Fprintf(flag.CommandLine.Output(),
			"  next 5 days\t\tShow entries in the next 5 days\n")
	}

	flag.Parse()

	return flag.Args()
}

func main() {
	var err error

	args := setFlags()

	db, err := store.Open(caldrDb)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if doRefresh == true {
		if err := refresh(db); err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
	}

	if taskdLaunch == true {
		if err := runTaskd(db); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	} else {
		if err := runCLI(db, args); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}

	os.Exit(0)
}

func refresh(db *store.Store) error {
	cd, err := dav.New(endpoint, username, password)
	if err != nil {
		return err
	}

	err = cd.RefreshCalendars()
	if err != nil {
		return err
	}

	paths := cd.GetAddressBookPaths()
	ics := cd.GetEventsInCalendar(paths[0])
	err = db.UpsertEvents(ics)
	if err != nil {
		return err
	}

	todos := cd.GetTodosInCalendar(paths[0])
	err = db.UpsertTodos(todos)
	if err != nil {
		return err
	}

	return nil
}
