package main

import "os"

func (s *Store) loadTableIntoMap(
	table *SSTable,
	data map[string]string,
) error {

	it, err := NewSSTIterator(table)
	if err != nil {
		return err
	}

	for it.Next() {
		data[it.Key()] = it.Value()
	}

	return it.Err()
}
func (s *Store) Compaction() error {

	if len(s.sstables) <= 1 {
		return nil
	}

	data := make(map[string]string)

	for _, table := range s.sstables {

		if err := s.loadTableIntoMap(
			table,
			data,
		); err != nil {
			return err
		}
	}

	// rebuild memtable
	mem := NewSkiplist()

	for k, v := range data {

		// remove tombstones
		if v == TOMBSTONE {
			continue
		}

		mem.Insert(k, v)
	}

	// save current memtable
	oldMemtable := s.memtable

	// temporarily replace
	s.memtable = mem

	// write compacted SSTable
	if err := s.Flush(); err != nil {
		s.memtable = oldMemtable
		return err
	}

	// restore
	s.memtable = oldMemtable

	// delete old SSTables
	for _, table := range s.sstables[:len(s.sstables)-1] {

		table.File.Close()

		if err := os.Remove(
			table.Filename,
		); err != nil {
			return err
		}
	}

	// keep only newest compacted SSTable
	s.sstables = s.sstables[len(s.sstables)-1:]

	return nil
}
