package curator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"

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
	// defer os.RemoveAll(autoConf.TempDir)

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
	var err error
	// Check input & output locations
	_, err = os.Stat(conf.Input)
	if os.IsNotExist(err) {
		return fmt.Errorf("Input %s %s", conf.Input, err)
	}

	// *** Bloom ***
	fmt.Printf("Applying bloom filter ...")
	bloomOutput := conf.Bloom.Output
	if bloomOutput == "" {
		bloomOutput = filepath.Join(conf.TempDir, "bloomed.json")
	}
	bloom, err := bloomer.GetBloomer(conf.Input, bloomOutput, conf.Bloom.FilterSave,
		conf.Bloom.FilterLoad, conf.Bloom.Workers, conf.Bloom.FilterSize, conf.Bloom.FilterHashes)
	if err != nil {
		return err
	}
	err = bloom.Start()
	if err != nil {
		return err
	}
	fmt.Printf("done!\n")

	// *** Index ***
	indexes := []string{}
	indexTmpDir := filepath.Join(conf.TempDir, "indexer")
	for _, key := range conf.Index.Keys {
		fmt.Printf("Computing %s index ...", key)
		output := path.Join(conf.TempDir, fmt.Sprintf("%s-unsorted.idx", key))
		indexes = append(indexes, output)
		index, err := indexer.GetIndexer(bloomOutput, output, key, conf.Index.Workers, indexTmpDir, conf.Index.NoCleanup)
		if err != nil {
			return err
		}
		err = index.Start()
		if err != nil {
			return err
		}
		fmt.Printf("done!\n")
	}
	if !conf.Index.NoCleanup {
		os.RemoveAll(indexTmpDir)
	}

	// *** Sort ***
	sortTmpDir := filepath.Join(conf.TempDir, "sorter")
	for _, index := range indexes {
		fmt.Printf("Sorting %s ...", index)
		output := path.Join(conf.OutputDir, path.Base(index))
		sort, err := sorter.GetSorter(index, output, int(conf.Sort.MaxMemory),
			int(conf.Sort.MaxGoRoutines), sortTmpDir, conf.Sort.NoCleanup)
		if err != nil {
			return err
		}
		sort.Start()
		fmt.Printf("done!\n")
	}
	return nil
}
