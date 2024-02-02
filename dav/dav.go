package dav

import (
	"context"
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
		dav.cdClient.FindCalendarHomeSet(context.Background(), fmt.Sprintf("principals/%s", dav.username))
	if err != nil {
		return dav, err
	}

	dav.calendars, err = dav.cdClient.FindCalendars(context.Background(), dav.calendarHomeSet)

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

	dav.objects[path], err = dav.cdClient.QueryCalendar(context.Background(), path, query)
	if err != nil {
		return err
	}

	return nil
}

func (dav *DAV) GetEventsInCalendar(path string) []ical.Event {
	var events []ical.Event

	if objs, ok := dav.objects[path]; ok {
		for i := 0; i < len(objs); i++ {
			if len(objs[i].Data.Component.Children) > 0 {
				fmt.Printf("%v\n\n", objs[i].Data.Component.Children[0].Name)
			}
			for _, ev := range objs[i].Data.Events() {
				events = append(events, ev)
			}
		}
	}

	return events
}

// "CATEGORIES"
// --> "マリウス"
// "COMPLETED"
// --> "20240114T141521Z"
// "CREATED"
// --> "20240112T185004Z"
// "DTSTAMP"
// --> "20240114T141731Z"
// "DUE"
// --> ""
// "TZID"
// --> "20240112T200001"
// "GEO"
// --> "5.01084248643789;-69.48780557866111"
// "LAST-MODIFIED"
// --> "20240114T141652Z"
// "PERCENT-COMPLETE"
// --> "100"
// "PRIORITY"
// --> "9"
// "SEQUENCE"
// --> "1"
// "STATUS"
// --> "COMPLETED"
// "SUMMARY"
// --> "Take photos of this and that"
// "UID"
// --> "3807009628322705224"
// "X-APPLE-SORT-ORDER"
// --> "713117750"
func (dav *DAV) GetTodosInCalendar(path string) []ical.Component {
	var todos []ical.Component

	fmt.Printf("%s\n", path)
	if _, ok := dav.objects[path]; ok {
		for i := 0; i < len(dav.objects[path]); i++ {
			for j := 0; j < len(dav.objects[path][i].Data.Component.Children); j++ {
				if dav.objects[path][i].Data.Component.Children[j].Name == "VTODO" {
					fmt.Printf("%#v\n\n", dav.objects[path][i].Data.Component.Children[j].Props)
					todos = append(todos, *dav.objects[path][i].Data.Component.Children[j])
				}
			}
		}
	}

	return todos
}
