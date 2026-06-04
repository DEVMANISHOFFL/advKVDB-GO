package main

import (
	"fmt"
)

func (s *Store) Flush() error {
	it := s.memtable.NewIterator()

	for it.Next() {
		key := it.Key()
		val := it.Value()
		record := fmt.Sprintf("%s:%s\n", key, val)

		_, err := s.sstable.WriteString(record)
		if err != nil {
			return err
		}
	}

	s.memtable = NewSkiplist()
	s.wal.Sync()
	s.sstable.Sync()
	if err := s.wal.Truncate(0); err != nil {
		return err
	}
	return nil
}
