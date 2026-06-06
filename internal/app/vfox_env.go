package app

import (
	"os"
	stdruntime "runtime"
	"strings"
)

// getCleanedEnvForVfox returns a copy of the current environment but sanitizes the PATH
// to remove any previously injected vfox SDK/shim paths.
func (a *App) getCleanedEnvForVfox() []string {
	env := os.Environ()
	vfoxHome := strings.TrimSpace(a.getVfoxHome())
	roots := a.vfoxManagedPathRoots()

	for i, envValue := range env {
		if strings.HasPrefix(strings.ToLower(envValue), "path=") {
			env[i] = "PATH=" + cleanPathValue(envValue[5:], roots)
			break
		}
	}
	// 添加伪装变量，使用 cmd/bash 可以避免 vfox 弹出子 shell 而导致死锁
	shellName := "bash"
	if stdruntime.GOOS == "windows" {
		shellName = "cmd"
	}
	env = upsertEnv(env, "VFOX_HOME", vfoxHome)
	env = upsertEnv(env, "__VFOX_SHELL", shellName)
	return env
}

func upsertEnv(env []string, key string, value string) []string {
	prefix := key + "="
	for i, envValue := range env {
		name, _, ok := strings.Cut(envValue, "=")
		if ok && envKeyEqual(name, key) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func envKeyEqual(leftKey string, rightKey string) bool {
	if stdruntime.GOOS == "windows" {
		return strings.EqualFold(leftKey, rightKey)
	}
	return leftKey == rightKey
}
