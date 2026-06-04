package main

import (
	"bufio"
	"strings"
)

type sstRes struct {
	k   string
	val string
}

func (s *Store) SSTReader(key string) (string, error) {

	if _, err := s.sstable.Seek(0, 0); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(s.sstable)

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
			return res.val, nil
		}
	}

	return "", scanner.Err()
}
