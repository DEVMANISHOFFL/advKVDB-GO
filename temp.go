package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

func (s *Store) TempCompaction() error {
	merged := make(map[string]string)

	for _, table := range s.sstables {

		if _, err := table.File.Seek(0, 0); err != nil {
			return err
		}
		scanner := bufio.NewScanner(table.File)

		for scanner.Scan() {
			line := scanner.Text()

			parts := strings.SplitN(line, ":", 2)

			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			val := parts[1]

			merged[key] = val
		}

		if err := scanner.Err(); err != nil {
			return err
		}

	}
	for key, val := range merged {
		if val == TOMBSTONE {
			delete(merged, key)
		}
	}

	filename := fmt.Sprintf("sst-%04d.db", s.nextTableID)

	s.nextTableID++

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	keys := make([]string, 0, len(merged))

	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		record := fmt.Sprintf(
			"%s:%s\n",
			key,
			merged[key],
		)

		if _, err := file.WriteString(record); err != nil {
			return err
		}
	}

	file.Sync()

	for _, table := range s.sstables {
		table.File.Close()
		os.Remove(table.Filename)
	}
	newTable := &SSTable{
		ID:       s.nextTableID - 1,
		Filename: filename,
		File:     file,
	}

	s.sstables = []*SSTable{newTable}

	return nil

}
