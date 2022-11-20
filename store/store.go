package store

import (
	"encoding/json"
	"fmt"

	"github.com/emersion/go-ical"
	"github.com/tidwall/buntdb"
)

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

func (s *Store) List() ([]ical.Event, error) {
	var events []ical.Event

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend(ical.PropDateTimeStart, func(k, v string) bool {
			var ev ical.Event
			if json.Unmarshal([]byte(v), &ev) == nil {
				events = append(events, ev)
			}
			return true
		})
	})

	return events, err
}
