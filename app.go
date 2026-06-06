package main

import (
	"context"
	"sync"
)

// App holds process-wide state for the Wails facade.
type App struct {
	ctx           context.Context
	homeMu        sync.RWMutex
	vfoxHome      string
	vfoxTaskMutex sync.Mutex
	vfoxTaskBusy  bool
}
