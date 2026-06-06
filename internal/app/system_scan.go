package app

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

func (a *App) scanSystemSdks() {
	a.loadSystemSdkCacheSnapshot()
	cleanEnv := a.systemSdkScanEnv()
	result := a.scanSystemSdkDefinitions(cleanEnv)

	result = a.filterSystemSdks(result)
	setSystemSdkCache(result)
	_ = a.writeJSONFile(a.getSystemSdkCacheFile(), result)
	a.emitEvent("system-sdks-ready")
}

func (a *App) loadSystemSdkCacheSnapshot() {
	data, err := os.ReadFile(a.getSystemSdkCacheFile())
	if err != nil {
		return
	}
	var cachedResult []SdkInfo
	if err := json.Unmarshal(data, &cachedResult); err != nil || len(cachedResult) == 0 {
		return
	}
	cachedResult = a.filterSystemSdks(cachedResult)
	setSystemSdkCache(cachedResult)
	if len(cachedResult) > 0 {
		a.emitEvent("system-sdks-ready")
	}
}

func (a *App) systemSdkScanEnv() []string {
	// Keep the cleaned PATH local to child processes; os.Setenv would race with vfox commands.
	originalPath := os.Getenv("PATH")
	cleanPath := cleanPathValue(originalPath, a.vfoxManagedPathRoots())

	baseEnv := os.Environ()
	var cleanEnv []string
	for _, envValue := range baseEnv {
		if strings.HasPrefix(strings.ToLower(envValue), "path=") {
			cleanEnv = append(cleanEnv, "PATH="+cleanPath)
		} else {
			cleanEnv = append(cleanEnv, envValue)
		}
	}
	return cleanEnv
}

func (a *App) scanSystemSdkDefinitions(cleanEnv []string) []SdkInfo {
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make([]SdkInfo, 0, len(systemSDKDefs))

	for _, def := range systemSDKDefs {
		wg.Add(1)
		go func(d systemSDKDef) {
			defer wg.Done()
			for _, exePath := range findExecutableCandidates(d.Exe, cleanEnv) {
				if exePath == "" || a.isVfoxManagedPath(exePath) {
					continue
				}
				version := a.tryGetVersionWithEnv(exePath, d.VerArgs, cleanEnv)
				if version == "" || !isUsableSystemVersion(version) {
					continue
				}
				mu.Lock()
				result = append(result, SdkInfo{
					Name:     d.Name,
					Source:   "system",
					Path:     exePath,
					Versions: []SdkVersion{{Version: version}},
				})
				mu.Unlock()
				return
			}
		}(def)
	}
	wg.Wait()
	return result
}

func (a *App) filterSystemSdks(sdks []SdkInfo) []SdkInfo {
	result := make([]SdkInfo, 0, len(sdks))
	for _, sdk := range sdks {
		if a.isVfoxManagedPath(sdk.Path) {
			continue
		}
		versions := make([]SdkVersion, 0, len(sdk.Versions))
		for _, version := range sdk.Versions {
			if isUsableSystemVersion(version.Version) {
				versions = append(versions, version)
			}
		}
		if len(versions) == 0 {
			continue
		}
		sdk.Versions = versions
		result = append(result, sdk)
	}
	return result
}
