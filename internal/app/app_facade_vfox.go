package app

// RunVfoxCommand runs a short vfox command with task locking when needed.
func (a *App) RunVfoxCommand(args ...string) (string, error) {
	return a.runVfoxCommandWithLock(args...)
}

// RunVfoxWithProgress runs a long vfox command and streams output to the UI.
func (a *App) RunVfoxWithProgress(args []string) error {
	return a.runVfoxWithProgressLocked(args)
}
