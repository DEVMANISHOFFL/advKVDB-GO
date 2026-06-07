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

	filter := NewBloomFilter(s.memtable.Size, 0.01)

	table := &SSTable{
		ID:       id,
		Filename: filename,
		File:     file,
		Filter:   filter,
	}

	s.sstables = append(s.sstables, table)

	currentOffset := uint32(0)
	var indexBuffer []byte
	for it.Next() {
		key := it.Key()
		val := it.Value()

		filter.Add(key)

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
	indexSize := uint32(len(indexBuffer))

	if _, err := table.File.Write(indexBuffer); err != nil {
		return err
	}
	currentOffset += indexSize

	bloomStartOffset := currentOffset

	bloomBufferSize := 8 + 8 + (len(filter.bitset) * 8)
	bloomBuf := make([]byte, bloomBufferSize)

	binary.LittleEndian.PutUint64(bloomBuf[0:8], uint64(filter.m))
	binary.LittleEndian.PutUint64(bloomBuf[8:16], uint64(filter.k))

	bOffset := 16
	for _, val := range filter.bitset {
		binary.LittleEndian.PutUint64(bloomBuf[bOffset:bOffset+8], val)
		bOffset += 8
	}

	if _, err := table.File.Write(bloomBuf); err != nil {
		return err
	}
	bloomSize := uint32(len(bloomBuf))

	footer := make([]byte, 16)

	binary.LittleEndian.PutUint32(footer[0:4], indexStartOffset)
	binary.LittleEndian.PutUint32(footer[4:8], indexSize)
	binary.LittleEndian.PutUint32(footer[8:12], bloomStartOffset)
	binary.LittleEndian.PutUint32(footer[12:16], bloomSize)

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
