package main

import (
	"fmt"
	"os"
)

func (s *Store) Flush() error {
	it := s.memtable.NewIterator()

	filename := fmt.Sprintf("sst-%04d.db", s.nextTableID)
	s.nextTableID++

	file, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_RDWR,
		0664,
	)

	if err != nil {
		return err
	}

	table := &SSTable{
		ID:       s.nextTableID,
		Filename: filename,
		File:     file,
	}

	s.sstables = append(s.sstables, table)

	for it.Next() {
		key := it.Key()
		val := it.Value()
		record := fmt.Sprintf("%s:%s\n", key, val)

		_, err := table.File.WriteString(record)
		if err != nil {
			return err
		}
	}

	s.memtable = NewSkiplist()
	s.wal.Sync()
	table.File.Sync()
	if err := s.wal.Truncate(0); err != nil {
		return err
	}
	return nil
}
