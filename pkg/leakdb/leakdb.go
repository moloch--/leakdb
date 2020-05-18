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
	"time"

	"github.com/moloch--/leakdb/api"
)

const (
	jsonContentType = "application/json"
	apiKeyHeader    = "x-api-key"
)

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
func (client *Client) Query(querySet *api.QuerySet) (*api.ResultSet, error) {

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
	resultSet := &api.ResultSet{}
	if err := json.Unmarshal(bodyBytes, resultSet); err != nil {
		return nil, err
	}
	return resultSet, nil
}

// QueryAll - Query for all results
func (client *Client) QueryAll(querySet api.QuerySet) (*api.ResultSet, error) {
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
