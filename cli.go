package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mrusme/caldr/store"
)

func runCLI(db *store.Store, args []string) error {
	sT, eT, err := getStartEndByArgs(args)
	if err != nil {
		return err
	}

	calEvents, err := db.List(sT, eT)
	if err != nil {
		return err
	}

	sort.Slice(calEvents, func(i, j int) bool {
		return calEvents[i].StartsAt.Before(calEvents[j].StartsAt)
	})

	return output(calEvents)
}

func output(calEvents []store.CalEvent) error {
	var t *template.Template

	if outputJson == true {
		b, err := json.MarshalIndent(calEvents, "", "  ")
		if err != nil {
			return err
		}

		fmt.Printf(string(b))
	} else {
		if len(caldrTmpl) > 0 {
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
			err := t.ExecuteTemplate(os.Stdout, path.Base(caldrTmpl), calEvents)
			if err != nil {
				return err
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

	return nil
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
