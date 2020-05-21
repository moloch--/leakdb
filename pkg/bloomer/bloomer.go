package bloomer

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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/willf/bloom"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	lineBufferSize = 4096
)

// Bloom - Tracks a single bloom job
type Bloom struct {
	outputFile  *os.File
	workers     []*Worker
	bloomFilter *bloom.BloomFilter
	targets     []string
	queue       chan string
	save        string
	wg          *sync.WaitGroup
}

// Progress - Returns items bloomed and number of duplicates
func (b *Bloom) Progress() (int, int) {
	count := 0
	duplicates := 0
	for _, worker := range b.workers {
		count += worker.Count
		duplicates += worker.CountDuplicates
	}
	return count, duplicates
}

// Start - Start the bloom filter workers
func (b *Bloom) Start() error {

	defer b.outputFile.Close()

	lines := make(chan string, lineBufferSize)
	go lineQueue(b.targets, lines)

	for _, worker := range b.workers {
		worker.start()
	}

	for line := range lines {
		line = strings.TrimSpace(line)
		if 0 < len(line) {
			b.queue <- line
		}
	}
	for _, worker := range b.workers {
		worker.Quit <- true
	}

	// Optionally save bloom filter
	if 0 < len(b.save) {
		saveFile, err := os.Create(b.save)
		if err != nil {
			return err
		}
		defer saveFile.Close()
		b.bloomFilter.WriteTo(saveFile)
	}

	b.wg.Wait()
	return nil
}

// Worker - Worker thread
type Worker struct {
	ID              int
	Queue           <-chan string
	Quit            chan bool
	Bloom           *bloom.BloomFilter
	BloomMutex      *sync.RWMutex
	Wg              *sync.WaitGroup
	OutputMutex     *sync.Mutex
	Output          *os.File
	Count           int
	CountDuplicates int
}

func (w *Worker) start() {
	go func() {
		w.Wg.Add(1)
		for {
			select {
			case line := <-w.Queue:
				w.Count++
				w.BloomMutex.Lock()
				exists := w.Bloom.TestAndAddString(line)
				w.BloomMutex.Unlock()
				if !exists {
					w.OutputMutex.Lock()
					w.Output.WriteString(fmt.Sprintf("%s\n", line))
					w.OutputMutex.Unlock()
				} else {
					w.CountDuplicates++
				}
			case <-w.Quit:
				w.Wg.Done()
				return
			}
		}
	}()
}

// GetBloomer - Start the bloomer
func GetBloomer(target string, output, saveFilter, loadFilter string, maxWorkers, filterSize, filterHashes uint) (*Bloom, error) {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	targets, err := getTargets(target)
	if err != nil {
		return nil, err
	}

	outputFile, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	// Create filter and optionally load content from previously saved file
	bloomFilter := bloom.New(uint(filterSize*gb), filterHashes)
	if _, err := os.Stat(loadFilter); !os.IsNotExist(err) {
		loadFile, err := os.Open(loadFilter)
		if err != nil {
			return nil, err
		}
		defer loadFile.Close()
		bloomFilter.ReadFrom(loadFile)
	}

	queue := make(chan string)
	quit := make(chan bool)
	outputMutex := sync.Mutex{}
	bloomMutex := sync.RWMutex{}
	wg := &sync.WaitGroup{}

	workers := []*Worker{}
	for id := 1; id <= int(maxWorkers); id++ {
		worker := &Worker{
			ID:          id,
			Queue:       queue,
			Quit:        quit,
			Bloom:       bloomFilter,
			BloomMutex:  &bloomMutex,
			OutputMutex: &outputMutex,
			Output:      outputFile,
			Wg:          wg,
		}
		workers = append(workers, worker)
	}

	return &Bloom{
		targets:     targets,
		bloomFilter: bloomFilter,
		outputFile:  outputFile,
		workers:     workers,
		queue:       queue,
		wg:          wg,
	}, nil
}

// getTargets - Get targets from target directory
func getTargets(target string) ([]string, error) {
	targetStat, err := os.Stat(target)
	if err != nil {
		return []string{}, err
	}
	switch mode := targetStat.Mode(); {
	case mode.IsDir():
		files, err := ioutil.ReadDir(target)
		if err != nil {
			return []string{}, nil
		}
		targets := []string{}
		for _, file := range files {
			if err != nil || file.IsDir() {
				continue
			}
			targetPath := filepath.Join(target, file.Name())
			targets = append(targets, targetPath)
		}
		return targets, nil
	case mode.IsRegular():
		return []string{target}, nil
	}
	return []string{}, nil
}

func lineQueue(targets []string, lines chan<- string) error {
	defer close(lines)
	for _, target := range targets {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			return err
		}
		file, err := os.Open(target)
		if err != nil {
			return err
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				lines <- line
				break
			}
			if err != nil {
				return err
			}
			lines <- line
		}
	}
	return nil
}
