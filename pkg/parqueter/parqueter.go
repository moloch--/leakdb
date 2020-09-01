package parqueter

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
	"encoding/json"
	"fmt"
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

	fileReader := bufio.NewReader(f)
	c.LineNumber = 0
	for {
		line, err := fileReader.ReadBytes('\n')
		if err != io.EOF && err != nil {
			return err
		}
		if 0 < len(line) {
			entry := &normalizer.Entry{}
			err = json.Unmarshal(line, entry)
			if err != nil {
				return fmt.Errorf("JSON unmarshal: %s", err)
			}
			if err = parquetWriter.Write(entry); err != nil {
				return err
			}
			c.LineNumber++
		}
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
func NewConverter(target string, output string) (*Converter, error) {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return nil, fmt.Errorf("Output '%s' already exists", output)
	}
	return &Converter{
		target:       target,
		output:       output,
		RowGroupSize: DefaultRowGroupSize,
	}, nil
}
