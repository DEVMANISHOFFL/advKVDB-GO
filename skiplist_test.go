package main

// import (
// 	"fmt"
// 	"math/rand"
// 	"testing"
// )

// func TestInsertSearch(t *testing.T) {
// 	sl := NewSkiplist()

// 	for i := 0; i < 1000; i++ {
// 		key := fmt.Sprintf("user:%v", i)
// 		val := fmt.Sprintf("%v-pass-%v", i, i%3)
// 		sl.Insert(key, val)
// 	}

// 	for i := 0; i < 1000; i++ {
// 		key := fmt.Sprintf("user:%v", i)

// 		val, ok := sl.Search(key)
// 		if !ok {
// 			t.Fatalf("key %s not found", key)
// 		}
// 		a := fmt.Sprintf("%v-pass-%v", i, i%3)

// 		if val != a {
// 			t.Fatalf("expected %v, got %v", a, val)
// 		}
// 	}
// }

// func TestDuplicateUpdate(t *testing.T) {
// 	sl := NewSkiplist()

// 	sl.Insert("user:1", "manish")
// 	sl.Insert("user:1", "sachin")

// 	val, ok := sl.Search("user:1")
// 	if !ok {
// 		t.Fatalf("key user:1 not found")
// 	}

// 	if sl.Size != 1 {
// 		t.Fatalf("expected size 1, got %d", sl.Size)
// 	}

// 	if val != "sachin" {
// 		t.Fatalf("expected sachin, got %v", val)
// 	}
// }

// func TestDeleteSkiplist(t *testing.T) {
// 	sl := NewSkiplist()

// 	sl.Insert("User:A", "UniqueA")
// 	sl.Insert("User:B", "UniqueB")
// 	sl.Insert("User:C", "UniqueC")
// 	sl.Insert("User:D", "UniqueD")
// 	sl.Insert("User:E", "UniqueE")

// 	sl.Delete("User:C")

// 	val, ok := sl.Search("User:C")

// 	if ok {
// 		t.Fatalf("expected key to be deleted")
// 	}

// 	if val != "" {
// 		t.Fatalf("expected empty value, got %q", val)
// 	}

// 	if _, ok := sl.Search("User:B"); !ok {
// 		t.Fatalf("User:B should still exist")
// 	}

// 	if _, ok := sl.Search("User:D"); !ok {
// 		t.Fatalf("User:D should still exist")
// 	}

// 	if sl.Size != 4 {
// 		t.Fatalf("expected size 4, got %d", sl.Size)
// 	}
// }

// func TestDeleteNonExistent(t *testing.T) {
// 	sl := NewSkiplist()

// 	sl.Insert("a", "1")
// 	deleted := sl.Delete("dne")

// 	if deleted {
// 		t.Fatalf("key should not be available")
// 	}

// 	if sl.Size != 1 {
// 		t.Fatalf("expected size 1, got %d", sl.Size)
// 	}
// }

// func TestRandomizedAgainstMap(t *testing.T) {
// 	r := rand.New(rand.NewSource(42))

// 	sl := NewSkiplist()
// 	truth := make(map[string]string)

// 	for i := 0; i <= 10_000; i++ {
// 		a := r.Intn(1000)
// 		key := fmt.Sprintf("user-%d", a)
// 		value := fmt.Sprintf("pass-%d", a)

// 		sl.Insert(key, value)
// 		truth[key] = value
// 	}

// 	for key, expected := range truth {
// 		val_Skiplist, ok := sl.Search(key)
// 		if !ok {
// 			t.Fatalf("key %s not found", key)
// 		}

// 		if expected != val_Skiplist {
// 			t.Fatalf(
// 				"key=%s expected=%s got=%s",
// 				key,
// 				expected,
// 				val_Skiplist,
// 			)
// 		}

// 	}

// 	if sl.Size != len(truth) {
// 		t.Fatalf("expected size to be same")
// 	}
// }

// func TestOrderedIteration(t *testing.T) {
// 	sl := NewSkiplist()

// 	expected := []string{
// 		"a",
// 		"b",
// 		"c",
// 		"d",
// 	}

// 	kv := [][]string{{"b", "2"}, {"a", "1"}, {"d", "4"}, {"c", "3"}}

// 	for i := range kv {
// 		sl.Insert(kv[i][0], kv[i][1])
// 	}
// 	it := sl.NewIterator()

// 	index := 0

// 	for it.Next() {
// 		actual := it.Key()
// 		if actual != expected[index] {
// 			t.Fatalf(
// 				"expected %s got %s at position %d",
// 				expected[index],
// 				actual,
// 				index,
// 			)
// 		}
// 		index++
// 	}

// }
