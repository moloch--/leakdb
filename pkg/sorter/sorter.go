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
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
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
	Entries   []Entry
	Dir       string
	FileName  string
	Size      int // Number of entires in tape file
	MergeSize int // Number of entires in merge buffer
	Position  int
	Messages  chan<- string
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
	t.Entries = make([]Entry, t.MergeSize)
	stop := position + t.MergeSize
	for index := 0; position < stop; index++ {
		tapeFile.Seek(int64(position)*entrySize, 0)
		buf := make([]byte, entrySize)
		_, err := io.ReadAtLeast(tapeFile, buf, entrySize)
		if err == io.EOF {
			// t.Entries = t.Entries[:position]
			t.Entries = append([]Entry(nil), t.Entries[:index]...)
			break
		}
		if err != nil {
			panic(err)
		}
		t.Entries[index] = Entry{
			Digest: buf[:digestSize],
			Offset: buf[digestSize:],
		}
		position++
	}
	t.Position = position
}

// Pop - Pop lowest value from tape
func (t *Tape) Pop() (Entry, bool) {
	var entry Entry
	if len(t.Entries) == 0 {
		if t.IsEndOfTape() {
			return Entry{}, false // End of tape
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
//               The file is zero indexed but the Postion will
//               get incremented until EOF, so no Size-1.
func (t *Tape) IsEndOfTape() bool {
	return t.Position == t.Size
}

// Index - An index file
type Index struct {
	file          *os.File
	Output        *os.File
	Info          os.FileInfo
	Size          int // Number of entries
	MaxGoRoutines int // Max number of worker go routines
	MaxMemory     int // size of buffer in bytes
	Messages      chan<- string
	Tapes         []*Tape
	TapeDir       string
	NoTapeCleanup bool
	Heap          *binaryheap.Heap
}

// Get - Get an index entry at position
func (idx *Index) Get(index int) Entry {
	position := int64(index * entrySize)
	entry := Entry{
		Digest: make([]byte, digestSize),
		Offset: make([]byte, offsetSize),
	}
	idx.file.ReadAt(entry.Digest, position)
	idx.file.ReadAt(entry.Offset, position+digestSize)
	return entry
}

// PopulateHeap - Populate the heap with lowest values from sorted tapes
func (idx *Index) PopulateHeap() {
	for tapeIndex, tape := range idx.Tapes {
		entry, okay := tape.Pop()
		if okay {
			entry.TapeIndex = tapeIndex
			idx.Heap.Push(entry)
		}
	}
}

// Drain - Drain buffer to file
func (idx *Index) Drain(outputBuf []Entry) {
	for _, entry := range outputBuf {
		_, err := idx.Output.Write(entry.Digest)
		if err != nil {
			panic(err)
		}
		_, err = idx.Output.Write(entry.Offset)
		if err != nil {
			panic(err)
		}
	}
}

// ceilDivideInt - Divide two ints and round up
func ceilDivideInt(a, b int) int {
	return int(math.Ceil(float64(a) / float64(b)))
}

// Sort - Sorts the index
func (idx *Index) Sort() {

	err := os.MkdirAll(idx.TapeDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer func() {
		if !idx.NoTapeCleanup {
			os.RemoveAll(idx.TapeDir)
		}
	}()

	memPerWorker := ceilDivideInt(idx.MaxMemory, runtime.NumCPU()) // Max memory per worker
	tapeSize := ceilDivideInt(memPerWorker, entrySize)             // Number of entries in a single tape
	numberOfTapes := ceilDivideInt(idx.Size, tapeSize)             // Total number of tapes we need
	memPerTape := ceilDivideInt(idx.MaxMemory, numberOfTapes+1)    // Size in bytes
	mergeBufLen := ceilDivideInt(memPerTape, entrySize)            // Len of slice

	idx.Messages <- fmt.Sprintf("Max Memory: %d bytes, CPU Cores: %d, Memory Per Core: %d",
		idx.MaxMemory, runtime.NumCPU(), memPerWorker)
	idx.Messages <- fmt.Sprintf("Tape Size: %d, Number of Tapes: %d, Memory Per Tape: %d",
		tapeSize, numberOfTapes, memPerTape)

	wg := sync.WaitGroup{}
	workers := []*Worker{}
	queue := make(chan *Tape)
	quit := make(chan bool)

	// Start n workers equal to CPU core(s)
	// max memory will be approx splitBufSize*CPU Cores
	for id := 1; id <= runtime.NumCPU(); id++ {
		wg.Add(1)
		worker := &Worker{
			ID:            id,
			Queue:         queue,
			Quit:          quit,
			Wg:            &wg,
			MaxGoRoutines: idx.MaxGoRoutines,
		}
		worker.start()
		workers = append(workers, worker)
	}

	for tapeIndex := 0; tapeIndex < numberOfTapes; tapeIndex++ {
		idx.Messages <- fmt.Sprintf("Loading tape %d of %d ...", tapeIndex+1, numberOfTapes)
		tape := idx.CreateTape(tapeIndex, tapeSize)
		tape.MergeSize = mergeBufLen
		idx.Tapes = append(idx.Tapes, tape)
		idx.Messages <- fmt.Sprintf("Sorting tape %d of %d (%d entries) ...",
			tapeIndex+1, numberOfTapes, len(tape.Entries))
		queue <- tape // Feed tapes to workers
	}
	for _, worker := range workers {
		worker.Quit <- true
	}
	wg.Wait() // Wait for all quicksorts to complete

	// K-way merge sort using binary heap
	idx.Messages <- fmt.Sprintf("Buffering %d entries (%d bytes) ...", mergeBufLen, memPerTape)
	for _, tape := range idx.Tapes {
		tape.Prefetch(0)
	}
	idx.PopulateHeap()

	idx.Messages <- fmt.Sprintf("Merging tapes, please wait ...")
	outputBuf := make([]Entry, 0)
	count := 0
	mod := int(float64(idx.Size) / 100.0)
	for {
		value, okay := idx.Heap.Pop()
		count++
		if count%mod == 0 {
			percent := (float64(count) / float64(idx.Size)) * 100.0
			idx.Messages <- fmt.Sprintf("Merged %0.f%%", percent)
		}
		if !okay {
			panic("Failed to pop value from heap")
		}
		entry := value.(Entry)
		outputBuf = append(outputBuf, entry)
		if mergeBufLen < len(outputBuf) {
			idx.Drain(outputBuf)
			outputBuf = nil
			outputBuf = make([]Entry, 0)
		}
		nextEntry, okay := idx.Tapes[entry.TapeIndex].Pop()
		if okay {
			nextEntry.TapeIndex = entry.TapeIndex
			idx.Heap.Push(nextEntry)
		}
		if idx.IsMergeCompleted() {
			break
		}
	}
	idx.Drain(outputBuf)
	outputBuf = nil
}

// IsMergeCompleted - Returns true if all tapes have ended and heap is size 0
func (idx *Index) IsMergeCompleted() bool {
	for _, tape := range idx.Tapes {
		if !tape.IsEndOfTape() || 0 < len(tape.Entries) {
			return false
		}
	}
	if 0 < idx.Heap.Size() {
		return false
	}
	return true
}

// CreateTape - Creates a tape and loads the entire tap into memory
func (idx *Index) CreateTape(id int, n int) *Tape {
	tape := &Tape{
		ID:       id,
		Dir:      idx.TapeDir,
		FileName: fmt.Sprintf("%s_%d.tape", idx.Info.Name(), id),
		Position: 0,
		Messages: idx.Messages,
		Entries:  make([]Entry, n),
	}
	for entryIndex := 0; entryIndex < n; entryIndex++ {
		buf := make([]byte, entrySize)
		_, err := io.ReadAtLeast(idx.file, buf, entrySize)
		if err == io.EOF {
			// tape.Entries = tape.Entries[:entryIndex]
			tape.Entries = append([]Entry(nil), tape.Entries[:entryIndex]...)
			break
		}
		if err != nil {
			panic(err)
		}
		tape.Entries[entryIndex] = Entry{
			Digest: buf[:digestSize],
			Offset: buf[digestSize:],
		}
	}
	tape.Size = len(tape.Entries)
	return tape
}

// Worker - An instance of quicksort
type Worker struct {
	ID            int
	Queue         <-chan *Tape
	Quit          chan bool
	Wg            *sync.WaitGroup
	MaxGoRoutines int
}

func (w *Worker) start() {
	go func() {
		for {
			select {
			case tape := <-w.Queue:
				Quicksort(tape.Entries, w.MaxGoRoutines)
				tape.Save()
			case <-w.Quit:
				w.Wg.Done()
				return
			}
		}
	}()
}

// Quicksort - New quicksort implementation based on:
//             http://azundo.github.io/blog/concurrent-quicksort-in-go/
func Quicksort(entries []Entry, maxWorkers int) {
	if len(entries) <= 1 {
		return
	}
	workers := make(chan int, maxWorkers-1)
	for id := 0; id < (maxWorkers - 1); id++ {
		workers <- 1
	}
	workerQsort(entries, nil, workers)
}

func workerQsort(entries []Entry, done chan int, workers chan int) {
	// report to caller that we're finished
	if done != nil {
		defer func() { done <- 1 }()
	}

	if len(entries) <= 1 {
		return
	}
	// since we may use the doneChannel synchronously
	// we need to buffer it so the synchronous code will
	// continue executing and not block waiting for a read
	doneChannel := make(chan int, 1)

	pivotIndex := partition(entries)

	select {
	case <-workers:
		// if we have spare workers, use a goroutine
		go workerQsort(entries[:pivotIndex+1], doneChannel, workers)
	default:
		// if no spare workers, sort synchronously
		workerQsort(entries[:pivotIndex+1], nil, workers)
		// calling this here as opposed to using the defer
		doneChannel <- 1
	}
	// use the existing goroutine to sort above the pivot
	workerQsort(entries[pivotIndex+1:], nil, workers)
	// if we used a goroutine we'll need to wait for
	// the async signal on this channel, if not there
	// will already be a value in the channel and it shouldn't block
	<-doneChannel
	return
}

func partition(entries []Entry) (swapIndex int) {
	pivotIndex, pivotValue := pickPivot(entries)

	// swap right-most element and pivot
	entries[len(entries)-1], entries[pivotIndex] = entries[pivotIndex], entries[len(entries)-1]

	// sort elements keeping track of pivot's idx
	for index := 0; index < len(entries)-1; index++ {
		if entries[index].Value() < pivotValue {
			entries[index], entries[swapIndex] = entries[swapIndex], entries[index]
			swapIndex++
		}
	}

	// swap pivot back to its place and return
	entries[swapIndex], entries[len(entries)-1] = entries[len(entries)-1], entries[swapIndex]
	return
}

func pickPivot(entries []Entry) (int, uint64) {
	pivotIndex := rand.Intn(len(entries))
	pivot := entries[pivotIndex]
	return pivotIndex, pivot.Value()
}

func qsort(entries []Entry) {
	if len(entries) <= 1 {
		return
	}
	pivotIndex := partition(entries)
	qsort(entries[:pivotIndex+1])
	qsort(entries[pivotIndex+1:])
	return
}

func checkSort(messages chan<- string, idx *Index, verbose bool) {
	messages <- "Checking sort ... "
	if idx.Info.Size()%entrySize != 0 {
		messages <- "\nWarning: File size is irregular!\n"
	}
	for index := 0; index < idx.Size-1; index++ {
		entry := idx.Get(index)
		if verbose {
			messages <- fmt.Sprintf("%09d - [%d : %v]\n", index, entry.Value(), entry.Offset)
		}
		nextEntry := idx.Get(index + 1)
		if nextEntry.Value() < entry.Value() {
			if verbose {
				messages <- fmt.Sprintf("%09d - [%d : %v]\n", index, nextEntry.Value(), nextEntry.Offset)
			}
			messages <- "\nIndex is not sorted correctly!\n"
			return
		}
	}
	messages <- "sorted!\n"
}

// EntryComparer - Compares entries in an index
func EntryComparer(a, b interface{}) int {
	aAsserted := a.(Entry)
	aValue := aAsserted.Value()
	bAsserted := b.(Entry)
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

// Start - Start the sorting process
func Start(messages chan<- string, index, output string, maxMemory int, maxGoRoutines int, tempDir string, noTapeCleanup bool) error {
	indexStat, err := os.Stat(index)
	if os.IsNotExist(err) || indexStat.IsDir() {
		return err
	}
	indexFile, err := os.Open(index)
	if err != nil {
		return err
	}
	defer indexFile.Close()

	idx := &Index{
		file:          indexFile,
		Info:          indexStat,
		Size:          int(indexStat.Size() / entrySize),
		MaxMemory:     maxMemory * Mb,
		MaxGoRoutines: maxGoRoutines,
		Messages:      messages,
		TapeDir:       filepath.Join(tempDir, ".tapes"),
		NoTapeCleanup: noTapeCleanup,
		Heap:          binaryheap.NewWith(EntryComparer),
	}
	outputFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	idx.Output = outputFile

	idx.Sort()

	return nil
}
