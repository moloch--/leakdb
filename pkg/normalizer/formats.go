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
	"errors"
	"regexp"
	"strings"
)

const (
	emailRegex    = "(^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]{2,63})"
	passwordRegex = "[^\\x00-\\x1F\\x80-\\x9F]*"

	colonNewline      = "colon-newline"
	semicolonNewline  = "semicolon-newline"
	whitespaceNewline = "whitespace-newline"
)

var (
	coloneNewlinePattern     = regexp.MustCompile(emailRegex + ":" + passwordRegex)
	semicolonNewlinePattern  = regexp.MustCompile(emailRegex + ";" + passwordRegex)
	whitespaceNewlinePattern = regexp.MustCompile(emailRegex + "[ \t]+" + passwordRegex)

	// Formats - Valid formats
	Formats = map[string]Format{
		colonNewline:      ColonNewline{},
		semicolonNewline:  SemicolonNewline{},
		whitespaceNewline: WhitespaceNewline{},
	}
)

// SupportedFormats - List of supported formats
func SupportedFormats() []string {
	keys := []string{}
	for key := range Formats {
		keys = append(keys, key)
	}
	return keys
}

// Format - Pair of line pattern and normalizer func
type Format interface {
	GetName() string
	GetPattern() *regexp.Regexp
	Normalize(line string) (string, string, string, string, error)
}

// ColonNewline - The colon/newline delimited format
type ColonNewline struct{}

// GetName - Return the format's name
func (cn ColonNewline) GetName() string {
	return colonNewline
}

// GetPattern - Return the format's name
func (cn ColonNewline) GetPattern() *regexp.Regexp {
	return coloneNewlinePattern
}

// Normalize - Normalize a line, return email, user, domain, password, error
func (cn ColonNewline) Normalize(line string) (string, string, string, string, error) {
	if !cn.GetPattern().MatchString(line) {
		return "", "", "", "", errors.New("Pattern mismatch")
	}
	linePieces := strings.Split(line, ":")
	if len(linePieces) != 2 {
		return "", "", "", "", errors.New("Line is missing field")
	}
	email := strings.ToLower(linePieces[0])
	password := linePieces[1]
	emailPieces := strings.Split(email, "@")
	return email, emailPieces[0], emailPieces[1], password, nil
}

// SemicolonNewline - The colon/newline delimited format
type SemicolonNewline struct{}

// GetName - Return the format's name
func (cn SemicolonNewline) GetName() string {
	return semicolonNewline
}

// GetPattern - Return the format's name
func (cn SemicolonNewline) GetPattern() *regexp.Regexp {
	return semicolonNewlinePattern
}

// Normalize - Normalize a line, return email, user, domain, password, error
func (cn SemicolonNewline) Normalize(line string) (string, string, string, string, error) {
	if !cn.GetPattern().MatchString(line) {
		return "", "", "", "", errors.New("Pattern mismatch")
	}
	linePieces := strings.Split(line, ";")
	if len(linePieces) != 2 {
		return "", "", "", "", errors.New("Line is missing field")
	}
	email := strings.ToLower(linePieces[0])
	password := linePieces[1]
	emailPieces := strings.Split(email, "@")
	return email, emailPieces[0], emailPieces[1], password, nil
}

// WhitespaceNewline - The colon/newline delimited format
type WhitespaceNewline struct{}

// GetName - Return the format's name
func (cn WhitespaceNewline) GetName() string {
	return whitespaceNewline
}

// GetPattern - Return the format's name
func (cn WhitespaceNewline) GetPattern() *regexp.Regexp {
	return whitespaceNewlinePattern
}

// Normalize - Normalize a line, return email, user, domain, password, error
func (cn WhitespaceNewline) Normalize(line string) (string, string, string, string, error) {
	if !cn.GetPattern().MatchString(line) {
		return "", "", "", "", errors.New("Pattern mismatch")
	}
	lineFields := strings.Fields(line)
	linePieces := []string{}
	for _, piece := range lineFields {
		if 0 < len(piece) {
			linePieces = append(linePieces, piece)
		}
	}
	if len(linePieces) != 2 {
		return "", "", "", "", errors.New("Line is missing field")
	}
	email := strings.ToLower(linePieces[0])
	password := linePieces[1]
	emailPieces := strings.Split(email, "@")
	return email, emailPieces[0], emailPieces[1], password, nil
}
