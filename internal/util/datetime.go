package util

import (
	"fmt"
	"time"
)

const (
	baseFormat = "2006-01-02 15:04:05"
)

func ParseTime(timeStr string) (time.Time, error) {
	t1, e := time.Parse(baseFormat+" MST", timeStr+" UTC")
	if e != nil {
		return time.Time{}, fmt.Errorf("could not parse date: %v", e)
	}
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get location from date str: %s", timeStr)
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

func SerializeTime(time time.Time) string {
	return time.Format(baseFormat)
}
