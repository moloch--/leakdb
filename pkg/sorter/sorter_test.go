package sorter

import (
	"io/ioutil"
	"os"
	"testing"
)

const (
	maxGoRoutines = 1000
	maxMemory     = 2048
)

func testSort(t *testing.T, input string) {
	output, err := ioutil.TempFile("", "output.idx")
	if err != nil {
		t.Errorf("temp file error %s", err)
		return
	}
	defer os.Remove(output.Name())

	tempDir, err := ioutil.TempDir("", "leakdb_")
	if err != nil {
		t.Errorf("Temp error: %s\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	sorter, err := GetSorter(input, output.Name(), maxMemory, maxGoRoutines, tempDir, false)
	sorter.Start()
	if err != nil {
		t.Errorf("Sort error: %s\n", err)
		return
	}

	output.Seek(0, 0)
	sorted, err := CheckSort(output.Name(), false)
	if err != nil {
		t.Errorf("Check sort error: %s\n", err)
		return
	}
	if !sorted {
		t.Error("Failed to correctly sort index")
		return
	}
}

func TestSorterSmallEmail(t *testing.T) {
	testSort(t, "../../test/small-email-unsorted.idx")
}

func TestSorterSmallUser(t *testing.T) {
	testSort(t, "../../test/small-user-unsorted.idx")
}

func TestSorterSmallDomain(t *testing.T) {
	testSort(t, "../../test/small-domain-unsorted.idx")
}
