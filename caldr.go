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

	"github.com/charmbracelet/lipgloss"
	"github.com/mrusme/caldr/dav"
	"github.com/mrusme/caldr/store"
	"github.com/mrusme/caldr/taskd"
)

func main() {
	var err error
	var username string
	var password string
	var endpoint string
	var caldrDb string
	var caldrTmpl string

	var refresh bool
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
		ics := cd.GetEventsInCalendar(paths[0])

		err = db.Upsert(ics)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
	}

	var t *template.Template
	if len(caldrTmpl) > 0 && outputJson == false {
		t = template.Must(template.New("caldr").Funcs(template.FuncMap{
			"Style": func() lipgloss.Style {
				return lipgloss.NewStyle()
			},
			"Color": func(color string) lipgloss.Color {
				return lipgloss.Color(color)
			},
			"SplitByDate": func(calEvents []store.CalEvent) map[string][]store.CalEvent {
				var byDate map[string][]store.CalEvent = make(map[string][]store.CalEvent)

				for i := 0; i < len(calEvents); i++ {
					date := calEvents[i].StartsAt.Format("2006-01-02")
					byDate[date] = append(byDate[date], calEvents[i])
				}

				return byDate
			},
		}).ParseFiles(caldrTmpl))
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
			for _, calEvent := range calEvents {
				fmt.Printf("%s: %s\n",
					calEvent.StartsAt.Format("2006-01-02 15:04:05"),
					calEvent.Name,
				)
			}
		}
	}

	td, err := taskd.New(4433, "crt.pem", "key.pem")
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = td.Launch()
	fmt.Print(err)
	fmt.Printf("%v", td)

	os.Exit(0)
}

func getStartEndByArgs(args []string) (time.Time, time.Time, error) {
	var today time.Time = time.Now()

	var sT time.Time
	var eT time.Time

	if len(args) > 0 {
		firstArg := strings.ToLower(args[0])
		switch firstArg {
		case "today":
			sT, eT = getStartEndForDate(today)
		case "tomorrow":
			sT, eT = getStartEndForDate(today.AddDate(0, 0, 1))
		case "in", "next":
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
				if firstArg == "next" {
					sT, _ = getStartEndForDate(today)
				}
			}
		}
	} else {
		sT, eT = getStartEndForDate(today)
	}

	return sT, eT, nil

}

func getStartEndForDate(t time.Time) (time.Time, time.Time) {
	y, m, d := t.Date()
	sT := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	eT := time.Date(y, m, d, 23, 59, 59, 999999999, t.Location())
	return sT, eT
}
