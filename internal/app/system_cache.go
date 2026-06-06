package app

import "sync"

var (
	systemSdkCache   []SdkInfo
	systemSdkCacheMu sync.RWMutex
)

func (a *App) getCachedSystemSdks() []SdkInfo {
	systemSdkCacheMu.RLock()
	defer systemSdkCacheMu.RUnlock()
	result := make([]SdkInfo, len(systemSdkCache))
	copy(result, systemSdkCache)
	return result
}

func setSystemSdkCache(sdks []SdkInfo) {
	systemSdkCacheMu.Lock()
	systemSdkCache = sdks
	systemSdkCacheMu.Unlock()
}

func (a *App) getSystemSdkCacheFile() string {
	return a.getVfoxHomePath("gui-system-sdks-cache.json")
}
