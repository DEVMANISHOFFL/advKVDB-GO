package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

func (s *Store) Compaction() error {
	// map for kv
	merged := make(map[string]string)
	// range all sstables
	for _, table := range s.sstables {
		// take cursor to 0,0
		if _, err := table.File.Seek(0, 0); err != nil {
			return err
		}
		scanner := bufio.NewScanner(table.File)

		// scan file
		for scanner.Scan() {
			line := scanner.Text()
			// split into parts (:)
			parts := strings.SplitN(line, ":", 2)

			// extract kv
			key := parts[0]
			val := parts[1]

			// keep them in map
			merged[key] = val
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}
	// range merged and merge for tombstone values, if found delete from map
	for key, val := range merged {
		if val == TOMBSTONE {
			delete(merged, key)
		}
	}

	// have new file name and nextTable++
	filename := fmt.Sprintf("sst-%04d.db", s.nextTableID)
	s.nextTableID++

	// open that file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	// append all keys to new map called keys and sort it.
	keys := make([]string, 0, len(merged))

	for key := range merged {
		keys = append(keys, key)
	}

	slices.Sort(keys)
	// range new key map and create record to write to new ssfile (compacted write)
	for _, key := range keys {
		record := fmt.Sprintf("%s:%s\n", key, merged[key])

		if _, err := file.WriteString(record); err != nil {
			return err
		}
	}
	// sync file
	file.Sync()
	// range table and close all also remove table.filename
	for _, table := range s.sstables {
		table.File.Close()
		os.Remove(table.Filename)
	}
	// newtable
	newTable := &SSTable{
		ID:       s.nextTableID - 1,
		Filename: filename,
		File:     file,
	}

	// append this new table in sstable
	s.sstables = append(s.sstables, newTable)
	return nil
}
