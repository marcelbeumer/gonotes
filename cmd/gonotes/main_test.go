package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempCWD(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() err = %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir(tmp) err = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("Chdir(old) err = %v", err)
		}
	})
	return tmp
}

func TestRunNewValidations(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "stdin and file conflict",
			args: []string{"-", "-f", "draft.md"},
			want: "cannot use both stdin (-) and -f",
		},
		{
			name: "frontmatter key-value pair mismatch",
			args: []string{"-Fk", "href"},
			want: "-Fk and -Fv must be provided in equal counts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runNew(tt.args)
			if err == nil {
				t.Fatalf("runNew(%v) err = <nil>, want error", tt.args)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("runNew(%v) err = %q, want substring %q", tt.args, err.Error(), tt.want)
			}
		})
	}
}

func TestRunNewDryRunInEmptyWorkspace(t *testing.T) {
	tmp := withTempCWD(t)
	idDir := filepath.Join(tmp, "notes", "by", "id")
	if err := os.MkdirAll(idDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() err = %v", err)
	}

	err := runNew([]string{"-n", "-t", "Test Note"})
	if err != nil {
		t.Fatalf("runNew() err = %v", err)
	}
}
