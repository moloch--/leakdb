package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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

	TLSCertificate string
	TLSKey         string
}

// StartTLS - Start TLS server
func (s *Server) StartTLS(host string, port uint16) {
	bind := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/", s.SearchHandler)
	err := http.ListenAndServeTLS(bind, s.TLSCertificate, s.TLSKey, nil)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

// Start - Start server
func (s *Server) Start(host string, port uint16) {
	bind := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/", s.SearchHandler)
	err := http.ListenAndServe(bind, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// SearchHandler - Process search requests
func (s *Server) SearchHandler(resp http.ResponseWriter, req *http.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		msg := fmt.Sprintf("Failed to read request body %s", err)
		http.Error(resp, msg, http.StatusBadRequest)
		return
	}
	query := &QuerySet{}
	err = json.Unmarshal(body, query)
	if err != nil {
		msg := fmt.Sprintf("Failed to decode request %s", err)
		http.Error(resp, msg, http.StatusBadRequest)
		return
	}

	resultSet := &ResultSet{}
	var results []searcher.Credential
	if query.Email != "" {
		results, err = s.emailSearch(query)
	} else if query.User != "" {
		results, err = s.userSearch(query)
	} else if query.Domain != "" {
		results, err = s.domainSearch(query)
	} else {
		err = errors.New("Invalid query: does not contain valid key")
	}
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
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
		msg := fmt.Sprintf("Failed to serialized result set %s", err)
		http.Error(resp, msg, http.StatusBadRequest)
	} else {
		resp.Write(data)
	}
}

func (s *Server) userSearch(query *QuerySet) ([]searcher.Credential, error) {
	if s.UserIndex == "" {
		return nil, errors.New("No user index file")
	}
	return searcher.Start(s.Messages, query.User, s.JSONFile, s.UserIndex)
}

func (s *Server) emailSearch(query *QuerySet) ([]searcher.Credential, error) {
	if s.EmailIndex == "" {
		return nil, errors.New("No email index file")
	}
	return searcher.Start(s.Messages, query.Email, s.JSONFile, s.EmailIndex)
}

func (s *Server) domainSearch(query *QuerySet) ([]searcher.Credential, error) {
	if s.DomainIndex == "" {
		return nil, errors.New("No domain index file")
	}
	return searcher.Start(s.Messages, query.Domain, s.JSONFile, s.DomainIndex)
}
