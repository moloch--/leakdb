package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/moloch--/leakdb/pkg/searcher"
)

const (
	// BadRequest - HTTP Bad Request
	BadRequest = 400
)

var (
	hexPattern = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	b64Pattern = regexp.MustCompile(`^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`)
)

// QuerySet - A LeakDB query
type QuerySet struct {
	Email  string `json:"email"`
	Domain string `json:"domain"`
	User   string `json:"user"`
	Page   int    `json:"page"`
}

// Credential - A result credential
type Credential struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// IsBlank - Password appears to be blank
func (cred *Credential) IsBlank() bool {

	if len(cred.Password) == 0 {
		return true
	}

	// Some dumps contain hardcoded 'blank' values
	if cred.Password == "xxx" {
		return true
	}

	return false
}

// IsHash - Password *appears* to a hash
func (cred *Credential) IsHash() bool {

	// I'm not aware of any common hashes that would be less than 8 chars
	if len(cred.Password) < 8 {
		return false
	}

	// Min length based on hex-encoded MD5 or higher
	hexMatched := hexPattern.MatchString(cred.Password)
	if hexMatched && 32 <= len(cred.Password) {
		return true
	}

	// Min length based on b64-encoded MD5 (minus padding) or higher
	b64Matched := b64Pattern.MatchString(cred.Password)
	if b64Matched && 22 <= len(cred.Password) {
		return true
	}

	return false
}

// ResultSet - Result of a query
type ResultSet struct {
	Count   int          `json:"count"`
	Page    int          `json:"page"`
	Pages   int          `json:"pages"`
	Results []Credential `json:"results"`
}

// Server - A server object
type Server struct {
	Messages chan string

	JSONFile    string
	EmailIndex  string
	UserIndex   string
	DomainIndex string
}

// SearchHandler - Process search requests
func (s *Server) SearchHandler(resp http.ResponseWriter, req *http.Request) (int, error) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return BadRequest, fmt.Errorf("Failed to read request body %s", err)
	}
	query := &QuerySet{}
	err = json.Unmarshal(body, query)
	if err != nil {
		return BadRequest, fmt.Errorf("Failed to decode request body %s", err)
	}

	resultSet := &ResultSet{}
	var results []searcher.Credential
	if query.Email != "" {
		results, err = searcher.Start(s.Messages, query.Email, s.JSONFile, s.EmailIndex)
		if err != nil {
			return BadRequest, fmt.Errorf("Email search failed %s", err)

		}
	} else if query.User != "" {
		results, err = searcher.Start(s.Messages, query.User, s.JSONFile, s.UserIndex)
		if err != nil {
			return BadRequest, fmt.Errorf("User search failed %s", err)

		}
	} else if query.Domain != "" {
		results, err = searcher.Start(s.Messages, query.Domain, s.JSONFile, s.DomainIndex)
		if err != nil {
			return BadRequest, fmt.Errorf("Domain search failed %s", err)

		}
	} else {
		return BadRequest, errors.New("Invalid query: does not contain valid key/value")

	}

	resultSet.Page = 0
	resultSet.Pages = 1
	resultSet.Count = len(results)
	resultSet.Results = []Credential{}
	for _, result := range results {
		resultSet.Results = append(resultSet.Results, Credential{
			Email:    result.Email,
			Password: result.Password,
		})
	}
	data, err := json.Marshal(resultSet)
	if err != nil {
		return BadRequest, fmt.Errorf("Failed to serialized result set %s", err)
	}
	resp.Write(data)

	return 200, nil
}
