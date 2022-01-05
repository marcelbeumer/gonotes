package note

import (
	"errors"
	"fmt"
	"time"
)

func ParseTime(dateStr string) (time.Time, error) {
	t1, e := time.Parse("2006-01-02 15:04:05 MST", dateStr+" UTC")
	if e != nil {
		return time.Time{},
			errors.New(fmt.Sprintf("Could not parse date: %v", e))
	}
	loc, _ := time.LoadLocation("Europe/Berlin")
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
