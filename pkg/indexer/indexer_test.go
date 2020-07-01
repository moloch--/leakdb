package indexer

import (
	"io/ioutil"
	"os"
	"testing"
)

func testIndex(t *testing.T, input string, key string, expectedSize int) {
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

	indexer, err := GetIndexer(input, output.Name(), key, 1, tempDir, false)
	if err != nil {
		t.Errorf("Index compute error: %s\n", err)
		return
	}
	indexer.Start()

	output.Seek(0, 0)
	fileInfo, err := output.Stat()
	if err != nil {
		t.Errorf("Output file state: %s\n", err)
		return
	}
	if fileInfo.Size()%entrySize != 0 {
		t.Errorf("Irregular output file modulo: %d\n", fileInfo.Size()%entrySize)
		return
	}
	if fileInfo.Size()/entrySize != int64(expectedSize) {
		t.Errorf("Irregular output file size: %d\n", fileInfo.Size()/entrySize)
		return
	}
}

func TestIndexerSmallEmail(t *testing.T) {
	testIndex(t, "../../test/small-bloomed.json", "email", 50)
}

func TestIndexerLargeEmail(t *testing.T) {
	testIndex(t, "../../test/large-bloomed.json", "email", 8000)
}

func TestIndexerSmallUser(t *testing.T) {
	testIndex(t, "../../test/small-bloomed.json", "user", 50)
}

func TestIndexerLargeUser(t *testing.T) {
	testIndex(t, "../../test/large-bloomed.json", "user", 8000)
}

func TestIndexerSmallDomain(t *testing.T) {
	testIndex(t, "../../test/small-bloomed.json", "domain", 50)
}

func TestIndexerLargeDomain(t *testing.T) {
	testIndex(t, "../../test/large-bloomed.json", "domain", 8000)
}
