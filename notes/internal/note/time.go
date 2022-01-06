package note

import (
	"errors"
	"fmt"
	"time"
)

const (
	baseFormat = "2006-01-02 15:04:05"
)

func parseTime(dateStr string) (time.Time, error) {
	t1, e := time.Parse(baseFormat+" MST", dateStr+" UTC")
	if e != nil {
		return time.Time{},
			errors.New(fmt.Sprintf("Could not parse date: %v", e))
	}
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{},
			errors.New(fmt.Sprintf("Could not get location from date str: %s", dateStr))
	}
	t2 := time.Date(
		t1.Year(),
		t1.Month(),
		t1.Day(),
		t1.Hour(),
		t1.Minute(),
		t1.Second(),
		t1.Nanosecond(),
		loc,
	)
	return t2, nil
}

func serializeTime(time *time.Time) string {
	if time == nil {
		return ""
	}
	return time.Format(baseFormat)
}
