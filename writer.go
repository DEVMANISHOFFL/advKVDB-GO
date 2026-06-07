package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func (s *Store) Flush() error {
	it := s.memtable.NewIterator()

	id := s.nextTableID
	s.nextTableID++

	filename := fmt.Sprintf("sst-%04d.db", id)
	file, err := os.OpenFile(
		filename,
		os.O_CREATE|os.O_RDWR,
		0664,
	)

	if err != nil {
		return err
	}

	table := &SSTable{
		ID:       id,
		Filename: filename,
		File:     file,
	}

	s.sstables = append(s.sstables, table)

	currentOffset := uint32(0)
	var indexBuffer []byte

	for it.Next() {
		key := it.Key()
		val := it.Value()

		keyLen := uint16(len(key))
		valLen := uint32(len(val))

		totalSize := 2 + 4 + int(keyLen) + int(valLen)
		indexEntrySize := 2 + 4 + int(keyLen)

		buf := make([]byte, totalSize)

		idx := make([]byte, indexEntrySize)

		offset := 0

		binary.LittleEndian.PutUint16(buf[offset:], keyLen)
		offset += 2

		binary.LittleEndian.PutUint32(buf[offset:], valLen)
		offset += 4

		copy(buf[offset:], key)
		offset += int(keyLen)

		copy(buf[offset:], val)
		offset += int(valLen)

		binary.LittleEndian.PutUint16(idx[0:2], keyLen)
		binary.LittleEndian.PutUint32(idx[2:6], currentOffset)
		copy(idx[6:], key)

		indexBuffer = append(indexBuffer, idx...)

		if _, err := table.File.Write(buf); err != nil {
			return err
		}
		currentOffset += uint32(len(buf))
	}

	indexStartOffset := currentOffset

	if _, err := table.File.Write(indexBuffer); err != nil {
		return err
	}

	footer := make([]byte, 8)

	binary.LittleEndian.PutUint32(
		footer[0:4],
		indexStartOffset,
	)

	binary.LittleEndian.PutUint32(
		footer[4:8],
		uint32(len(indexBuffer)),
	)

	if _, err := table.File.Write(footer); err != nil {
		return err
	}

	s.memtable = NewSkiplist()
	s.wal.Sync()
	table.File.Sync()
	if err := s.wal.Truncate(0); err != nil {
		return err
	}
	return nil
}
