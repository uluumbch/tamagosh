package bookmark

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Store struct {
	path string
	data map[string][]string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(dir, "bookmarks.json"),
		data: map[string][]string{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &s.data)
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *Store) Add(scope, path string) error {
	for _, p := range s.data[scope] {
		if p == path {
			return nil
		}
	}
	s.data[scope] = append(s.data[scope], path)
	sort.Strings(s.data[scope])
	return s.save()
}

func (s *Store) Remove(scope, path string) error {
	list := s.data[scope]
	out := list[:0]
	for _, p := range list {
		if p != path {
			out = append(out, p)
		}
	}
	s.data[scope] = out
	return s.save()
}

func (s *Store) List(scope string) []string {
	out := make([]string, len(s.data[scope]))
	copy(out, s.data[scope])
	return out
}
