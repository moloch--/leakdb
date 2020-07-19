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
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Entry - A single entry
type Entry struct {
	Email    string `json:"email"`
	User     string `json:"user"`
	Domain   string `json:"domain"`
	Password string `json:"password"`
}

// Normalize - Normalizer job
type Normalize struct {
	Format     Format
	Targets    []string
	Output     *os.File
	Recursive  bool
	SkipPrefix string
	SkipSuffix string

	target      string
	targetCount int

	Errors []error
}

// GetStatus - Return the current target file and line number
func (n *Normalize) GetStatus() (string, int) {
	return n.target, n.targetCount
}

func (n *Normalize) lineQueue(lines chan<- string) {
	defer close(lines)
	for _, target := range n.Targets {
		if n.SkipPrefix != "" && strings.HasPrefix(target, n.SkipPrefix) {
			continue
		}
		if n.SkipSuffix != "" && strings.HasSuffix(target, n.SkipSuffix) {
			continue
		}
		err := n.normalizeFile(lines, target)
		if err != nil {
			n.Errors = append(n.Errors, err)
		}
	}
}

func (n *Normalize) normalizeFile(lines chan<- string, target string) error {
	file, err := os.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()

	n.target = target
	n.targetCount = 0
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if err == io.EOF {
			if 0 < len(line) {
				lines <- line
			}
			break
		}
		if err != nil {
			return err
		}
		n.targetCount++
		if 0 < len(line) {
			lines <- line
		}
	}
	return nil
}

// Start - Start the normalization process
func (n *Normalize) Start() {

	defer n.Output.Close()

	lines := make(chan string)
	go n.lineQueue(lines)

	for line := range lines {
		email, user, domain, password, err := n.Format.Normalize(line)
		if err != nil {
			continue
		}
		data, err := json.Marshal(&Entry{
			Email:    email,
			User:     user,
			Domain:   domain,
			Password: password,
		})
		if err != nil {
			panic(err)
		}
		n.Output.Write(data)
		n.Output.Write([]byte("\n"))
	}
}

// GetNormalizer - Start the normalizer
func GetNormalizer(format Format, target string, output string, recursive bool, skipPrefix, skipSuffix string) (*Normalize, error) {
	targets, err := getTargets(target, recursive)
	if err != nil {
		return nil, err
	}
	outputFile, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &Normalize{
		Format:     format,
		Targets:    targets,
		Output:     outputFile,
		Recursive:  recursive,
		SkipPrefix: skipPrefix,
		SkipSuffix: skipSuffix,
	}, nil
}

// getTargets - Get targets from target directory
func getTargets(target string, recursive bool) ([]string, error) {
	targetStat, err := os.Stat(target)
	if err != nil {
		return []string{}, err
	}
	targets := []string{}
	switch mode := targetStat.Mode(); {
	case mode.IsDir():
		if recursive {
			err = filepath.Walk(target, func(currentPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					targets = append(targets, currentPath)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			files, err := ioutil.ReadDir(target)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				if err != nil || file.IsDir() {
					continue
				}
				targets = append(targets, filepath.Join(target, file.Name()))
			}
		}
	case mode.IsRegular():
		targets = []string{target}
	}
	return targets, nil
}
