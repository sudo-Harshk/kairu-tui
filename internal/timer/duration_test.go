package timer

import (
	"testing"
)

func TestParseDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		wantSeconds int
		wantErr     bool
	}{
		{name: "minutes", input: "25", wantSeconds: 25 * 60},
		{name: "hoursMinutes", input: "1:00", wantSeconds: 60 * 60},
		{name: "zeroHoursMinutes", input: "0:30", wantSeconds: 30 * 60},
		{name: "trimmed", input: "  5  ", wantSeconds: 5 * 60},
		{name: "empty", input: "", wantErr: true},
		{name: "zeroMinutes", input: "0", wantErr: true},
		{name: "negativeMinutes", input: "-5", wantErr: true},
		{name: "invalidMinutes", input: "1:60", wantErr: true},
		{name: "invalidFormat", input: "1:2:3", wantErr: true},
		{name: "notNumber", input: "abc", wantErr: true},
		{name: "negativePart", input: "1:-1", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDurationInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (input=%q)", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error (input=%q): %v", tc.input, err)
			}
			if got != tc.wantSeconds {
				t.Fatalf("got %d seconds, want %d (input=%q)", got, tc.wantSeconds, tc.input)
			}
		})
	}
}

func TestFormatDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seconds int
		want    string
	}{
		{seconds: 0, want: "0"},
		{seconds: -10, want: "0"},
		{seconds: 60, want: "1"},
		{seconds: 600, want: "10"},
		{seconds: 3600, want: "1:00"},
		{seconds: 3660, want: "1:01"},
		{seconds: 7320, want: "2:02"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			got := FormatDurationInput(tc.seconds)
			if got != tc.want {
				t.Fatalf("got %q, want %q (seconds=%d)", got, tc.want, tc.seconds)
			}
		})
	}
}
