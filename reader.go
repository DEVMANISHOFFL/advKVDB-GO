package main

import (
	"encoding/binary"
	"os"
	"strings"
)

type sstRes struct {
	k   string
	val string
}

func (s *Store) SSTReader(table *SSTable, searchKey string) (string, error, bool) {

	stat, err := table.File.Stat()
	if err != nil {
		return "", err, false
	}

	fileSize := stat.Size()

	footer := make([]byte, 8)

	if _, err := table.File.ReadAt(
		footer,
		fileSize-8,
	); err != nil {
		return "", err, false
	}

	indexStartOffset := binary.LittleEndian.Uint32(
		footer[0:4],
	)

	indexSize := binary.LittleEndian.Uint32(
		footer[4:8],
	)

	indexBuf := make([]byte, indexSize)

	if _, err := table.File.ReadAt(
		indexBuf,
		int64(indexStartOffset),
	); err != nil {
		return "", err, false
	}

	var dataOffset uint32
	found := false

	ptr := 0

	for ptr < len(indexBuf) {

		keyLen := binary.LittleEndian.Uint16(
			indexBuf[ptr : ptr+2],
		)
		ptr += 2

		offset := binary.LittleEndian.Uint32(
			indexBuf[ptr : ptr+4],
		)
		ptr += 4

		key := string(
			indexBuf[ptr : ptr+int(keyLen)],
		)
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

	if _, err := table.File.ReadAt(
		header,
		int64(dataOffset),
	); err != nil {
		return "", err, false
	}

	keyLen := binary.LittleEndian.Uint16(
		header[0:2],
	)

	valLen := binary.LittleEndian.Uint32(
		header[2:6],
	)

	payload := make(
		[]byte,
		int(keyLen)+int(valLen),
	)

	payloadOffset := int64(dataOffset) + 6

	if _, err := table.File.ReadAt(
		payload,
		payloadOffset,
	); err != nil {
		return "", err, false
	}

	key := string(payload[:keyLen])

	if key != searchKey {
		return "", nil, false
	}

	val := string(payload[keyLen:])

	return val, nil, true	
}

func (s *Store) LoadSSTable() error {
	files, err := os.ReadDir(".")

	if err != nil {
		return err
	}

	for _, file := range files {
		name := file.Name()

		if !strings.HasPrefix(name, "sst-") {
			continue
		}

		f, err := os.Open(name)
		if err != nil {
			return err
		}

		table := &SSTable{
			Filename: name,
			File:     f,
		}

		s.sstables = append(s.sstables, table)

	}

	return nil
}
