package app_test

import (
	"os"
	"path/filepath"
)

func osStat(dir, key string) (os.FileInfo, error) { return os.Stat(filepath.Join(dir, key)) }
