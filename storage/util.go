package storage

import (
	"path/filepath"
	"os"
)

func wrapOpen(fPath string) (*os.File, error) {
	file, err := os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(fPath), 0755)
		if err != nil {
			return nil, err
		}
		return wrapOpen(fPath)
	}
	return file, err
}

