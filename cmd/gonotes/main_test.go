package main

import (
	"strings"
	"testing"
)

func TestRunUpdateSelectorValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "no selector",
			args: []string{"-d", "2026-04-01 10:00:00"},
			want: "exactly one target selector",
		},
		{
			name: "multiple selectors",
			args: []string{"-i", "20260328-1", "-f", "20260328-1-note.md", "-d", "2026-04-01 10:00:00"},
			want: "exactly one target selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runUpdate(tt.args)
			if err == nil {
				t.Fatalf("runUpdate(%v) err = <nil>, want error", tt.args)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("runUpdate(%v) err = %q, want substring %q", tt.args, err.Error(), tt.want)
			}
		})
	}
}

func TestRunUpdateRejectsTitleWithAll(t *testing.T) {
	err := runUpdate([]string{"-a", "-t", "Renamed"})
	if err == nil {
		t.Fatal("runUpdate() err = <nil>, want error")
	}
	if !strings.Contains(err.Error(), "-t is not allowed with -a") {
		t.Fatalf("runUpdate() err = %q, want -a/-t validation", err.Error())
	}
}

func TestRunUpdateRejectsTagRewriteMismatch(t *testing.T) {
	err := runUpdate([]string{"-i", "20260328-1", "-tm", "^foo", "-d", "2026-04-01 10:00:00"})
	if err == nil {
		t.Fatal("runUpdate() err = <nil>, want error")
	}
	if !strings.Contains(err.Error(), "-tm and -tr must be provided in equal counts") {
		t.Fatalf("runUpdate() err = %q, want -tm/-tr mismatch error", err.Error())
	}
}

func TestRunUpdateRejectsTagsAndRewritesTogether(t *testing.T) {
	err := runUpdate([]string{"-i", "20260328-1", "-T", "foo", "-tm", "^foo$", "-tr", "bar"})
	if err == nil {
		t.Fatal("runUpdate() err = <nil>, want error")
	}
	if !strings.Contains(err.Error(), "cannot combine -T with -tm/-tr") {
		t.Fatalf("runUpdate() err = %q, want -T/-tm conflict error", err.Error())
	}
}

func TestRunUpdateRequiresMutation(t *testing.T) {
	err := runUpdate([]string{"-i", "20260328-1"})
	if err == nil {
		t.Fatal("runUpdate() err = <nil>, want error")
	}
	if !strings.Contains(err.Error(), "at least one mutation is required") {
		t.Fatalf("runUpdate() err = %q, want missing mutation error", err.Error())
	}
}

func TestRunUpdateRejectsUnknownOutputFormat(t *testing.T) {
	err := runUpdate([]string{"-i", "20260328-1", "-d", "2026-04-01 10:00:00", "-n", "-o", "yaml"})
	if err == nil {
		t.Fatal("runUpdate() err = <nil>, want error")
	}
	if !strings.Contains(err.Error(), "unknown output format") {
		t.Fatalf("runUpdate() err = %q, want unknown output format error", err.Error())
	}
}
