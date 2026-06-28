package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"

	"github.com/geoberle/pulse/internal/informer"
	"github.com/geoberle/pulse/internal/workitem"
)

var _ informer.Store = (*Store)(nil)

type Store struct {
	path string
	log  logr.Logger
}

func New(path string, log logr.Logger) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
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

func (s *Store) Load() ([]*workitem.WorkItem, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		s.log.Error(err, "failed to read state file, starting fresh")
		return nil, nil
	}

	var items []*workitem.WorkItem
	if err := json.Unmarshal(data, &items); err != nil {
		s.log.Error(err, "failed to unmarshal state file, starting fresh")
		return nil, nil
	}

	var valid []*workitem.WorkItem
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := item.UnmarshalSpecRecursive(); err != nil {
			s.log.Error(err, "dropping item with invalid spec", "id", item.ID)
			continue
		}
		valid = append(valid, item)
	}
	return valid, nil
}
