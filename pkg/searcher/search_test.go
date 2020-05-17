package searcher

import (
	"testing"
)

const (
	smallJSON = "../../test/small-bloomed.json"
	largeJSON = "../../test/large-bloomed.json"

	smallEmailIndex = "../../test/small-email-sorted.idx"
	largeEmailIndex = "../../test/large-email-sorted.idx"

	smallUserIndex = "../../test/small-user-sorted.idx"
	largeUserIndex = "../../test/large-user-sorted.idx"

	smallDomainIndex = "../../test/small-domain-sorted.idx"
	largeDomainIndex = "../../test/large-domain-sorted.idx"
)

var (
	smallCreds = []*Credential{
		{
			Email:    "jfashion16@ebay.co.uk",
			User:     "jfashion16",
			Domain:   "ebay.co.uk",
			Password: "Q4MqeIEG",
		},
		{
			Email:    "mmathivath@gov.uk",
			User:     "mmathivath",
			Domain:   "gov.uk",
			Password: "unvPnAyz",
		},
		{
			Email:    "ebernardinellic@soup.io",
			User:     "ebernardinellic",
			Domain:   "soup.io",
			Password: "1omqKFWMF",
		},
	}

	largeCreds = []*Credential{
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

func TestSearchSmallEmail(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range smallCreds {
		results, err := Start(messages, cred.Email, smallJSON, smallEmailIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}
		if results[0].Email != cred.Email || results[0].Password != cred.Password {
			t.Error("Search returned wrong result")
			return
		}
	}
}

func TestSearchSmallUser(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range smallCreds {
		results, err := Start(messages, cred.User, smallJSON, smallUserIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}
		if results[0].User != cred.User || results[0].Password != cred.Password {
			t.Error("Search returned wrong result")
			return
		}
	}
}

func TestSearchSmallDomain(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range smallCreds {
		results, err := Start(messages, cred.Domain, smallJSON, smallDomainIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}
		if results[0].Domain != cred.Domain || results[0].Password != cred.Password {
			t.Error("Search returned wrong result")
			return
		}
	}
}

func TestSearchLargeEmail(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range largeCreds {
		results, err := Start(messages, cred.Email, largeJSON, largeEmailIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}
		if results[0].Email != cred.Email || results[0].Password != cred.Password {
			t.Error("Search returned wrong result")
			return
		}
	}
}

func TestSearchLargeUser(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range largeCreds {
		results, err := Start(messages, cred.User, largeJSON, largeUserIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}
		if results[0].User != cred.User || results[0].Password != cred.Password {
			t.Error("Search returned wrong result")
			return
		}
	}
}

func TestSearchLargeDomain(t *testing.T) {
	messages := logChannel(t)
	defer close(messages)
	for _, cred := range largeCreds {
		results, err := Start(messages, cred.Domain, largeJSON, largeDomainIndex)
		if err != nil {
			t.Errorf("Search failed %s", err)
			return
		}
		if len(results) < 1 {
			t.Error("Search returned zero results")
			return
		}

		// Should return more than one result
		found := false
		for _, result := range results {
			if result.Domain != cred.Domain || result.Password != cred.Password {
				found = true
			}
		}
		if !found {
			t.Error("Search returned wrong result")
			return
		}
	}
}
