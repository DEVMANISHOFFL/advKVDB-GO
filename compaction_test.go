package main

import (
	"fmt"
	"testing"
)

func TestCompaction(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	store.Set("a", "1")
	store.Set("b", "2")
	store.Set("c", "3")
	store.Set("d", "4")

	store.Set("b", "20")
	store.Set("c", TOMBSTONE)
	store.Set("e", "5")
	store.Set("f", "6")

	if len(store.sstables) != 2 {
		t.Fatalf("expected 2 SSTables, got %d", len(store.sstables))
	}

	if err := store.Compaction(); err != nil {
		t.Fatal(err)
	}
	for _, table := range store.sstables {
		fmt.Println(table.Filename)
	}

	if len(store.sstables) != 1 {
		t.Fatalf("expected 1 SSTable after compaction")
	}

	val, _ := store.Get("a")
	if val != "1" {
		t.Fatalf("expected a=1 got %s", val)
	}

	val, _ = store.Get("b")
	if val != "20" {
		t.Fatalf("expected b=20 got %s", val)
	}

	val, _ = store.Get("c")
	if val != "key not found" {
		t.Fatalf("c should be deleted")
	}

	val, _ = store.Get("e")
	if val != "5" {
		t.Fatalf("expected e=5 got %s", val)
	}
}
