package api

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
	server := &Server{
		JSONFile:    "../test/large-bloomed.json",
		UserIndex:   "../test/large-user-sorted.idx",
		EmailIndex:  "../test/large-email-sorted.idx",
		DomainIndex: "../test/large-domain-sorted.idx",
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
		t.Errorf("handler returned unexpected result: got %d want %d",
			result.Count, 13)
	}
}
