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

var (
	colonNewlineData = []string{
		"foo@bar.com:hunter2",
		"foo2@bar.com:password",
		"foo3@baz.com:monkey12",
	}
	semicolonNewlineData = []string{
		"foo@bar.com;hunter2",
		"foo2@bar.com;password",
		"foo3@baz.com;monkey12",
	}
	whitespaceNewlineData = []string{
		"foo@bar.com hunter2",
		"foo2@bar.com   password",
		"foo3@baz.com\tmonkey12",
	}
)

func TestColonNewline(t *testing.T) {
	cn := ColonNewline{}
	email, user, domain, password, err := cn.Normalize(colonNewlineData[0])
	if err != nil {
		t.Error(err)
	}
	if email != "foo@bar.com" || user != "foo" || domain != "bar.com" || password != "hunter2" {
		t.Error("Failed to parse line correctly")
	}
	for _, line := range colonNewlineData {
		_, _, _, _, err := cn.Normalize(line)
		if err != nil {
			t.Error(err)
		}
	}
	_, _, _, _, err = cn.Normalize(semicolonNewlineData[0])
	if err == nil {
		t.Error("Matched invalid line")
	}
}

func TestSemicolonNewline(t *testing.T) {
	sn := SemicolonNewline{}
	email, user, domain, password, err := sn.Normalize(semicolonNewlineData[0])
	if err != nil {
		t.Error(err)
	}
	if email != "foo@bar.com" || user != "foo" || domain != "bar.com" || password != "hunter2" {
		t.Error("Failed to parse line correctly")
	}
	for _, line := range semicolonNewlineData {
		_, _, _, _, err := sn.Normalize(line)
		if err != nil {
			t.Error(err)
		}
	}
	_, _, _, _, err = sn.Normalize(colonNewlineData[0])
	if err == nil {
		t.Error("Matched invalid line")
	}
}

func TestWhitespaceNewline(t *testing.T) {
	wn := WhitespaceNewline{}
	email, user, domain, password, err := wn.Normalize(whitespaceNewlineData[0])
	if err != nil {
		t.Error(err)
	}
	if email != "foo@bar.com" || user != "foo" || domain != "bar.com" || password != "hunter2" {
		t.Error("Failed to parse line correctly")
	}
	for _, line := range whitespaceNewlineData {
		_, _, _, _, err := wn.Normalize(line)
		if err != nil {
			t.Error(err)
		}
	}
	_, _, _, _, err = wn.Normalize(colonNewlineData[0])
	if err == nil {
		t.Error("Matched invalid line")
	}
}
