package leakdb

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moloch--/leakdb/pkg/searcher"
)

const (
	largeJSON = "../../test/large-bloomed.json"

	largeEmailIndex  = "../../test/large-email-sorted.idx"
	largeUserIndex   = "../../test/large-user-sorted.idx"
	largeDomainIndex = "../../test/large-domain-sorted.idx"
)

var (
	largeCreds = []*searcher.Credential{
		{
			Email:    "edengatenu@shutterfly.com",
			User:     "edengatenu",
			Domain:   "shutterfly.com",
			Password: "EMPvdd",
		},
		{
			Email:    "mhagwoodjr@princeton.edu",
			User:     "mhagwoodjr",
			Domain:   "princeton.edu",
			Password: "JqWEST",
		},
		{
			Email:    "dfrowdelw@sun.com",
			User:     "dfrowdelw",
			Domain:   "sun.com",
			Password: "JJJW2AS",
		},
	}
)

func logChannel(t *testing.T) chan string {
	messages := make(chan string)
	go func() {
		for msg := range messages {
			t.Log(msg)
		}
	}()
	return messages
}

func getTestServer(t *testing.T, messages chan string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("Failed to read request body %s", err)
			return
		}
		query := &QuerySet{}
		err = json.Unmarshal(body, query)
		if err != nil {
			t.Errorf("Failed to decode request body %s", err)
			return
		}

		resultSet := &ResultSet{}
		var results []searcher.Credential
		if query.Email != "" {
			results, err = searcher.Start(messages, query.Email, largeJSON, largeEmailIndex)
			if err != nil {
				t.Errorf("Email search failed %s", err)
				return
			}
		} else if query.User != "" {
			results, err = searcher.Start(messages, query.User, largeJSON, largeUserIndex)
			if err != nil {
				t.Errorf("User search failed %s", err)
				return
			}
		} else if query.Domain != "" {
			results, err = searcher.Start(messages, query.Domain, largeJSON, largeDomainIndex)
			if err != nil {
				t.Errorf("Domain search failed %s", err)
				return
			}
		} else {
			t.Error("Invalid query: does not contain valid key/value")
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
			t.Errorf("Failed to serialized result set %s", err)
			return
		}
		resp.Write(data)
	}))
}

func TestLeakDBEmailQuery(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)

	server := getTestServer(t, messages)
	defer server.Close()

	client := &Client{
		HTTPClient: server.Client(),
		URL:        server.URL,
		APIToken:   "foobar",
	}

	for _, cred := range largeCreds {
		query := &QuerySet{Email: cred.Email}
		resultSet, err := client.Query(query)
		if err != nil {
			t.Errorf("Client response error %s", err)
			return
		}

		found := false
		for _, result := range resultSet.Results {
			if result.Email == cred.Email && result.Password == result.Password {
				found = true
			}
		}
		if !found {
			t.Error("API response contained an incorrect result")
			return
		}
	}
}

func TestLeakDBUserQuery(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)

	server := getTestServer(t, messages)
	defer server.Close()

	client := &Client{
		HTTPClient: server.Client(),
		URL:        server.URL,
		APIToken:   "foobar",
	}

	for _, cred := range largeCreds {
		query := &QuerySet{User: cred.User}
		resultSet, err := client.Query(query)
		if err != nil {
			t.Errorf("Client response error %s", err)
			return
		}

		found := false
		for _, result := range resultSet.Results {
			if result.Email == cred.Email && result.Password == result.Password {
				found = true
			}
		}
		if !found {
			t.Error("API response contained an incorrect result")
			return
		}
	}
}

func TestLeakDBDomainQuery(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)

	server := getTestServer(t, messages)
	defer server.Close()

	client := &Client{
		HTTPClient: server.Client(),
		URL:        server.URL,
		APIToken:   "foobar",
	}

	for _, cred := range largeCreds {
		query := &QuerySet{Domain: cred.Domain}
		resultSet, err := client.Query(query)
		if err != nil {
			t.Errorf("Client response error %s", err)
			return
		}

		found := false
		for _, result := range resultSet.Results {
			if result.Email == cred.Email && result.Password == result.Password {
				found = true
			}
		}
		if !found {
			t.Error("API response contained an incorrect result")
			return
		}
	}
}
