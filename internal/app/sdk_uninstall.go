package app

func (a *App) uninstallVersion(name, version string) error {
	name, version, err := validateSdkNameAndVersion(name, version)
	if err != nil {
		return err
	}
	releaseTask, err := a.tryStartVfoxTask()
	if err != nil {
		a.emitEvent("vfox-busy")
		return err
	}
	defer releaseTask()

	return a.uninstallVersionUnlocked(name, version)
}

func (a *App) uninstallVersionUnlocked(name, version string) error {
	return a.runVfoxWithProgress([]string{"uninstall", name + "@" + version})
}
