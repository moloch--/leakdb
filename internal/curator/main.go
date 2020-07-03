package curator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/moloch--/leakdb/pkg/bloomer"
	"github.com/moloch--/leakdb/pkg/indexer"
	"github.com/moloch--/leakdb/pkg/sorter"
	"github.com/spf13/cobra"
)

// BloomConfig - Bloom filter configuration
type BloomConfig struct {
	FilterSize   uint   `json:"filter_size"`
	FilterHashes uint   `json:"filter_hashes"`
	Workers      uint   `json:"workers"`
	FilterLoad   string `json:"filter_load"`
	FilterSave   string `json:"filter_save"`
	Output       string `json:"output"`
}

// IndexConfig - Index generation configuration
type IndexConfig struct {
	Workers   uint     `json:"workers"`
	Keys      []string `json:"keys"`
	NoCleanup bool     `json:"no_cleanup"`
}

// SortConfig - Sort configuration
type SortConfig struct {
	MaxGoRoutines uint `json:"max_goroutines"`
	MaxMemory     uint `json:"max_memory"`
	NoCleanup     bool `json:"no_cleanup"`
}

// AutoConfig - A complete config for the auto command
type AutoConfig struct {
	Bloom *BloomConfig `json:"bloom"`
	Index *IndexConfig `json:"index"`
	Sort  *SortConfig  `json:"sort"`

	Input     string `json:"input_dir"`
	OutputDir string `json:"output_dir"`
	TempDir   string `json:"temp_dir"`
}

func mainRun(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
		return
	}

	autoConf := &AutoConfig{
		Bloom: &BloomConfig{},
		Index: &IndexConfig{NoCleanup: false, Keys: []string{"user", "email", "domain"}},
		Sort:  &SortConfig{NoCleanup: false},
	}

	// Workers
	workers, err := cmd.Flags().GetUint(workersFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", workersFlagStr, err)
		return
	}
	if workers < 1 {
		workers = 1
	}
	autoConf.Bloom.Workers = workers
	autoConf.Index.Workers = workers

	// Bloom Filter Options
	autoConf.Bloom.FilterSize, err = cmd.Flags().GetUint(filterSizeFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterSizeFlagStr, err)
		return
	}
	autoConf.Bloom.FilterHashes, err = cmd.Flags().GetUint(filterHashesFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterHashesFlagStr, err)
		return
	}
	autoConf.Bloom.FilterLoad, err = cmd.Flags().GetString(filterLoadFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterLoadFlagStr, err)
		return
	}
	autoConf.Bloom.FilterSave, err = cmd.Flags().GetString(filterSaveFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", filterSaveFlagStr, err)
		return
	}

	// Memory/goroutines
	autoConf.Sort.MaxMemory, err = cmd.Flags().GetUint(maxMemoryFlagStr)
	if autoConf.Sort.MaxMemory < 1 {
		autoConf.Sort.MaxMemory = 1
	}
	autoConf.Sort.MaxGoRoutines, err = cmd.Flags().GetUint(maxGoRoutinesFlagStr)
	if autoConf.Sort.MaxGoRoutines < 1 {
		autoConf.Sort.MaxGoRoutines = 1
	}

	// Target input/output
	autoConf.Input, err = cmd.Flags().GetString(jsonFlagStr) // Dir or file of normalized json
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", jsonFlagStr, err)
		return
	}
	autoConf.OutputDir, err = cmd.Flags().GetString(outputFlagStr) // Output dir of indexes
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", outputFlagStr, err)
		return
	}
	if autoConf.OutputDir == "" {
		autoConf.OutputDir = filepath.Join(cwd, "leakdb")
	}
	if _, err = os.Stat(autoConf.OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(autoConf.OutputDir, 0700)
		if err != nil {
			fmt.Printf(Warn+"Error creating output directory %s", err)
			return
		}
	}

	// Temp Dir
	autoConf.TempDir, err = cmd.Flags().GetString(tempDirFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", tempDirFlagStr, err)
		return
	}
	rand.Seed(time.Now().UnixNano())
	dirName := fmt.Sprintf("leakdb-tmp-%d", rand.Intn(999999))
	if autoConf.TempDir == "" {
		autoConf.TempDir = filepath.Join(cwd, dirName)
	} else {
		autoConf.TempDir = filepath.Join(autoConf.TempDir, dirName)
	}
	err = os.MkdirAll(autoConf.TempDir, 0700)
	if err != nil {
		fmt.Printf(Warn+"Failed to create temp dir %s", err)
		return
	}
	defer os.RemoveAll(autoConf.TempDir)

	err = auto(autoConf)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	}
}

func defaultConf(generate string) error {
	workers := uint(runtime.NumCPU())
	conf := &AutoConfig{
		Input:     "",
		OutputDir: "",
		TempDir:   "",
		Bloom: &BloomConfig{
			FilterSize:   8,
			FilterHashes: 14,
			Workers:      workers,
			FilterLoad:   "",
			FilterSave:   "",
			Output:       "bloomed.json",
		},
		Index: &IndexConfig{
			Workers:   workers,
			Keys:      []string{"email", "user", "domain"},
			NoCleanup: false,
		},
		Sort: &SortConfig{
			MaxGoRoutines: 10000,
			MaxMemory:     2048,
			NoCleanup:     false,
		},
	}

	data, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(generate, data, 0644)
}

func auto(conf *AutoConfig) error {
	started := time.Now()
	// Check input & output locations
	_, err := os.Stat(conf.Input)
	if os.IsNotExist(err) {
		return fmt.Errorf("Input error %s %s", conf.Input, err)
	}

	// *** Bloom ***
	bloomed, err := bloom(conf)
	if err != nil {
		return err
	}

	// *** Index ***
	indexes, err := index(bloomed, conf)
	if err != nil {
		return err
	}

	// *** Sort ***
	err = sort(indexes, conf)
	if err != nil {
		return err
	}

	fmt.Printf("Completed in %s\n", time.Now().Sub(started))

	return nil
}

func bloom(conf *AutoConfig) (string, error) {
	fmt.Printf("Applying bloom filter ...\u001b[s")
	bloomOutput := conf.Bloom.Output
	if bloomOutput == "" {
		bloomOutput = filepath.Join(conf.OutputDir, "bloomed.json")
	}
	bloom, err := bloomer.GetBloomer(conf.Input, bloomOutput, conf.Bloom.FilterSave,
		conf.Bloom.FilterLoad, conf.Bloom.Workers, conf.Bloom.FilterSize, conf.Bloom.FilterHashes)
	if err != nil {
		return "", err
	}

	// Progress animation
	done := make(chan bool)
	go bloomProgress(bloom, done)
	err = bloom.Start()
	done <- true
	<-done
	if err != nil {
		return "", err
	}
	fmt.Printf("\u001b[u done!\n")
	return bloomOutput, nil
}

func bloomProgress(bloom *bloomer.Bloom, done chan bool) {
	stdout := bufio.NewWriter(os.Stdout)
	fmt.Println()
	fmt.Println()
	fmt.Println()
	lastCount := 0
	for {
		select {
		case <-time.After(time.Second):
			count, duplicates := bloom.Progress()
			delta := count - lastCount
			fmt.Printf("\u001b[2A")
			fmt.Printf("\r\u001b[2K   Uniques = %d (%d/sec)\n", count-duplicates, delta)
			fmt.Printf("\r\u001b[2KDuplicates = %d\n", duplicates)
			stdout.Flush()
			lastCount = count
		case <-done:
			fmt.Printf("\u001b[2K")
			fmt.Printf("\u001b[1A")
			fmt.Printf("\u001b[2K")
			fmt.Printf("\u001b[2A")
			stdout.Flush()
			done <- true
			return
		}
	}
}

func index(bloomOutput string, conf *AutoConfig) ([]string, error) {
	indexes := []string{}
	indexTmpDir := filepath.Join(conf.TempDir, "indexer")
	for _, key := range conf.Index.Keys {
		fmt.Printf("\r\u001b[2K\rComputing %s index ...\u001b[s", key)
		output := path.Join(conf.TempDir, fmt.Sprintf("%s.idx", key))
		indexes = append(indexes, output)
		index, err := indexer.GetIndexer(bloomOutput, output, key, conf.Index.Workers, indexTmpDir, conf.Index.NoCleanup)
		if err != nil {
			return nil, err
		}

		done := make(chan bool)
		go indexProgress(index, done)
		err = index.Start()
		done <- true
		<-done
		if err != nil {
			return nil, err
		}

		fmt.Printf("\u001b[u done!\n")
	}
	if !conf.Index.NoCleanup {
		os.RemoveAll(indexTmpDir)
	}
	return indexes, nil
}

func indexProgress(index *indexer.Indexer, done chan bool) {
	lastCount := 0
	fmt.Println()
	for {
		select {
		case <-time.After(time.Second):
			count := index.Count()
			delta := count - lastCount
			fmt.Printf("\r\u001b[2KIndexed %d (%d/sec)", count, delta)
			lastCount = count
		case <-done:
			fmt.Printf("\r\u001b[2K")
			fmt.Printf("\u001b[1A")
			done <- true
			return
		}
	}
}

func sort(indexes []string, conf *AutoConfig) error {
	sortTmpDir := filepath.Join(conf.TempDir, "sorter")
	for _, index := range indexes {
		fmt.Printf("\r\u001b[2K\rSorting %s ...\u001b[s", path.Base(index))
		output := path.Join(conf.OutputDir, path.Base(index))
		sort, err := sorter.GetSorter(index, output, int(conf.Sort.MaxMemory),
			int(conf.Sort.MaxGoRoutines), sortTmpDir, conf.Sort.NoCleanup)
		if err != nil {
			return err
		}
		done := make(chan bool)
		go sortProgress(sort, done)
		sort.Start()
		done <- true
		<-done
		fmt.Printf("\u001b[u done!\n")
	}
	return nil
}

func sortProgress(sort *sorter.Sorter, done chan bool) {
	spin := 0
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	stdout := bufio.NewWriter(os.Stdout)
	stats := &runtime.MemStats{}
	started := time.Now()
	elapsed := time.Now().Sub(started)
	maxHeap := float64(0)
	fmt.Println()
	fmt.Println()
	for {
		select {
		case <-done:
			fmt.Printf("\u001b[2K\r")
			fmt.Printf("\u001b[1A\r")
			fmt.Printf("\u001b[2K\r")
			done <- true
			return
		case <-time.After(250 * time.Millisecond):
			fmt.Printf("\u001b[1A") // Move up one
			runtime.ReadMemStats(stats)
			if spin%10 == 0 {
				// Calculating time is kind of expensive, so update once per ~second
				elapsed = time.Now().Sub(started)
			}
			heapAllocGb := float64(stats.HeapAlloc) / float64(gb)
			if maxHeap < heapAllocGb {
				maxHeap = heapAllocGb
			}
			fmt.Printf("\u001b[2K\rGo routines: %d - Heap: %0.3fGb (Max: %0.3fGb) - Time: %v\n",
				runtime.NumGoroutine(), heapAllocGb, maxHeap, elapsed)
			status := sort.Status
			if status == sorter.StatusMerging {
				status = fmt.Sprintf("%s (%.2f%%)", status, sort.MergePercent)
				fmt.Printf("\u001b[2K\r %s %s ... ", frames[spin%10], status)
			} else if status == sorter.StatusSorting {
				status = fmt.Sprintf("%s tape %d of %d", status, sort.CurrentTapeIndex, len(sort.Tapes))
				fmt.Printf("\u001b[2K\r %s %s ... ", frames[spin%10], status)
			} else {
				fmt.Printf("\u001b[2K\r %s %s ... ", frames[spin%10], status)
			}
			spin++
			stdout.Flush()
		}
	}
}
