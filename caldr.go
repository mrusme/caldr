package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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

	// args := flag.Args()

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

	// var t *template.Template
	// if len(caldrTmpl) > 0 && outputJson == false {
	// 	t = template.Must(template.New("caldr").Funcs(template.FuncMap{
	// 		"RenderPhoto": func(photo string) string {
	// 			return RenderPhoto(photo)
	// 		},
	// 		"RenderAddress": func(address string) string {
	// 			return RenderAddress(address)
	// 		},
	// 		"RenderBirthdate": func(dt string) string {
	// 			return RenderBirthdate(dt)
	// 		},
	// 	}).ParseFiles(caldrTmpl))
	// }

	// var foundIcs []icard.Card
	// var foundBdays []time.Time
	// var today time.Time = time.Now()
	//
	foundIcs, err := db.List()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	for _, ic := range foundIcs {
		// 	photo := ic.PreferredValue(icard.FieldPhoto)
		// 	photoRender := RenderPhoto(photo)
		//
		if outputJson == true {
			b, err := json.MarshalIndent(ic, "", "  ")
			if err != nil {
				fmt.Printf("%s\n", err)
				os.Exit(1)
			}

			fmt.Printf(string(b))
		} else {
			// 		if len(caldrTmpl) > 0 {
			// 			err := t.ExecuteTemplate(os.Stdout, path.Base(caldrTmpl), ic)
			// 			if err != nil {
			// 				fmt.Printf("%s\n", err)
			// 				os.Exit(1)
			// 			}
			// 		} else {
			fmt.Printf("%+v\n", ic.Props.Get(ical.PropSummary).Value)
			// 		}
		}
	}

	os.Exit(0)
}
