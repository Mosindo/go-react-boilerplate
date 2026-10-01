package app_test

import "example.com/api/internal/platform/storage"

func newStore(dir string) (storage.Store, error) { return storage.NewLocal(dir) }
