package app

import "testing"

func TestIsPluginNotInstalledSearchError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "plugin missing",
			message: "command failed: exit status 1, output: nodejs not supported, error: Plugin nodejs is not installed. Use the -y flag",
			want:    true,
		},
		{
			name:    "network failure",
			message: "command failed: output: get plugin index error: EOF",
			want:    false,
		},
		{
			name:    "missing without plugin word",
			message: "nodejs is not installed",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPluginNotInstalledSearchError(tt.message); got != tt.want {
				t.Fatalf("isPluginNotInstalledSearchError(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}
