package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type LogEntry struct {
	Cmd   string
	Key   string
	Value string
}

type Store struct {
	m   map[string]string
	mu  sync.RWMutex
	wal *os.File
}

func NewStore() (*Store, error) {
	file, err := os.OpenFile("wal.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return nil, fmt.Errorf("error opening wal.log")
	}
	return &Store{
		m:   make(map[string]string),
		wal: file,
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
			s.m[entry.Key] = entry.Value
		}

		if entry.Cmd == "DELETE" {
			delete(s.m, entry.Key)
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
	s.wal.Sync()

	s.m[key] = val
	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.m[key]; ok {
		return val, ok
	}
	return "", false
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
	s.wal.Sync()

	if _, ok := s.m[key]; ok {
		delete(s.m, key)
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

	store.Set("user-1", "manish")
	store.Set("user-2", "hello\nworld")
	store.Delete("user-1")

	if val, ok := store.Get("user-2"); ok {
		fmt.Println(val)
	} else {
		fmt.Println("key not found")
	}

}
