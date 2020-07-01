package curator

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
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/moloch--/leakdb/pkg/sorter"
	"github.com/spf13/cobra"
)

const (
	maxGoRoutines = 10000
)

var sortCmd = &cobra.Command{
	Use:   "sort",
	Short: "Sort an index file",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		index, err := cmd.Flags().GetString(indexFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", indexFlagStr, err)
			return
		}
		output, err := cmd.Flags().GetString(outputFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", outputFlagStr, err)
			return
		}
		if output == "" {
			output = fmt.Sprintf("%s_sorted.idx", index)
		}
		maxMemory, err := cmd.Flags().GetUint(maxMemoryFlagStr)
		if maxMemory < 1 {
			maxMemory = 1
		}
		noCleanup, err := cmd.Flags().GetBool(noCleanupFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", noCleanupFlagStr, err)
			return
		}
		tempDir, err := cmd.Flags().GetString(tempDirFlagStr)
		if err != nil {
			fmt.Printf(Warn+"Failed to parse --%s flag: %s\n", tempDirFlagStr, err)
			return
		}
		if tempDir == "" {
			cwd, _ := os.Getwd()
			tempDir, err = ioutil.TempDir(cwd, ".leakdb_")
			if err != nil {
				fmt.Printf(Warn+"Temp error: %s\n", err)
				return
			}
		}
		if !noCleanup {
			defer os.RemoveAll(tempDir)
		}

		sort, err := sorter.GetSorter(index, output, int(maxMemory), maxGoRoutines, tempDir, noCleanup)
		if err != nil {
			fmt.Printf(Warn+"%s\n", err)
			return
		}
		sort.Start()
	},
}

func progress(start time.Time, messages chan string, done chan bool) {
	spin := 0
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	stdout := bufio.NewWriter(os.Stdout)
	stats := &runtime.MemStats{}
	elapsed := time.Now().Sub(start)
	maxHeap := float64(0)
	fmt.Println()
	for {
		select {
		case <-done:
			fmt.Printf("\u001b[1A") // Move up one
			fmt.Println("\u001b[2K\rSorting completed!")
			defer func() { done <- true }()
			return
		case msg := <-messages:
			fmt.Printf("\u001b[1A")
			fmt.Printf("\u001b[2K\r%s\n\n", msg)
		case <-time.After(100 * time.Millisecond):
			fmt.Printf("\u001b[1A") // Move up one
			runtime.ReadMemStats(stats)
			if spin%10 == 0 {
				// Calculating time is kind of expensive, so update once per ~second
				elapsed = time.Now().Sub(start)
			}
			heapAllocGb := float64(stats.HeapAlloc) / float64(gb)
			if maxHeap < heapAllocGb {
				maxHeap = heapAllocGb
			}
			fmt.Printf("\u001b[2K\rGo routines: %d - Heap: %0.3fGb (Max: %0.3fGb) - Time: %v\n",
				runtime.NumGoroutine(), heapAllocGb, maxHeap, elapsed)
			fmt.Printf("\u001b[2K\rSorting, please wait ... %s ",
				frames[spin%10])
			spin++
			stdout.Flush()
		}
	}
}
