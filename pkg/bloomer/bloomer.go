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

// Worker - Worker thread
type Worker struct {
	ID          int
	Queue       <-chan string
	Quit        chan bool
	Bloom       *bloom.BloomFilter
	BloomMutex  *sync.RWMutex
	Wg          *sync.WaitGroup
	OutputMutex *sync.Mutex
	Output      *os.File
}

func (w *Worker) start() {
	go func() {
		for {
			select {
			case line := <-w.Queue:
				w.BloomMutex.Lock()
				exists := w.Bloom.TestAndAddString(line)
				w.BloomMutex.Unlock()
				if !exists {
					w.OutputMutex.Lock()
					w.Output.WriteString(fmt.Sprintf("%s\n", line))
					w.OutputMutex.Unlock()
				}
			case <-w.Quit:
				w.Wg.Done()
				return
			}
		}
	}()
}

// Start - Start the bloomer
func Start(targets []string, output, saveFilter, loadFilter string, maxWorkers, filterSize, filterHashes uint) error {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	lines := make(chan string, lineBufferSize)
	go lineQueue(targets, lines)

	outputFile, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Create filter and optionally load content from previously saved file
	bloomFilter := bloom.New(uint(filterSize*gb), filterHashes)
	if _, err := os.Stat(loadFilter); !os.IsNotExist(err) {
		loadFile, err := os.Open(loadFilter)
		if err != nil {
			return err
		}
		defer loadFile.Close()
		bloomFilter.ReadFrom(loadFile)
	}

	queue := make(chan string)
	quit := make(chan bool)
	outputMutex := sync.Mutex{}
	bloomMutex := sync.RWMutex{}
	wg := sync.WaitGroup{}
	workers := []*Worker{}
	for id := 1; id <= int(maxWorkers); id++ {
		wg.Add(1)
		worker := &Worker{
			ID:          id,
			Queue:       queue,
			Quit:        quit,
			Bloom:       bloomFilter,
			BloomMutex:  &bloomMutex,
			Wg:          &wg,
			OutputMutex: &outputMutex,
			Output:      outputFile,
		}
		worker.start()
		workers = append(workers, worker)
	}
	for line := range lines {
		line = strings.TrimSpace(line)
		if 0 < len(line) {
			queue <- line
		}
	}
	for _, worker := range workers {
		worker.Quit <- true
	}
	wg.Wait()

	// Optionally save bloom filter
	if 0 < len(saveFilter) {
		saveFile, err := os.Create(saveFilter)
		if err != nil {
			return err
		}
		defer saveFile.Close()
		bloomFilter.WriteTo(saveFile)
	}
	return nil
}

// GetTargets - Get targets from target directory
func GetTargets(target string) ([]string, error) {
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
