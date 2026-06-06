package main

type AppConfig struct {
	VfoxHome string `json:"vfoxHome"`
}

type DownloadPathInfo struct {
	Path              string `json:"path"`
	DefaultPath       string `json:"defaultPath"`
	IsDefault         bool   `json:"isDefault"`
	HasMigratableData bool   `json:"hasMigratableData"`
}
