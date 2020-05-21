package normalizer

import "testing"

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

func TestGetTargets(t *testing.T) {
	files, err := getTargets("../../test/a", false)
	if err != nil {
		t.Error(err)
		return
	}
	if len(files) != 1 {
		t.Errorf("Unexpected number of targets %d", len(files))
		return
	}
	if files[0] != "../../test/a/a.txt" {
		t.Errorf("Unexpected target '%s'", files[0])
		return
	}

	files, err = getTargets("../../test/a", true)
	if err != nil {
		t.Error(err)
		return
	}
	if len(files) != 2 {
		t.Errorf("Unexpected number of recursive targets %d: %v", len(files), files)
		return
	}

}
