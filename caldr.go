package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mrusme/caldr/dav"
	"github.com/mrusme/caldr/store"
)

func main() {
	var err error
	var username string
	var password string
	var endpoint string
	var caldrDb string
	var caldrTmpl string

	var refresh bool
	var birthdays bool
	var outputJson bool

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
		&refresh,
		"r",
		false,
		"Refresh local icard database",
	)
	flag.BoolVar(
		&birthdays,
		"birthdays",
		false,
		"List contacts that have their birthday today",
	)
	flag.BoolVar(
		&outputJson,
		"j",
		false,
		"Output JSON",
	)

	flag.Parse()

	args := flag.Args()

	db, err := store.Open(caldrDb)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if refresh == true {
		cd, err := dav.New(endpoint, username, password)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		err = cd.RefreshCalendars()
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		paths := cd.GetAddressBookPaths()
		fmt.Println(paths)
		ics := cd.GetEventsInCalendar(paths[0])

		err = db.Upsert(ics)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
	}

	var t *template.Template
	if len(caldrTmpl) > 0 && outputJson == false {
		t = template.Must(template.New("caldr").Funcs(template.FuncMap{}).ParseFiles(caldrTmpl))
	}

	sT, eT, err := getStartEndByArgs(args)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	calEvents, err := db.List(sT, eT)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	sort.Slice(calEvents, func(i, j int) bool {
		return calEvents[i].StartsAt.Before(calEvents[j].StartsAt)
	})

	if outputJson == true {
		b, err := json.MarshalIndent(calEvents, "", "  ")
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}

		fmt.Printf(string(b))
	} else {
		if len(caldrTmpl) > 0 {
			err := t.ExecuteTemplate(os.Stdout, path.Base(caldrTmpl), calEvents)
			if err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}
		} else {
			// fmt.Printf("%+v\n", ic.Props.Get(ical.PropSummary).Value)
		}
	}

	os.Exit(0)
}

func getStartEndByArgs(args []string) (time.Time, time.Time, error) {
	var today time.Time = time.Now()

	var sT time.Time
	var eT time.Time

	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "today":
			sT, eT = getStartEndForDate(today)
		case "tomorrow":
			sT, eT = getStartEndForDate(today.AddDate(0, 0, 1))
		case "in":
			if len(args) == 3 {
				i, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				switch strings.ToLower(args[2]) {
				case "day", "days":
					sT, eT = getStartEndForDate(today.AddDate(0, 0, i))
				case "week", "weeks":
					sT, eT = getStartEndForDate(today.AddDate(0, 0, i*7))
				case "month", "months":
					sT, eT = getStartEndForDate(today.AddDate(0, i, 0))
				case "year", "years":
					sT, eT = getStartEndForDate(today.AddDate(i, 0, 0))
				}
			}
		}
	} else {
		sT = today
		eT = today.AddDate(0, 3, 0)
	}

	return sT, eT, nil

}

func getStartEndForDate(t time.Time) (time.Time, time.Time) {
	y, m, d := t.Date()
	sT := time.Date(y, m, d, 0, 0, 0, 0, t.UTC().Location())
	eT := time.Date(y, m, d, 23, 59, 59, 0, t.UTC().Location())

	return sT, eT
}
