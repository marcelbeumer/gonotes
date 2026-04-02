package gonotes

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  Leading and trailing  ", "leading-and-trailing"},
		{"Special!@#$%Characters", "special-characters"},
		{"multiple---dashes", "multiple-dashes"},
		{"UPPER CASE", "upper-case"},
		{"already-good", "already-good"},
		{"foo/bar baz", "foo-bar-baz"},
		{"---leading-dashes---", "leading-dashes"},
		{"", ""},
		{"a", "a"},
		{"Hello   World", "hello-world"},
		{"café résumé", "caf-r-sum"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
