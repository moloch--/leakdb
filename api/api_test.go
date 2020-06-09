package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	expected = `{"count":1,"page":0,"pages":1,"results":[{"email":"acirlosmg@nsw.gov.au","password":"avXtGXM"}]}`
)

func TestSearchHandler(t *testing.T) {
	null := make(chan string)
	go func() {
		for range null {
		}
	}()
	defer close(null)

	server := &Server{
		JSONFile:    "../test/large-bloomed.json",
		UserIndex:   "../test/large-user-sorted.idx",
		EmailIndex:  "../test/large-email-sorted.idx",
		DomainIndex: "../test/large-domain-sorted.idx",

		Messages: null,
	}
	handler := http.HandlerFunc(server.SearchHandler)
	email, _ := json.Marshal(&QuerySet{Email: "acirlosmg@nsw.gov.au"})
	user, _ := json.Marshal(&QuerySet{User: "acirlosmg"})
	domain, _ := json.Marshal(&QuerySet{Domain: "nsw.gov.au"})

	// Email Query
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(email))
	if err != nil {
		t.Error(err)
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	// User Query
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/", bytes.NewBuffer(user))
	if err != nil {
		t.Error(err)
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	// Domain Query
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/", bytes.NewBuffer(domain))
	if err != nil {
		t.Error(err)
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	result := &ResultSet{}
	err = json.Unmarshal([]byte(rr.Body.String()), result)
	if err != nil {
		t.Error(err)
	}
	if result.Count != 13 {
		t.Errorf("handler returned unexpected result: got %v want %v",
			result.Count, 13)
	}
}
