package searcher

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
	"io"
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
	entry := &Entry{
		Digest: make([]byte, digestSize),
		Offset: make([]byte, offsetSize),
	}
	_, err := indexFile.ReadAt(entry.Digest, position)
	if err != nil && err != io.EOF {
		panic(fmt.Sprintf("file read error as position %d (%s)", position, err))
	}
	_, err = indexFile.ReadAt(entry.Offset, position+digestSize)
	if err != nil && err != io.EOF {
		panic(fmt.Sprintf("file read error as position %d (%s)", position+digestSize, err))
	}
	return entry
}

func binaryTreeWalk(messages chan<- string, needle uint64, indexFile *os.File, numberOfEntries int) (int, error) {
	lower := 0
	upper := numberOfEntries - 1 // Zero index
	messages <- fmt.Sprintf("Needle is %d, lower = %d, upper = %d\n", needle, lower, upper)
	for lower <= upper {
		middle := lower + ((upper - lower) / 2)
		messages <- fmt.Sprintf("middle = %d\n", middle)
		entryValue := GetEntry(indexFile, middle).Value()
		if needle < entryValue {
			messages <- fmt.Sprintf("Needle is lower than %d (%d)\n", middle, entryValue)
			upper = middle - 1
		} else if entryValue < needle {
			messages <- fmt.Sprintf("Needle is higher than %d (%d)\n", middle, entryValue)
			lower = middle + 1
		} else {
			messages <- fmt.Sprintf("End search at %d\n", entryValue)
			return middle, nil
		}
	}
	return -1, errors.New("Entry not found")
}

func binaryTreeSearch(messages chan<- string, needle uint64, targetFile, indexFile *os.File, numberOfEntries int) []Credential {
	results := []Credential{}
	match, err := binaryTreeWalk(messages, needle, indexFile, numberOfEntries)
	if err != nil {
		return []Credential{}
	}
	match--
	for GetEntry(indexFile, match).Value() == needle {
		match-- // Walk backwards and find the first entry
	}
	match++
	messages <- fmt.Sprintf("First match at: %d\n", match)
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

func find(messages chan<- string, value string, targetFile, indexFile *os.File, numberOfEntries int) ([]Credential, error) {
	digest := sha256.Sum256([]byte(value))
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(buf, digest[:digestSize])
	needle := binary.LittleEndian.Uint64(buf)
	messages <- fmt.Sprintf("Finding %s -> %x\n", value, buf)
	results := binaryTreeSearch(messages, needle, targetFile, indexFile, numberOfEntries)
	return results, nil
}

// Start - Fine a value in the index file
func Start(messages chan<- string, value string, target string, index string) ([]Credential, error) {
	targetStat, err := os.Stat(target)
	if os.IsNotExist(err) || targetStat.IsDir() {
		return nil, err
	}
	targetFile, err := os.OpenFile(target, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer targetFile.Close()

	indexStat, err := os.Stat(index)
	if os.IsNotExist(err) || indexStat.IsDir() {
		return nil, err
	}
	indexFile, err := os.OpenFile(index, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer indexFile.Close()

	numberOfEntries := int(indexStat.Size() / EntrySize)
	return find(messages, value, targetFile, indexFile, numberOfEntries)
}
