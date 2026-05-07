package main

import (
	"fmt"

	ics "github.com/arran4/golang-ical"
)

func buildCalendar(l []Lesson, name string) string {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetName(name)
	for i, lesson := range l {
		e := cal.AddEvent(fmt.Sprintf("bakalari-%d-%d", lesson.Start.Unix(), i))
		e.SetStartAt(lesson.Start)
		e.SetEndAt(lesson.End)

		summary := lesson.Name
		description := fmt.Sprintf("Vyučující: %s", lesson.Teacher)

		if lesson.Room != "" {
			e.SetLocation(lesson.Room)
		}

		if lesson.ChangeText != "" {
			summary = "Změna: " + lesson.Name
			description += fmt.Sprintf("\nZměna: %s", lesson.ChangeText)
		}

		e.SetSummary(summary)
		e.SetDescription(description)
	}
	return cal.Serialize()
}
