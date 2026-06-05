package main

import (
	"bufio"
	"os"
	"strings"
)

type sstRes struct {
	k   string
	val string
}

func (s *Store) SSTReader(table *SSTable, key string) (string, error, bool) {

	if _, err := table.File.Seek(0, 0); err != nil {
		return "", err, false
	}
	scanner := bufio.NewScanner(table.File)

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.SplitN(line, ":", 2)

		if len(parts) != 2 {
			continue
		}

		res := sstRes{
			k:   parts[0],
			val: parts[1],
		}
		if res.k == key {
			return res.val, nil, true
		}
	}

	return "", scanner.Err(), false
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
