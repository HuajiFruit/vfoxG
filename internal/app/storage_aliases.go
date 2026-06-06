package app

import "vfoxG/internal/storage"

func (a *App) writeJSONFile(path string, v interface{}) error {
	return storage.WriteJSONFile(path, v)
}
