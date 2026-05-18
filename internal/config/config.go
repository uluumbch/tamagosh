package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Connection struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	User    string `json:"user"`
	PassKey string `json:"pass_key"`
}

type Store struct {
	Connections []Connection `json:"connections"`
}

func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Store{Connections: []Connection{}}, nil
	}
	if err != nil {
		return nil, err
	}
	s := &Store{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.Connections == nil {
		s.Connections = []Connection{}
	}
	return s, nil
}

func Save(path string, s *Store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
