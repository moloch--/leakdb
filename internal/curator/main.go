package curator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	TempDir   string   `json:"temp_dir"`
}

// SortConfig - Sort configuration
type SortConfig struct {
	MaxGoRoutines uint   `json:"max_goroutines"`
	MaxMemory     uint   `json:"max_memory"`
	NoCleanup     bool   `json:"no_cleanup"`
	TempDir       string `json:"temp_dir"`
}

// AutoConfig - A complete config for the auto command
type AutoConfig struct {
	Bloom *BloomConfig `json:"bloom"`
	Index *IndexConfig `json:"index"`
	Sort  *SortConfig  `json:"sort"`

	InputDir  string `json:"input_dir"`
	OutputDir string `json:"output_dir"`
	TempDir   string `json:"temp_dir"`
	NoCleanup bool   `json:"no_cleanup"`
	Verbose   bool   `json:"verbose"`
}

func mainRun(cmd *cobra.Command, args []string) {
	generate, err := cmd.Flags().GetString(generateFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", generateFlagStr, err)
		return
	}
	if generate != "" {
		err := defaultConf(generate)
		if err != nil {
			fmt.Printf(Warn+"Failed to generate config %s\n", err)
		}
		return
	}

	confFlag, err := cmd.Flags().GetString(configFlagStr)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", configFlagStr, err)
		return
	}
	if confFlag == "" {
		fmt.Printf(Warn+"Missing --%s\n", configFlagStr)
		return
	}
	data, err := ioutil.ReadFile(confFlag)
	if err != nil {
		fmt.Printf(Warn+"Failed to read config %s\n", err)
		return
	}
	autoConf := &AutoConfig{}
	err = json.Unmarshal(data, autoConf)
	if err != nil {
		fmt.Printf(Warn+"Failed to parse config %s\n", err)
		return
	}

	err = auto(autoConf)
	if err != nil {
		fmt.Printf(Warn+"%s\n", err)
	}
}

func defaultConf(generate string) error {
	workers := uint(runtime.NumCPU())
	conf := &AutoConfig{
		InputDir:  "",
		OutputDir: "",
		TempDir:   "",
		Verbose:   false,
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
			TempDir:   "",
		},
		Sort: &SortConfig{
			MaxGoRoutines: 10000,
			MaxMemory:     2048,
			NoCleanup:     false,
			TempDir:       "",
		},
	}

	data, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(generate, data, 0600)
}

func auto(conf *AutoConfig) error {
	var err error
	// Check input & output locations
	stat, err := os.Stat(conf.InputDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("Input dir (%s) %s", conf.InputDir, err)
	}
	if !stat.IsDir() {
		return errors.New("input_dir must be a directory")
	}

	if _, err = os.Stat(conf.OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(conf.OutputDir, 0700)
		if err != nil {
			return err
		}
	}

	workDir := conf.TempDir
	if workDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		workDir, err = ioutil.TempDir(cwd, "leakdb_workdir_")
		if err != nil {
			return err
		}
	}

	// *** Bloom ***
	fmt.Printf("Applying bloom filter ...")
	bloomOutput := conf.Bloom.Output
	if bloomOutput == "" {
		bloomOutput = fmt.Sprintf("%s.json", path.Dir(conf.InputDir))
	}
	bloom, err := bloomer.GetBloomer(conf.InputDir, bloomOutput, conf.Bloom.FilterSave, conf.Bloom.FilterLoad,
		conf.Bloom.Workers, conf.Bloom.FilterSize, conf.Bloom.FilterHashes)
	if err != nil {
		return err
	}
	err = bloom.Start()
	if err != nil {
		return err
	}
	fmt.Printf("done!\n")

	// *** Index ***
	fmt.Printf("Computing indexes ...")
	indexes := []string{}
	for _, key := range conf.Index.Keys {
		tempDir := conf.Index.TempDir
		if tempDir == "" {
			tempDir, err = ioutil.TempDir("", "leakdb_")
			if err != nil {
				return err
			}
		}

		output := path.Join(workDir, fmt.Sprintf("%s-%s.idx", path.Dir(conf.InputDir), key))
		indexes = append(indexes, output)
		index, err := indexer.GetIndexer(bloomOutput, output, key, conf.Index.Workers, tempDir, conf.Index.NoCleanup)
		if err != nil {
			return err
		}
		index.Start()
	}
	if !conf.NoCleanup {
		for _, index := range indexes {
			defer os.Remove(index)
		}
	}
	fmt.Printf("done!\n")

	// *** Sort ***
	fmt.Printf("Sorting indexes ...")

	messages := make(chan string)
	defer close(messages)
	go func() {
		for message := range messages {
			if conf.Verbose {
				fmt.Printf(message)
			}
		}
	}()

	for _, index := range indexes {
		output := path.Join(conf.OutputDir, path.Base(index))
		tempDir := conf.Sort.TempDir
		if tempDir == "" {
			tempDir, err = ioutil.TempDir("", "leakdb_")
			if err != nil {
				return err
			}
		}
		sort, err := sorter.GetSorter(index, output, int(conf.Sort.MaxMemory), int(conf.Sort.MaxGoRoutines), tempDir, conf.Sort.NoCleanup)
		if err != nil {
			return err
		}
		sort.Start()
	}

	fmt.Printf("done!\n")
	return nil
}
