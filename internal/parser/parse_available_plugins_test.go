package parser

import "testing"

func TestParseAvailablePluginsOutput(t *testing.T) {
	out := `AVAILABLE PLUGINS
python ✓ https://github.com/version-fox/vfox-python
nodejs √ https://github.com/version-fox/vfox-nodejs
Use 'vfox add <plugin>' to add a plugin
`

	got := AvailablePlugins(out)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Name != "python" || !got[0].IsOfficial || got[0].URL == "" {
		t.Fatalf("unexpected first plugin: %+v", got[0])
	}
	if got[1].Name != "nodejs" || !got[1].IsOfficial || got[1].URL == "" {
		t.Fatalf("unexpected second plugin: %+v", got[1])
	}
}

func TestIsOfficialPluginStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "✓", want: true},
		{status: "√", want: true},
		{status: "yes", want: true},
		{status: "true", want: true},
		{status: "✗", want: false},
		{status: "x", want: false},
		{status: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isOfficialPluginStatus(tt.status); got != tt.want {
				t.Fatalf("isOfficialPluginStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
