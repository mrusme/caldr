package dav

import (
	"fmt"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

type DAV struct {
	httpClient webdav.HTTPClient
	cdClient   *caldav.Client

	endpoint string
	username string
	password string

	calendarHomeSet string
	calendars       []caldav.Calendar

	objects map[string][]caldav.CalendarObject
}

func New(endpoint, username, password string) (*DAV, error) {
	var err error

	dav := new(DAV)
	dav.objects = make(map[string][]caldav.CalendarObject)

	dav.endpoint = endpoint
	dav.username = username
	dav.password = password

	dav.httpClient = webdav.HTTPClientWithBasicAuth(nil, dav.username, dav.password)
	dav.cdClient, err = caldav.NewClient(dav.httpClient, dav.endpoint)
	if err != nil {
		return nil, err
	}

	dav.calendarHomeSet, err =
		dav.cdClient.FindCalendarHomeSet(fmt.Sprintf("principals/%s", dav.username))
	if err != nil {
		return dav, err
	}

	dav.calendars, err = dav.cdClient.FindCalendars(dav.calendarHomeSet)

	return dav, nil
}

func (dav *DAV) GetAddressBookPaths() []string {
	var paths []string

	for _, ab := range dav.calendars {
		paths = append(paths, ab.Path)
	}

	return paths
}

func (dav *DAV) RefreshCalendars() error {
	for _, ab := range dav.calendars {
		err := dav.RefreshCalendar(ab.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dav *DAV) RefreshCalendar(path string) error {
	var err error
	query := new(caldav.CalendarQuery)
	query.CompRequest = caldav.CalendarCompRequest{
		Props: []string{
			ical.PropUID,
		},
		AllComps: true,
	}
	// query.Limit = 10

	dav.objects[path], err = dav.cdClient.QueryCalendar(path, query)
	if err != nil {
		return err
	}

	return nil
}

func (dav *DAV) GetEventsInCalendar(path string) []ical.Event {
	var events []ical.Event

	if objs, ok := dav.objects[path]; ok {
		for i := 0; i < len(objs); i++ {
			for _, ev := range objs[i].Data.Events() {
				events = append(events, ev)
			}
		}
	}

	return events
}
