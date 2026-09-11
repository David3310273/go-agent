package core

/*
simplified interface to avoid option bloat across different storage backends.
*/
type Storage interface {
	SetConfig(path string)
	Stop()
}

func InitStorageClient(storage Storage, configPath string) {
	storage.SetConfig(configPath)
}

func StopStorageClient(storage Storage) {
	storage.Stop()
}
