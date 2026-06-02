package main

import (
	"fmt"
	"testing"
)

func TestInsertSearch(t *testing.T) {
	sl := NewSkiplist()

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:%v", i)
		val := fmt.Sprintf("%v-pass-%v", i, i%3)
		sl.Insert(key, val)
	}

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user:%v", i)

		val, ok := sl.Search(key)
		if !ok {
			t.Fatalf("key %s not found", key)
		}
		a := fmt.Sprintf("%v-pass-%v", i, i%3)

		if val != a {
			t.Fatalf("expected %v, got %v", a, val)
		}
	}
}

func TestDuplicateUpdate(t *testing.T) {
	sl := NewSkiplist()

	sl.Insert("user:1", "manish")
	sl.Insert("user:1", "sachin")

	val, ok := sl.Search("user:1")
	if !ok {
		t.Fatalf("key user:1 not found")
	}

	if sl.Size != 1 {
		t.Fatalf("expected size 1, got %d", sl.Size)
	}

	if val != "sachin" {
		t.Fatalf("expected sachin, got %v", val)
	}
}

func TestOrderedIteration(t *testing.T) {
	sl := NewSkiplist()

	expected := []string{
		"a",
		"b",
		"c",
		"d",
	}

	kv := [][]string{{"b", "2"}, {"a", "1"}, {"d", "4"}, {"c", "3"}}

	for i := range kv {
		sl.Insert(kv[i][0], kv[i][1])
	}
	it := sl.NewIterator()

	index := 0

	for it.Next() {
		actual := it.Key()

		if actual != expected[index] {
			t.Fatalf("failed")
		}

		index++
	}

}
