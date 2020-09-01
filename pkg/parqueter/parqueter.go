package parquet

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	"github.com/moloch--/leakdb/pkg/normalizer"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
)

const (
	// DefaultRowGroupSize - Default row group size (128Mb)
	DefaultRowGroupSize = 128 * 1024 * 1024
)

// Converter - Convert JSON entry to parquet format
type Converter struct {
	target string
	output string
	lines  chan string

	RowGroupSize int64
	LineNumber   int
}

// Start - Convert target to parquet
func (c *Converter) Start() error {
	fw, err := local.NewLocalFileWriter(c.output)
	if err != nil {
		return err
	}

	parquetWriter, err := writer.NewParquetWriter(fw, new(normalizer.Entry), 4)
	if err != nil {
		return err
	}
	parquetWriter.RowGroupSize = c.RowGroupSize
	parquetWriter.CompressionType = parquet.CompressionCodec_SNAPPY

	f, err := os.Open(c.target)
	defer f.Close()
	if err != nil {
		return err
	}

	rd := bufio.NewReader(f)
	c.LineNumber = 0
	for {
		line, err := rd.ReadBytes('\n')
		if err != io.EOF && err != nil {
			return err
		}
		entry := &normalizer.Entry{}
		err = json.Unmarshal(line, entry)
		if err != nil {
			return err
		}
		if err = parquetWriter.Write(entry); err != nil {
			return err
		}
		c.LineNumber++
		if err == io.EOF {
			break
		}
	}

	if err = parquetWriter.WriteStop(); err != nil {
		return err
	}
	fw.Close()
	return nil
}

// NewConverter - Create new converter
func NewConverter(target string, output string) *Converter {
	return &Converter{
		target:       target,
		output:       output,
		RowGroupSize: DefaultRowGroupSize,
	}
}

func lineReader()
