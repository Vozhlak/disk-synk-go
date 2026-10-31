package differ

import "disk-sync/src/internal/storage"

func Diff(source, dest []storage.File) []storage.File {
	destSet := make(map[string]struct{}, len(dest))

	for _, f := range dest {
		destSet[f.Name] = struct{}{}
	}

	diff := make([]storage.File, 0, len(source))

	for _, f := range source {
		if _, exists := destSet[f.Name]; !exists {
			diff = append(diff, f)
		}
	}

	return diff
}
