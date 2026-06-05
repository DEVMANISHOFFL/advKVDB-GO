package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const MEMTABLE_SIZE = 4
const TOMBSTONE = "__DELETED__"

type LogEntry struct {
	Cmd   string
	Key   string
	Value string
}
type SSTable struct {
	ID       int
	Filename string
	File     *os.File
}

type Store struct {
	mu          sync.RWMutex
	wal         *os.File
	memtable    *Skiplist
	sstables    []*SSTable
	nextTableID int
}

func NewStore() (*Store, error) {

	file, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)

	if err != nil {
		return nil, fmt.Errorf("error opening wal.log")
	}
	return &Store{
		wal:      file,
		memtable: NewSkiplist(),
		mu:       sync.RWMutex{},
		sstables: make([]*SSTable, 0),
	}, nil
}

func (s *Store) Replay() error {

	file, err := os.Open("wal.log")
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		var entry LogEntry

		if err := json.Unmarshal(line, &entry); err != nil {
			return err
		}

		if entry.Cmd == "SET" {
			s.memtable.Insert(entry.Key, entry.Value)
		}

		if entry.Cmd == "DELETE" {
			s.memtable.Delete(entry.Key)
		}

	}
	return scanner.Err()
}

func (s *Store) Set(key, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := LogEntry{
		Cmd:   "SET",
		Key:   key,
		Value: val,
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := s.wal.Write(bytes); err != nil {
		return err
	}
	if _, err := s.wal.WriteString("\n"); err != nil {
		return err
	}
	if err := s.wal.Sync(); err != nil {
		return err
	}
	s.memtable.Insert(entry.Key, entry.Value)

	if s.memtable.Size >= MEMTABLE_SIZE {
		return s.Flush()
	}

	return nil
}

func (s *Store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.memtable.Search(key)

	if ok {
		if val == TOMBSTONE {
			return "key not found", nil
		}

		return val, nil
	}

	for i := len(s.sstables) - 1; i >= 0; i-- {
		val, err, ok := s.SSTReader(s.sstables[i], key)

		if err != nil {
			return "", err
		}

		if !ok {
			continue
		}

		if val == TOMBSTONE {
			return "key not found", nil
		}

		return val, nil
	}

	return "key not found", nil
}

func (s *Store) Delete(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := LogEntry{
		Cmd:   "DELETE",
		Key:   key,
		Value: TOMBSTONE,
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return false, err
	}

	if _, err := s.wal.Write(bytes); err != nil {
		return false, err
	}
	if _, err := s.wal.WriteString("\n"); err != nil {
		return false, err
	}
	// Calling s.wal.Sync() triggers an fsync system call. This bypasses the Page Cache and forces the mechanical
	// hard drive or SSD to physically record the data before the function returns. It is much slower, but it guarantees
	// durability.
	if err := s.wal.Sync(); err != nil {
		return false, err
	}
	s.memtable.Insert(key, TOMBSTONE)
	return false, nil
}

func main() {
	store, err := NewStore()
	if err != nil {
		panic(err)
	}

	store.Replay()

	if err := store.LoadSSTable(); err != nil {
		panic(err)
	}
	// for i := range 100 {
	// 	k, v := fmt.Sprintf("user-%d", i), fmt.Sprintf("pass-%d", i%3)
	// 	store.Set(k, v)
	// }
	store.Delete("user-30")
	op, _ := store.Get("user-30")

	fmt.Println(op)
}
