package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/araddon/dateparse"
	"github.com/emersion/go-ical"
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
		t = template.Must(template.New("caldr").Funcs(template.FuncMap{
			"GetSummary": func(ic ical.Event) string {
				return GetPropValueSafe(&ic, ical.PropSummary)
			},
			"GetDescription": func(ic ical.Event) string {
				rplcr := strings.NewReplacer("\\n", "\n", "\\,", ",")
				return rplcr.Replace(
					GetPropValueSafe(&ic, ical.PropDescription),
				)
			},
			"GetDateTimeStart": func(ic ical.Event, frmt string) string {
				val := GetPropValueSafe(&ic, ical.PropDateTimeStart)
				return ParseDateTime(val).Format(frmt)
			},
			"GetDateTimeEnd": func(ic ical.Event, frmt string) string {
				val := GetPropValueSafe(&ic, ical.PropDateTimeEnd)
				return ParseDateTime(val).Format(frmt)
			},
			"GetDateTimeStamp": func(ic ical.Event, frmt string) string {
				val := GetPropValueSafe(&ic, ical.PropDateTimeStamp)
				return ParseDateTime(val).Format(frmt)
			},
			"GetURL": func(ic ical.Event) string {
				return GetPropValueSafe(&ic, ical.PropURL)
			},
		}).ParseFiles(caldrTmpl))
	}

	var today time.Time = time.Now()
	var displayDate time.Time

	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "today":
			displayDate = today
		case "tomorrow":
			displayDate = today.AddDate(0, 0, 1)
		case "in":
			if len(args) == 3 {
				i, err := strconv.Atoi(args[1])
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				switch strings.ToLower(args[2]) {
				case "day", "days":
					displayDate = today.AddDate(0, 0, i)
				case "week", "weeks":
					displayDate = today.AddDate(0, 0, i*7)
				case "month", "months":
					displayDate = today.AddDate(0, i, 0)
				case "year", "years":
					displayDate = today.AddDate(i, 0, 0)
				}
			}
		}
	}

	foundIcs, err := db.List()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	for _, ic := range foundIcs {
		if !displayDate.IsZero() {
			val := GetPropValueSafe(&ic, ical.PropDateTimeStart)
			dt := ParseDateTime(val)

			if !dt.Truncate(24 * time.Hour).
				Equal(displayDate.Truncate(24 * time.Hour)) {
				continue
			}
		}

		if outputJson == true {
			b, err := json.MarshalIndent(ic, "", "  ")
			if err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}

			fmt.Printf(string(b))
		} else {
			if len(caldrTmpl) > 0 {
				err := t.ExecuteTemplate(os.Stdout, path.Base(caldrTmpl), ic)
				if err != nil {
					fmt.Printf("%s\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("%+v\n", ic.Props.Get(ical.PropSummary).Value)
			}
		}
	}

	os.Exit(0)
}

func GetPropValueSafe(ic *ical.Event, propName string) string {
	prop := ic.Props.Get(propName)
	if prop == nil {
		return ""
	}
	return prop.Value
}

func ParseDateTime(val string) time.Time {
	// Quick fix because PRs to dateparse are pointless:
	// https://github.com/araddon/dateparse/pulls?q=is%3Aopen+is%3Apr
	dtf := regexp.MustCompile(
		`([0-9]{4})([0-9]{2})([0-9]{2})T([0-9]{2})([0-9]{2})([0-9]{2})`)
	val = dtf.ReplaceAllString(val, "$1-$2-$3 $4:$5:$6")

	if dt, err := dateparse.ParseAny(val); err == nil {
		return dt
	}
	return time.Time{}
}
