package model

type PluginInfo struct {
	Name       string `json:"name"`
	IsAdded    bool   `json:"isAdded"`
	IsOfficial bool   `json:"isOfficial"`
	URL        string `json:"url"`
}
