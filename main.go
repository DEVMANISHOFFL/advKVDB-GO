package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const MEMTABLE_SIZE = 4

type LogEntry struct {
	Cmd   string
	Key   string
	Value string
}

type Store struct {
	mu       sync.RWMutex
	wal      *os.File
	memtable *Skiplist
	sstable  *os.File
}

func NewStore() (*Store, error) {
	file, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)
	sstableFile, err := os.OpenFile("sst-0001.db", os.O_CREATE|os.O_RDWR, 0664)

	if err != nil {
		return nil, fmt.Errorf("error opening wal.log")
	}
	return &Store{
		wal:      file,
		memtable: NewSkiplist(),
		mu:       sync.RWMutex{},
		sstable:  sstableFile,
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
	if !ok {
		val, err := s.SSTReader(key)
		if err != nil {
			return "key not found in sstable", err
		}
		return val, nil
	}

	return val, nil
}

func (s *Store) Delete(key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := LogEntry{
		Cmd: "DELETE",
		Key: key,
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
	if ok := s.memtable.Delete(key); ok {
		return true, nil
	}
	return false, nil
}

func main() {
	store, err := NewStore()
	if err != nil {
		panic(err)
	}

	store.Replay()

	// for i := range 100 {
	// 	k, v := fmt.Sprintf("user-%d", i), fmt.Sprintf("pass-%d", i%3)
	// 	store.Set(k, v)
	// }

	store.Set("manishdevoffl1", "testing flush1")
	store.Set("manishdevoffl2", "testing flush2")
	store.Set("manishdevoffl3", "testing flush3")
	store.Set("manishdevoffl4", "testing flush4")

	op, _ := store.Get("manishdevoffl2")

	fmt.Println(op)
}
