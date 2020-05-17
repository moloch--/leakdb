package leakdb

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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	jsonContentType = "application/json"
	apiKeyHeader    = "x-api-key"
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

// Client - An HTTP client object
type Client struct {
	HTTPClient *http.Client
	URL        string
	APIToken   string
}

// ClientHTTPConfig - Configure the HTTP client
type ClientHTTPConfig struct {
	ProxyURL          string
	SkipTLSValidation bool
	Timeout           time.Duration // Timeout in seconds
}

// Query - Perform a LeakDB query
func (client *Client) Query(querySet *QuerySet) (*ResultSet, error) {

	body, err := json.Marshal(querySet)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, client.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set(apiKeyHeader, client.APIToken)

	response, err := client.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Non-200 response code (%d)", response.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	resultSet := &ResultSet{}
	if err := json.Unmarshal(bodyBytes, resultSet); err != nil {
		return nil, err
	}
	return resultSet, nil
}

// QueryAll - Query for all results
func (client *Client) QueryAll(querySet QuerySet) (*ResultSet, error) {
	return nil, nil
}

// NewClient - Init new client object with config
func NewClient(apiURL, apiToken string, config ClientHTTPConfig) (*Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.SkipTLSValidation,
		},
		TLSHandshakeTimeout: config.Timeout,
	}
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &Client{
		URL:      apiURL,
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
	}, nil
}
