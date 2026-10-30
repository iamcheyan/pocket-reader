package readerd

import (
	"os"
	"path/filepath"
)

func hasSupportedFile(root string) (bool, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		full := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			found, err := hasSupportedFile(full)
			if err != nil {
				return false, err
			}
			if found {
				return true, nil
			}
			continue
		}
		if entry.Type().IsRegular() && providerFor(full) != nil {
			return true, nil
		}
	}
	return false, nil
}
