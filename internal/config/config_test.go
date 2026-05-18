package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.json")

	want := &Store{Connections: []Connection{
		{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra", PassKey: "ssh/atlantic"},
	}}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Connections) != 0 {
		t.Fatalf("expected empty store, got %+v", s)
	}
}
