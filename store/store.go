package store

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/araddon/dateparse"
	"github.com/emersion/go-ical"
	"github.com/teambition/rrule-go"
	"github.com/tidwall/buntdb"
)

type CalEvent struct {
	Name        string
	Description string
	StartsAt    time.Time
	EndsAt      time.Time
}

type Store struct {
	db *buntdb.DB
}

func Open(path string) (*Store, error) {
	var err error
	s := new(Store)

	s.db, err = buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	s.db.CreateIndex(ical.PropDateTimeStart, "*",
		buntdb.IndexJSON(fmt.Sprintf("Props.%s", ical.PropDateTimeStart)),
	)

	return s, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) Upsert(ics []ical.Event) error {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		for _, ic := range ics {
			mic, err := json.Marshal(ic)
			if err != nil {
				return err
			}
			tx.Set(ic.Props.Get(ical.PropUID).Value, string(mic), nil)
		}
		return nil
	})
	return err
}

func (s *Store) List(startT, endT time.Time) ([]CalEvent, error) {
	var calEvents []CalEvent

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(ical.PropDateTimeStart, func(k, v string) bool {
			var ic ical.Event
			if json.Unmarshal([]byte(v), &ic) == nil {
				recurrence := GetPropValueSafe(&ic, ical.PropRecurrenceRule)
				if recurrence != "" {
					rr, err := rrule.StrToRRule(recurrence)
					if err == nil {
						rr.DTStart(ParseDateTime(GetPropValueSafe(&ic, ical.PropDateTimeStart)))
						rr.Until(endT)
						for _, t := range rr.All() {
							if t.After(startT) && t.Before(endT) {
								calEvents = append(calEvents, CalEvent{
									Name: GetPropValueSafe(&ic, ical.PropSummary),
									Description: appendNewLine(
										GetPropValueSafe(&ic, ical.PropDescription),
									),
									StartsAt: t,
									EndsAt: ParseDateTime(
										GetPropValueSafe(&ic, ical.PropDateTimeEnd),
									),
								})
							}
						}
					}
				} else {
					t := ParseDateTime(GetPropValueSafe(&ic, ical.PropDateTimeStart))

					if t.After(startT) && t.Before(endT) {
						calEvents = append(calEvents, CalEvent{
							Name: GetPropValueSafe(&ic, ical.PropSummary),
							Description: appendNewLine(
								GetPropValueSafe(&ic, ical.PropDescription),
							),
							StartsAt: t,
							EndsAt: ParseDateTime(
								GetPropValueSafe(&ic, ical.PropDateTimeEnd),
							),
						})
					}
				}

			}
			return true
		})
	})

	return calEvents, err
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
		`([0-9]{4})([0-9]{2})([0-9]{2})T([0-9]{2})([0-9]{2})([0-9]{2})(Z){0,1}`)
	val = dtf.ReplaceAllString(val, "$1-$2-$3 $4:$5:$6")

	if dt, err := dateparse.ParseAny(val); err == nil {
		return dt
	}
	return time.Time{}
}

func appendNewLine(s string) string {
	str := s
	if s != "" && s[len(s)-1] != '\n' {
		str = s + "\n"
	}
	return str
}
