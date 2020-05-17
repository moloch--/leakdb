package bloomer

import (
	"bufio"
	"io/ioutil"
	"os"
	"testing"
)

func TestBloomerSmall(t *testing.T) {
	targets := []string{"../../test/small.json"}
	output, err := ioutil.TempFile("", "output-sm.json")
	if err != nil {
		t.Errorf("temp file error %s", err)
	}
	defer os.Remove(output.Name())

	// Bloom file
	err = Start(targets, output.Name(), "", "", 1, 1, 4)
	if err != nil {
		t.Errorf("Bloomer failed: %s", err)
	}

	output.Seek(0, 0)
	scanner := bufio.NewScanner(output)
	lines := []string{}
	t.Logf("Scanning %s ...", output.Name())
	for scanner.Scan() {
		line := scanner.Text()
		t.Log(line)
		if err := scanner.Err(); err != nil {
			t.Errorf("reading input: %s", err)
			return
		}
		if len(line) <= 1 {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) != 50 {
		t.Errorf("Bloomer did not return 50 lines as expected (%d)", len(lines))
		return
	}
}

func TestBloomerLarge(t *testing.T) {
	targets := []string{"../../test/large.json"}
	output, err := ioutil.TempFile("", "output-lg.json")
	if err != nil {
		t.Errorf("temp file error %s", err)
	}
	defer os.Remove(output.Name())

	// Bloom file
	err = Start(targets, output.Name(), "", "", 1, 1, 4)
	if err != nil {
		t.Errorf("Bloomer failed: %s", err)
	}

	output.Seek(0, 0)
	scanner := bufio.NewScanner(output)
	lines := []string{}
	t.Logf("Scanning %s ...", output.Name())
	for scanner.Scan() {
		line := scanner.Text()
		if err := scanner.Err(); err != nil {
			t.Errorf("reading input: %s", err)
			return
		}
		if len(line) <= 1 {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) != 8000 {
		t.Errorf("Bloomer did not return 8000 lines as expected (%d)", len(lines))
		return
	}
}
