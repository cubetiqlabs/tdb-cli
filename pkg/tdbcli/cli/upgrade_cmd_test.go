package cli

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.1.9", 1},
		{"v0.9.0", "0.10.0", -1},
		{"1.0.0", "1.0.0-beta", 0},
		{"dev", "1.0.0", -1},
	}
	for _, tc := range cases {
		got, err := compareVersions(tc.cur, tc.latest)
		if err != nil {
			t.Fatalf("compareVersions(%q, %q) error: %v", tc.cur, tc.latest, err)
		}
		if got != tc.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tc.cur, tc.latest, got, tc.want)
		}
	}
}

func TestParseVersionParts(t *testing.T) {
	tests := map[string][3]int{
		"":           {0, 0, 0},
		"1":          {1, 0, 0},
		"1.2":        {1, 2, 0},
		"1.2.3":      {1, 2, 3},
		"v2.3.4":     {2, 3, 4},
		"1.2.3-beta": {1, 2, 3},
	}
	for input, want := range tests {
		got, err := parseVersionParts(input)
		if err != nil {
			t.Fatalf("parseVersionParts(%q) error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parseVersionParts(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestSanitizeVersion(t *testing.T) {
	cases := map[string]string{
		"v1.2.3":   "1.2.3",
		"V1.2.3":   "1.2.3",
		" 1.0 ":    "1.0",
		"dev":      "dev",
		"v":        "",
		"":         "",
		"v1.2.3+g": "1.2.3+g",
	}
	for in, want := range cases {
		if got := sanitizeVersion(in); got != want {
			t.Fatalf("sanitizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
