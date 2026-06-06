package app

import "fmt"

func (a *App) tryStartVfoxTask() (func(), error) {
	a.vfoxTaskMutex.Lock()
	if a.vfoxTaskBusy {
		a.vfoxTaskMutex.Unlock()
		return nil, fmt.Errorf("another terminal task is already running")
	}
	a.vfoxTaskBusy = true
	a.vfoxTaskMutex.Unlock()

	return func() {
		a.vfoxTaskMutex.Lock()
		a.vfoxTaskBusy = false
		a.vfoxTaskMutex.Unlock()
	}, nil
}
