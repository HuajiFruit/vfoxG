package parser

import "testing"

func TestParseCurrentSdkVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want string
	}{
		{name: "plain arrow", out: "-> v3.14.4\n", want: "3.14.4"},
		{name: "sdk arrow", out: "python -> v3.13.12\n", want: "3.13.12"},
		{name: "sdk colon", out: "python: v3.12.10\n", want: "3.12.10"},
		{name: "sdk at", out: "python@v3.11.9\n", want: "3.11.9"},
		{name: "not installed", out: "python not supported, error: python not installed\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CurrentSdkVersion("python", tt.out); got != tt.want {
				t.Fatalf("CurrentSdkVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
