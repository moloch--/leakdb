package search

/*
	---------------------------------------------------------------------
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
	----------------------------------------------------------------------
*/

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	digestSize = 6
	offsetSize = 6

	// EntrySize - Single entry size in bytes
	EntrySize = digestSize + offsetSize
)

// Credential - JSON parsed line
type Credential struct {
	Email    string
	User     string
	Domain   string
	Password string
}

// Entry - [48-bit digest][48-bit offset] = 96-bit (12 byte) entry
type Entry struct {
	Digest []byte
	Offset []byte
}

// Value - The numeric value of the digest
func (e *Entry) Value() uint64 {
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(buf, e.Digest)
	return binary.LittleEndian.Uint64(buf)
}

// OffsetInt64 - Offset as an int64
func (e *Entry) OffsetInt64() int64 {
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(buf, e.Offset)
	return int64(binary.LittleEndian.Uint64(buf))
}

// GetEntry - Get an index entry from file at index
func GetEntry(indexFile *os.File, index int) *Entry {
	position := int64(index * EntrySize)
	fmt.Printf("Reading entry at position %d\n", position)
	entry := &Entry{
		Digest: make([]byte, digestSize),
		Offset: make([]byte, offsetSize),
	}
	_, err := indexFile.ReadAt(entry.Digest, position)
	if err != nil {
		panic("file read error")
	}
	_, err = indexFile.ReadAt(entry.Offset, position+digestSize)
	if err != nil {
		panic("file read error")
	}
	return entry
}

func binaryTreeWalk(needle uint64, indexFile *os.File, numberOfEntries int) (int, error) {
	lower := 0
	upper := numberOfEntries - 1 // Zero index
	fmt.Printf("Needle is %d, lower = %d, upper = %d\n", needle, lower, upper)
	for lower <= upper {
		middle := lower + ((upper - lower) / 2)
		fmt.Printf("middle = %d\n", middle)
		entryValue := GetEntry(indexFile, middle).Value()
		if needle < entryValue {
			fmt.Printf("Needle is lower than %d (%d)\n", middle, entryValue)
			upper = middle - 1
		} else if entryValue < needle {
			fmt.Printf("Needle is higher than %d (%d)\n", middle, entryValue)
			lower = middle + 1
		} else {
			fmt.Printf("End search at %d\n", entryValue)
			return middle, nil
		}
	}
	return -1, errors.New("Entry not found")
}

func binaryTreeSearch(needle uint64, targetFile, indexFile *os.File, numberOfEntries int) []Credential {
	results := []Credential{}
	match, err := binaryTreeWalk(needle, indexFile, numberOfEntries)
	if err != nil {
		return []Credential{}
	}
	match--
	for GetEntry(indexFile, match).Value() == needle {
		match-- // Walk backwards and find the first entry
	}
	match++
	fmt.Printf("First match at: %d\n", match)
	for GetEntry(indexFile, match).Value() == needle {
		entry := GetEntry(indexFile, match)
		targetFile.Seek(entry.OffsetInt64(), 0)
		reader := bufio.NewReader(targetFile)
		line, _ := reader.ReadString('\n')
		var cred Credential
		json.Unmarshal([]byte(line), &cred)
		results = append(results, cred)
		match++
	}
	return results
}

func linearSearch(needle uint64, targetFile, indexFile *os.File, numberOfEntries int) []Credential {
	results := []Credential{}
	fmt.Printf("Using linear search ...\n")
	hit := false
	for index := 0; index < numberOfEntries; index++ {
		entry := GetEntry(indexFile, index)
		if entry.Value() == needle {
			fmt.Printf("%d == %d\n", entry.Value(), needle)
			hit = true
			fmt.Printf("Found a result at index %d (offset: %x)\n", index, entry.OffsetInt64())
			targetFile.Seek(entry.OffsetInt64(), 0)
			reader := bufio.NewReader(targetFile)
			line, _ := reader.ReadString('\n')
			fmt.Printf("Read: %s\n", line)
			var cred Credential
			json.Unmarshal([]byte(line), &cred)
			results = append(results, cred)
		} else if hit {
			break // We reached the end of all matches
		}
	}
	return results
}

// Find - Fine a value in the index file
func Find(value string, targetFile, indexFile *os.File, numberOfEntries int, linear bool) []Credential {
	digest := sha256.Sum256([]byte(value))
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(buf, digest[:digestSize])
	needle := binary.LittleEndian.Uint64(buf)

	fmt.Printf("Finding %s -> %x\n", value, buf)
	var results []Credential
	if linear {
		results = linearSearch(needle, targetFile, indexFile, numberOfEntries)
	} else {
		results = binaryTreeSearch(needle, targetFile, indexFile, numberOfEntries)
	}
	return results
}
