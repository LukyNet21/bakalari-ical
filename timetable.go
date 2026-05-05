package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Timetable struct {
	Hours    []Hour    `json:"Hours"`
	Days     []Day     `json:"Days"`
	Classes  []Class   `json:"Classes"`
	Groups   []Group   `json:"Groups"`
	Subjects []Subject `json:"Subjects"`
	Teachers []Teacher `json:"Teachers"`
	Rooms    []Room    `json:"Rooms"`
	Cycles   []Cycle   `json:"Cycles"`
}

type Hour struct {
	Id        int    `json:"Id"`
	Caption   string `json:"Caption"`
	BeginTime string `json:"BeginTime"`
	EndTime   string `json:"EndTime"`
}

type Day struct {
	Atoms          []Atom    `json:"Atoms"`
	DayOfWeek      int       `json:"DayOfWeek"`
	Date           time.Time `json:"Date"`
	DayDescription string    `json:"DayDescription"`
	DayType        string    `json:"DayType"`
}

type Atom struct {
	HourId      int      `json:"HourId"`
	GroupIds    []string `json:"GroupIds"`
	SubjectId   *string  `json:"SubjectId"`
	TeacherId   *string  `json:"TeacherId"`
	RoomId      *string  `json:"RoomId"`
	CycleIds    []string `json:"CycleIds"`
	Change      *Change  `json:"Change"`
	HomeworkIds []string `json:"HomeworkIds"`
	Theme       *string  `json:"Theme"`
}

type Change struct {
	ChangeSubject *string   `json:"ChangeSubject"`
	Day           time.Time `json:"Day"`
	Hours         string    `json:"Hours"`
	ChangeType    string    `json:"ChangeType"`
	Description   string    `json:"Description"`
	Time          string    `json:"Time"`
	TypeAbbrev    *string   `json:"TypeAbbrev"`
	TypeName      *string   `json:"TypeName"`
}

type Class struct {
	Id     string `json:"Id"`
	Abbrev string `json:"Abbrev"`
	Name   string `json:"Name"`
}

type Group struct {
	ClassId string `json:"ClassId"`
	Id      string `json:"Id"`
	Abbrev  string `json:"Abbrev"`
	Name    string `json:"Name"`
}

type Subject struct {
	Id     string `json:"Id"`
	Abbrev string `json:"Abbrev"`
	Name   string `json:"Name"`
}

type Teacher struct {
	Id     string `json:"Id"`
	Abbrev string `json:"Abbrev"`
	Name   string `json:"Name"`
}

type Room struct {
	Id     string `json:"Id"`
	Abbrev string `json:"Abbrev"`
	Name   string `json:"Name"`
}

type Cycle struct {
	Id     string `json:"Id"`
	Abbrev string `json:"Abbrev"`
	Name   string `json:"Name"`
}

type Lesson struct {
	Start      time.Time
	End        time.Time
	Name       string
	Teacher    string
	Room       string
	ChangeText string
}

func getTimetable(baseURL, accessToken string, targetDate time.Time) (*Timetable, error) {
	timetableURL := fmt.Sprintf("%s/api/3/timetable/actual?date=%s", strings.TrimRight(baseURL, "/"), targetDate.Format("2006-01-02"))
	r, err := http.NewRequest("GET", timetableURL, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var decodedTimetable Timetable
	if err := json.NewDecoder(res.Body).Decode(&decodedTimetable); err != nil {
		return nil, err
	}
	return &decodedTimetable, nil
}

func combineDateAndTime(date time.Time, clock string) (time.Time, error) {
	parsedClock, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q: %w", clock, err)
	}

	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		parsedClock.Hour(),
		parsedClock.Minute(),
		0,
		0,
		date.Location(),
	), nil
}

func parseTimetable(t Timetable) ([]Lesson, error) {
	hours := make(map[int]Hour)
	for _, h := range t.Hours {
		hours[h.Id] = h
	}

	subjects := make(map[string]Subject)
	for _, s := range t.Subjects {
		subjects[s.Id] = s
	}

	teachers := make(map[string]Teacher)
	for _, teacher := range t.Teachers {
		teachers[teacher.Id] = teacher
	}

	rooms := make(map[string]Room)
	for _, r := range t.Rooms {
		rooms[r.Id] = r
	}

	var lessons []Lesson

	for _, day := range t.Days {
		for _, atom := range day.Atoms {
			hour, ok := hours[atom.HourId]
			if !ok {
				continue
			}

			start, err := combineDateAndTime(day.Date, hour.BeginTime)
			if err != nil {
				return nil, err
			}

			end, err := combineDateAndTime(day.Date, hour.EndTime)
			if err != nil {
				return nil, err
			}

			lesson := Lesson{
				Start: start,
				End:   end,
			}

			if atom.SubjectId != nil {
				if subject, ok := subjects[*atom.SubjectId]; ok {
					lesson.Name = subject.Name
				} else {
					lesson.Name = *atom.SubjectId
				}
			}

			if atom.TeacherId != nil {
				if teacher, ok := teachers[*atom.TeacherId]; ok {
					lesson.Teacher = teacher.Name
				} else {
					lesson.Teacher = *atom.TeacherId
				}
			}

			if atom.RoomId != nil {
				if room, ok := rooms[*atom.RoomId]; ok {
					lesson.Room = room.Abbrev
				} else {
					lesson.Room = *atom.RoomId
				}
			}

			if atom.Change != nil {
				switch atom.Change.ChangeType {
				case "Canceled", "Removed":
					continue

				case "Added":
					lesson.ChangeText = "Přidáno / změněno: " + atom.Change.Description

					if lesson.Name == "" {
						lesson.Name = "Přidaná hodina"
					} else {
						lesson.Name = "Přidáno: " + lesson.Name
					}

				case "RoomChanged":
					lesson.ChangeText = "Změna místnosti: " + atom.Change.Description

					if lesson.Name == "" {
						lesson.Name = "Změna místnosti"
					} else {
						lesson.Name = "Změna místnosti: " + lesson.Name
					}

				case "Substitution":
					lesson.ChangeText = "Suplování: " + atom.Change.Description

					if lesson.Name == "" {
						lesson.Name = "Suplování"
					} else {
						lesson.Name = "Suplování: " + lesson.Name
					}

				default:
					lesson.ChangeText = "Změna: " + atom.Change.Description

					if lesson.Name == "" {
						lesson.Name = "Změna v rozvrhu"
					} else {
						lesson.Name = "Změna: " + lesson.Name
					}
				}
			}

			if lesson.Name == "" && atom.Theme != nil {
				lesson.Name = *atom.Theme
			}

			lessons = append(lessons, lesson)
		}
	}

	return lessons, nil
}
