package model

type SdkVersion struct {
	Version string `json:"version"`
}

type SdkInfo struct {
	Name     string       `json:"name"`
	Source   string       `json:"source"` // "vfox" or "system"
	Path     string       `json:"path"`   // 绝对路径（主要针对系统 SDK，vfox SDK的路径动态获取）
	Versions []SdkVersion `json:"versions"`
}

type SdkVersionDetail struct {
	Version   string `json:"version"`
	IsCurrent bool   `json:"isCurrent"`
}

type SdkDetail struct {
	Name     string             `json:"name"`
	Current  string             `json:"current"`
	Versions []SdkVersionDetail `json:"versions"`
}
