package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/workitem"
)

type Store struct {
	path string
	log  logr.Logger
}

func New(path string, log logr.Logger) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}
	return &Store{path: path, log: log}, nil
}

func (s *Store) Save(items []*workitem.WorkItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".pulse-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// Load reads persisted WorkItems from disk. Tolerates partial corruption:
// nil items and items with invalid specs are logged and dropped. Returns
// nil error even when items are dropped (tolerate-on-read convention).
func (s *Store) Load() ([]*workitem.WorkItem, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var items []*workitem.WorkItem
	if err := json.Unmarshal(data, &items); err != nil {
		s.log.Info("corrupt state file, starting fresh", "error", err)
		return nil, nil
	}

	var valid []*workitem.WorkItem
	for i, item := range items {
		if item == nil {
			s.log.Info("dropping nil item from persisted state", "index", i)
			continue
		}
		normalizeNamesRecursive(item)
		if err := item.UnmarshalSpecRecursive(); err != nil {
			s.log.Error(err, "dropping item with invalid spec", "name", item.Name)
			continue
		}
		valid = append(valid, item)
	}
	return valid, nil
}

func normalizeNamesRecursive(item *workitem.WorkItem) {
	item.Name = normalizeName(item.Name)
	for _, child := range item.Children {
		if child != nil {
			normalizeNamesRecursive(child)
		}
	}
}

func normalizeName(name string) string {
	name = strings.ReplaceAll(name, ":", ".")
	name = strings.ReplaceAll(name, "/", ".")
	return strings.ToLower(name)
}
