package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

type sstRes struct {
	k   string
	val string
}

type SSTIterator struct {
	table      *SSTable
	fileOffset int64
	endOffset  int64

	currentKey string
	currentVal string

	err error
}

func (s *Store) SSTReader(table *SSTable, searchKey string) (string, error, bool) {
	stat, err := table.File.Stat()
	if err != nil {
		return "", err, false
	}

	fileSize := stat.Size()
	footer := make([]byte, 16)

	// V2 Footer is 16 bytes long
	if _, err := table.File.ReadAt(footer, fileSize-16); err != nil {
		return "", err, false
	}

	indexStartOffset := binary.LittleEndian.Uint32(footer[0:4])
	indexSize := binary.LittleEndian.Uint32(footer[4:8])

	indexBuf := make([]byte, indexSize)
	if _, err := table.File.ReadAt(indexBuf, int64(indexStartOffset)); err != nil {
		return "", err, false
	}

	var dataOffset uint32
	found := false
	ptr := 0

	for ptr < len(indexBuf) {
		keyLen := binary.LittleEndian.Uint16(indexBuf[ptr : ptr+2])
		ptr += 2

		offset := binary.LittleEndian.Uint32(indexBuf[ptr : ptr+4])
		ptr += 4

		key := string(indexBuf[ptr : ptr+int(keyLen)])
		ptr += int(keyLen)

		if key == searchKey {
			dataOffset = offset
			found = true
			break
		}
	}

	if !found {
		return "", nil, false
	}

	header := make([]byte, 6)
	if _, err := table.File.ReadAt(header, int64(dataOffset)); err != nil {
		return "", err, false
	}

	keyLen := binary.LittleEndian.Uint16(header[0:2])
	valLen := binary.LittleEndian.Uint32(header[2:6])

	payload := make([]byte, int(keyLen)+int(valLen))
	payloadOffset := int64(dataOffset) + 6

	if _, err := table.File.ReadAt(payload, payloadOffset); err != nil {
		return "", err, false
	}

	key := string(payload[:keyLen])
	if key != searchKey {
		return "", nil, false
	}

	val := string(payload[keyLen:])
	return val, nil, true
}

func NewSSTIterator(table *SSTable) (*SSTIterator, error) {
	stat, err := table.File.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := stat.Size()
	footer := make([]byte, 16)

	if _, err := table.File.ReadAt(footer, fileSize-16); err != nil {
		return nil, err
	}

	indexStartOffset := binary.LittleEndian.Uint32(footer[0:4])

	return &SSTIterator{
		table:      table,
		fileOffset: 0,
		endOffset:  int64(indexStartOffset),
	}, nil
}

func (s *Store) LoadSSTable() error {
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	maxID := -1

	for _, file := range files {
		name := file.Name()
		if !strings.HasPrefix(name, "sst-") {
			continue
		}

		var id int
		_, err := fmt.Sscanf(name, "sst-%04d.db", &id)
		if err != nil {
			continue
		}

		f, err := os.Open(name)
		if err != nil {
			return err
		}

		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		fileSize := stat.Size()

		footer := make([]byte, 16)
		if _, err := f.ReadAt(footer, fileSize-16); err != nil {
			f.Close()
			return err
		}

		bloomStartOffset := binary.LittleEndian.Uint32(footer[8:12])
		bloomSize := binary.LittleEndian.Uint32(footer[12:16])

		bloomBuf := make([]byte, bloomSize)
		if _, err := f.ReadAt(bloomBuf, int64(bloomStartOffset)); err != nil {
			f.Close()
			return err
		}

		m := binary.LittleEndian.Uint64(bloomBuf[0:8])
		k := binary.LittleEndian.Uint64(bloomBuf[8:16])

		// Reassemble the bitset array
		bitsetLen := (bloomSize - 16) / 8
		bitset := make([]uint64, bitsetLen)

		bOffset := 16
		for i := uint32(0); i < bitsetLen; i++ {
			bitset[i] = binary.LittleEndian.Uint64(bloomBuf[bOffset : bOffset+8])
			bOffset += 8
		}

		filter := &BloomFilters{
			bitset: bitset,
			m:      uint(m),
			k:      uint(k),
		}

		table := &SSTable{
			ID:       id,
			Filename: name,
			File:     f,
			Filter:   filter,
		}

		s.sstables = append(s.sstables, table)

		if id > maxID {
			maxID = id
		}
	}

	s.nextTableID = maxID + 1
	return nil
}

func (it *SSTIterator) Next() bool {
	if it.err != nil {
		return false
	}

	if it.fileOffset >= it.endOffset {
		return false
	}

	header := make([]byte, 6)
	if _, err := it.table.File.ReadAt(header, it.fileOffset); err != nil {
		it.err = err
		return false
	}

	keyLen := binary.LittleEndian.Uint16(header[0:2])
	valLen := binary.LittleEndian.Uint32(header[2:6])

	payload := make([]byte, int(keyLen)+int(valLen))
	payloadOffset := it.fileOffset + 6

	if _, err := it.table.File.ReadAt(payload, payloadOffset); err != nil {
		it.err = err
		return false
	}

	it.currentKey = string(payload[:keyLen])
	it.currentVal = string(payload[keyLen:])

	it.fileOffset += 6 + int64(keyLen) + int64(valLen)
	return true
}

func (it *SSTIterator) Key() string   { return it.currentKey }
func (it *SSTIterator) Value() string { return it.currentVal }
func (it *SSTIterator) Err() error    { return it.err }
