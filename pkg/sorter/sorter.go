package sorter

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

External Quicksort - Memory constrained quicksort program. This allows us to
                     sort an array of values larger than the amount of RAM
                     available.

For example, for sorting 900 megabytes of data using only 100 megabytes of RAM:
 * Read 100 MB of the data in main memory and sort by some conventional method,
   like quicksort.  Write the sorted data to disk.
 * Repeat steps 1 and 2 until all of the data is in sorted 100 MB chunks (there
   are 900MB / 100MB = 9 chunks), which now need to be merged into one single output
   file.
 * Read the first 10 MB (= 100MB / (9 chunks + 1)) of each sorted chunk into input
   buffers in main memory and allocate the remaining 10 MB for an output buffer. (In
   practice, it might provide better performance to make the output buffer larger and
   the input buffers slightly smaller.)
 * Perform a 9-way merge and store the result in the output buffer. Whenever the
   output buffer fills, write it to the final sorted file and empty it. Whenever any
   of the 9 input buffers empties, fill it with the next 10 MB of its associated 100 MB
   sorted chunk until no more data from the chunk is available. This is the key step that
   makes external merge sort work externally -- because the merge algorithm only makes
   one pass sequentially through each of the chunks, each chunk does not have to be loaded
   completely; rather, sequential parts of the chunk can be loaded as needed.
*/

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/emirpasic/gods/trees/binaryheap"
)

const (
	digestSize = 6
	offsetSize = 6
	entrySize  = digestSize + offsetSize

	// Kb - Kilobyte
	Kb = 1024
	// Mb - Megabyte
	Mb = 1024 * Kb
	// Gb - Gigabyte
	Gb = 1024 * Mb

	// StatusNotStarted - The Indexer has been created but not started
	StatusNotStarted = "Not Started"
	// StatusStarting - Indexer is starting
	StatusStarting = "Starting"
	// StatusSorting - Indexer is sorting tapes
	StatusSorting = "Sorting"
	// StatusMerging - Indexer is merging tapes
	StatusMerging = "Merging"
)

// Entry - [48-bit digest][48-bit offset] =  96-bit (12 byte) entry
type Entry struct {
	Digest    []byte
	Offset    []byte
	TapeIndex int // Only used during merge
}

// Value - The numeric value of the digest
func (e *Entry) Value() uint64 {
	buf := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	copy(buf, e.Digest)
	return binary.LittleEndian.Uint64(buf)
}

// Tape - A subsection of the index file that we can sort in-memory
type Tape struct {
	ID        int
	Entries   []*Entry
	Dir       string
	FileName  string
	Len       int // Number of entires in tape file
	MergeSize int // Number of entires in merge buffer
	Position  int
}

// Save - Save tape to disk in dir
func (t *Tape) Save() {
	tapeFilePath := filepath.Join(t.Dir, t.FileName)
	tapeFile, err := os.Create(tapeFilePath)
	if err != nil {
		panic(err)
	}
	defer tapeFile.Close()

	for _, entry := range t.Entries {
		_, err := tapeFile.Write(entry.Digest)
		if err != nil {
			panic(err)
		}
		_, err = tapeFile.Write(entry.Offset)
		if err != nil {
			panic(err)
		}
	}
	t.Entries = nil
}

// Prefetch - Prefetch t.MergeSize elements from position in tape
func (t *Tape) Prefetch(position int) {
	tapeFilePath := filepath.Join(t.Dir, t.FileName)
	tapeFile, err := os.Open(tapeFilePath)
	if err != nil {
		panic(err)
	}
	defer tapeFile.Close()

	t.Entries = nil
	t.Entries = make([]*Entry, t.MergeSize)
	stop := position + t.MergeSize
	for index := 0; position < stop; index++ {
		tapeFile.Seek(int64(position)*entrySize, 0)
		buf := make([]byte, entrySize)
		_, err := io.ReadAtLeast(tapeFile, buf, entrySize)
		if err == io.EOF {
			t.Entries = append([]*Entry(nil), t.Entries[:index]...)
			break
		}
		if err != nil {
			panic(err)
		}
		t.Entries[index] = &Entry{
			Digest: buf[:digestSize],
			Offset: buf[digestSize:],
		}
		position++
	}
	t.Position = position
}

// Pop - Pop lowest value from tape
func (t *Tape) Pop() (*Entry, bool) {
	var entry *Entry
	if len(t.Entries) == 0 {
		if t.IsEndOfTape() {
			return &Entry{}, false // End of tape
		}
		t.Prefetch(t.Position)
		entry = t.Entries[0]
	} else {
		entry = t.Entries[0]
		t.Entries = t.Entries[1:]
	}
	return entry, true
}

// IsEndOfTape - Returns true if end of tape has been reached
//               The file is zero indexed but the Position will
//               get incremented until EOF, so no Size-1.
func (t *Tape) IsEndOfTape() bool {
	return t.Position == t.Len
}

// Sorter - An index file
type Sorter struct {
	IndexPath  string
	Index      *os.File
	OutputPath string
	Output     *os.File
	Info       os.FileInfo

	MaxWorkers        int
	NumberOfEntires   int // Number of entries
	MaxMemory         int // size of buffer in bytes
	WorkerBufSize     int
	EntriesPerTape    int
	MaxPerTapeBufSize int
	MergeBufLen       int

	Tapes         []*Tape
	TapeDir       string
	NoTapeCleanup bool
	Heap          *binaryheap.Heap
	MergePercent  float64

	Workers []*Worker

	NumberOfTapes int
	Status        string
}

// Get - Get an index entry at position
func (s *Sorter) Get(index int) *Entry {
	position := int64(index * entrySize)
	entry := Entry{
		Digest: make([]byte, digestSize),
		Offset: make([]byte, offsetSize),
	}
	s.Index.ReadAt(entry.Digest, position)
	s.Index.ReadAt(entry.Offset, position+digestSize)
	return &entry
}

// PopulateHeap - Populate the heap with lowest values from sorted tapes
func (s *Sorter) PopulateHeap() {
	for tapeIndex, tape := range s.Tapes {
		entry, okay := tape.Pop()
		if okay {
			entry.TapeIndex = tapeIndex
			s.Heap.Push(entry)
		}
	}
}

// Drain - Drain buffer to file
func (s *Sorter) Drain(outputBuf []*Entry) {
	for _, entry := range outputBuf {
		_, err := s.Output.Write(entry.Digest)
		if err != nil {
			panic(err)
		}
		_, err = s.Output.Write(entry.Offset)
		if err != nil {
			panic(err)
		}
	}
}

// ceilDivideInt - Divide two ints and round up
func ceilDivideInt(a, b int) int {
	return int(math.Ceil(float64(a) / float64(b)))
}

// Start - Sorts the index
func (s *Sorter) Start() {
	s.Status = StatusStarting
	var err error

	s.Output, err = os.Create(s.OutputPath)
	if err != nil {
		panic(err)
	}
	defer s.Output.Close()

	s.Index, err = os.Open(s.IndexPath)
	if err != nil {
		panic(err)
	}
	defer s.Index.Close()

	err = os.MkdirAll(s.TapeDir, 0700)
	if err != nil {
		panic(err)
	}
	defer func() {
		if !s.NoTapeCleanup {
			os.RemoveAll(s.TapeDir)
		}
	}()

	//            Size = number of bytes
	// Len or NumberOf = number of entires in a slice or iterable
	s.WorkerBufSize = ceilDivideInt(s.MaxMemory, s.MaxWorkers)           // Max memory
	s.EntriesPerTape = ceilDivideInt(s.WorkerBufSize, entrySize)         // Size of each tape in bytes
	s.NumberOfTapes = ceilDivideInt(s.NumberOfEntires, s.EntriesPerTape) // Total number of tapes we need
	s.MaxPerTapeBufSize = ceilDivideInt(s.MaxMemory, s.NumberOfTapes+1)  // Merge tape buffer size
	s.MergeBufLen = ceilDivideInt(s.MaxPerTapeBufSize, entrySize)        // Len of slice

	wg := sync.WaitGroup{}
	s.Workers = []*Worker{}
	queue := make(chan *Tape)
	quit := make(chan bool)

	s.Status = StatusSorting
	for id := 1; id <= s.MaxWorkers; id++ {
		wg.Add(1)
		worker := &Worker{
			ID:             id,
			Queue:          queue,
			Quit:           quit,
			Wg:             &wg,
			TapesCompleted: 0,
		}
		worker.start()
		s.Workers = append(s.Workers, worker)
	}

	for tapeIndex := 0; tapeIndex < s.NumberOfTapes; tapeIndex++ {
		tape := s.CreateTape(tapeIndex, s.EntriesPerTape)
		tape.MergeSize = s.MergeBufLen
		s.Tapes = append(s.Tapes, tape)
		queue <- tape // Feed tapes to workers
	}
	for _, worker := range s.Workers {
		worker.Quit <- true
	}
	wg.Wait() // Wait for all quicksorts to complete

	// K-way merge sort using binary heap
	s.Status = StatusMerging
	for _, tape := range s.Tapes {
		tape.Prefetch(0)
	}
	s.PopulateHeap()

	outputBuf := make([]*Entry, 0)
	count := 0
	mod := int(float64(s.NumberOfEntires) / 100.0)
	if mod == 0 {
		mod = 1 // For small values mod can be 0 after integer math
	}
	for {
		value, okay := s.Heap.Pop()
		count++
		if count%mod == 0 {
			s.MergePercent = (float64(count) / float64(s.NumberOfEntires)) * 100.0
		}
		if !okay {
			panic("Failed to pop value from heap")
		}
		entry := value.(*Entry)
		outputBuf = append(outputBuf, entry)
		if s.MergeBufLen < len(outputBuf) {
			s.Drain(outputBuf)
			outputBuf = make([]*Entry, 0)
		}
		nextEntry, okay := s.Tapes[entry.TapeIndex].Pop()
		if okay {
			nextEntry.TapeIndex = entry.TapeIndex
			s.Heap.Push(nextEntry)
		}
		if s.IsMergeCompleted() {
			break
		}
	}
	s.Drain(outputBuf)
}

// IsMergeCompleted - Returns true if all tapes have ended and heap is size 0
func (s *Sorter) IsMergeCompleted() bool {
	for _, tape := range s.Tapes {
		if !tape.IsEndOfTape() || 0 < len(tape.Entries) {
			return false
		}
	}
	if 0 < s.Heap.Size() {
		return false
	}
	return true
}

// CreateTape - Creates a tape and loads the entire tap into memory
func (s *Sorter) CreateTape(id int, entriesPerTape int) *Tape {
	tape := &Tape{
		ID:       id,
		Dir:      s.TapeDir,
		FileName: fmt.Sprintf("%s_%d.tape", s.Info.Name(), id),
		Position: 0,
		Entries:  make([]*Entry, entriesPerTape),
	}
	for entryIndex := 0; entryIndex < entriesPerTape; entryIndex++ {
		buf := make([]byte, entrySize)
		_, err := io.ReadAtLeast(s.Index, buf, entrySize)
		if err == io.EOF {
			tape.Entries = append([]*Entry(nil), tape.Entries[:entryIndex]...)
			break
		}
		if err != nil {
			panic(err)
		}
		tape.Entries[entryIndex] = &Entry{
			Digest: buf[:digestSize],
			Offset: buf[digestSize:],
		}
	}
	tape.Len = len(tape.Entries)
	return tape
}

// TapesCompleted - Number of tapes completed
func (s *Sorter) TapesCompleted() int {
	sum := 0
	for _, worker := range s.Workers {
		sum += worker.TapesCompleted
	}
	return sum
}

// Worker - An instance of quicksort
type Worker struct {
	ID             int
	Queue          <-chan *Tape
	Quit           chan bool
	Wg             *sync.WaitGroup
	MaxGoRoutines  int
	TapesCompleted int
}

func (w *Worker) start() {
	go func() {
		for {
			select {
			case tape := <-w.Queue:
				Quicksort(tape.Entries)
				tape.Save()
				w.TapesCompleted++
			case <-w.Quit:
				w.Wg.Done()
				return
			}
		}
	}()
}

// Quicksort - Sort the entries
func Quicksort(entries []*Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value() > entries[j].Value()
	})
}

// EntryComparer - Compares entries in an index
func EntryComparer(a, b interface{}) int {
	aAsserted := a.(*Entry)
	aValue := aAsserted.Value()
	bAsserted := b.(*Entry)
	bValue := bAsserted.Value()
	switch {
	case aValue > bValue:
		return 1
	case aValue < bValue:
		return -1
	default:
		return 0
	}
}

// CheckSort - Check if an index is sorted
func CheckSort(index string, verbose bool) (bool, error) {

	indexStat, err := os.Stat(index)
	if os.IsNotExist(err) || indexStat.IsDir() {
		return false, err
	}
	indexFile, err := os.Open(index)
	if err != nil {
		return false, err
	}
	defer indexFile.Close()

	idx := &Sorter{
		Index: indexFile,
		Info:  indexStat,
	}

	if idx.Info.Size()%entrySize != 0 {
		return false, errors.New("Irregular file size")
	}
	for index := 0; index < idx.NumberOfEntires-1; index++ {
		entry := idx.Get(index)
		nextEntry := idx.Get(index + 1)
		if nextEntry.Value() < entry.Value() {
			msg := fmt.Sprintf("%09d - [%d : %v]\n", index, nextEntry.Value(), nextEntry.Offset)
			err := fmt.Errorf("Index is not sorted correctly: %s", msg)
			return false, err
		}
	}
	return true, nil
}

// GetSorter - Start the sorting process
func GetSorter(index, output string, maxWorkers, maxMemory int, tempDir string, noTapeCleanup bool) (*Sorter, error) {
	indexStat, err := os.Stat(index)
	if os.IsNotExist(err) {
		return nil, err
	}
	if indexStat.IsDir() || indexStat.Size() == 0 {
		return nil, errors.New("Invalid index file: target is directory or empty file")
	}

	sorter := &Sorter{
		IndexPath:       index,
		Info:            indexStat,
		NumberOfEntires: int(indexStat.Size() / entrySize),
		MaxWorkers:      maxWorkers,
		MaxMemory:       maxMemory * Mb,
		TapeDir:         filepath.Join(tempDir, ".tapes"),
		NoTapeCleanup:   noTapeCleanup,
		Heap:            binaryheap.NewWith(EntryComparer),
		OutputPath:      output,
	}
	return sorter, nil
}
