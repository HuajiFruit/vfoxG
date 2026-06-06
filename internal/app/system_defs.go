package app

type systemSDKDef struct {
	Name    string
	Exe     string
	VerArgs []string
}

var systemSDKDefs = []systemSDKDef{
	{"python", "python", []string{"--version"}},
	{"nodejs", "node", []string{"--version"}},
	{"java", "java", []string{"-version"}},
	{"golang", "go", []string{"version"}},
	{"rust", "rustc", []string{"--version"}},
	{"dotnet", "dotnet", []string{"--version"}},
	{"ruby", "ruby", []string{"--version"}},
	{"php", "php", []string{"--version"}},
	{"perl", "perl", []string{"--version"}},
	{"git", "git", []string{"--version"}},
	{"docker", "docker", []string{"--version"}},
	{"zig", "zig", []string{"version"}},
	{"lua", "lua", []string{"-v"}},
	{"gcc", "gcc", []string{"--version"}},
	{"cmake", "cmake", []string{"--version"}},
}
