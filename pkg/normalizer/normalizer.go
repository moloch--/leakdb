package normalizer

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
	"io"
	"os"
)

// Normalize - Normalizer job
type Normalize struct {
	format     Format
	target     string
	output     string
	recursive  bool
	skipPrefix string
	skipSuffix string
}

// Start - Start the normalizer
func Start(format Format, target string, output string, recursive bool, skipPrefix, skipSuffix string) (*Normalize, error) {

	normalizer := &Normalize{
		format:     format,
		target:     target,
		output:     output,
		recursive:  recursive,
		skipPrefix: skipPrefix,
		skipSuffix: skipSuffix,
	}

	return normalizer, nil
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
