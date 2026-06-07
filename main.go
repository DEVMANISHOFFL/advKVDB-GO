package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const MEMTABLE_SIZE = 4
const TOMBSTONE = "__DELETED__"
const (
	CMD_SET    byte = 0
	CMD_DELETE byte = 1
)

type LogEntry struct {
	Cmd   []byte
	Key   []byte
	Value []byte
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

	header := make([]byte, 7)

	for {
		_, err := io.ReadFull(file, header)
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		CMD := header[0]
		keyLen := binary.LittleEndian.Uint16(header[1:3])
		valLen := binary.LittleEndian.Uint32(header[3:7])

		payLoad := make([]byte, int(keyLen)+int(valLen))
		_, err = io.ReadFull(file, payLoad)
		if err != nil {
			return err
		}

		key := payLoad[:keyLen]
		val := payLoad[keyLen:]

		switch CMD {
		case CMD_SET:
			s.memtable.Insert(string(key), string(val))
		case CMD_DELETE:
			s.memtable.Insert(string(key), TOMBSTONE)
		}
	}
	return nil
}

func (s *Store) Set(key, val string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyLen := uint16(len(key))
	valLen := uint32(len(val))

	totalSize := 1 + 2 + 4 + int(keyLen) + int(valLen)

	buf := make([]byte, totalSize)

	offset := 0
	buf[offset] = CMD_SET
	offset += 1

	binary.LittleEndian.PutUint16(buf[offset:], keyLen)
	offset += 2

	binary.LittleEndian.PutUint32(buf[offset:], valLen)
	offset += 4

	copy(buf[offset:], key)
	offset += int(keyLen)

	copy(buf[offset:], val)
	offset += int(valLen)

	if _, err := s.wal.Write(buf); err != nil {
		return err
	}

	if err := s.wal.Sync(); err != nil {
		return err
	}

	s.memtable.Insert(key, val)

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

	keyLen := uint16(len(key))
	totalSize := 1 + 2 + 4 + int(keyLen)

	buf := make([]byte, totalSize)

	offset := 0

	buf[offset] = CMD_DELETE
	offset += 1

	binary.LittleEndian.PutUint16(buf[offset:], keyLen)
	offset += 2

	binary.LittleEndian.PutUint32(buf[offset:], 0)
	offset += 4

	copy(buf[offset:], key)
	offset += int(keyLen)

	if _, err := s.wal.Write(buf); err != nil {
		return false, err
	}

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

	// store.Set("survivor", "this data survived a crash")
	// store.Delete("survivor")
	val, _ := store.Get("survivor")
	// store.memtable.Delete("survivor")
	fmt.Println(val)
	// op, _ := store.Get("user-30")

	// fmt.Println(op)
}
