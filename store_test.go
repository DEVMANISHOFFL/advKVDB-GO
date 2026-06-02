package main

import (
	"os"
	"testing"
)

func setupStore(t *testing.T) *Store {
	t.Helper()

	_ = os.Remove("wal.log")

	store, err := NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store
}

func TestGetAndSet(t *testing.T) {
	store := setupStore(t)

	err := store.Set("User1", "manishdevoffl")
	if err != nil {
		t.Fatalf("set failed, %v", err)
	}

	got, ok := store.Get("User1")

	if !ok {
		t.Fatalf("expected key to exist")
	}

	if got != "manishdevoffl" {
		t.Fatalf("expected manishishdevoffl, got %s", got)
	}
}

func TestDelete(t *testing.T) {
	store := setupStore(t)

	if err := store.Set("user1", "manishdevoffl"); err != nil {
		t.Fatal(err)
	}

	_, err := store.Delete("user1")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := store.Get("user1")

	if ok {
		t.Fatalf("expected key to be deleted")
	}
}

func TestReplay(t *testing.T) {
	_ = os.Remove("wal.log")

	store := setupStore(t)

	if err := store.Set("user1", "manish"); err != nil {
		t.Fatal(err)
	}

	if err := store.Set("user2", "manishdevoffl"); err != nil {
		t.Fatal(err)
	}

	store.wal.Close()

	recovered, err := NewStore()

	if err != nil {
		t.Fatal(err)
	}

	if err := recovered.Replay(); err != nil {
		t.Fatal(err)
	}

	val, ok := recovered.Get("user2")
	if !ok {
		t.Fatalf("expected user2 after replay")
	}

	if val != "manishdevoffl" {
		t.Fatalf("expected manishdevoffl, got %s", val)
	}
}
