package main

/*
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
*/

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	pageSize = 1000
)

var (
	bigQueryMeta = BigQueryMeta{
		Table:       os.Getenv("BIGQUERY_TABLE"),
		ProjectID:   os.Getenv("BIGQUERY_PROJECT_ID"),
		Credentials: os.Getenv("BIGQUERY_CREDENTIALS"),
	}
)

// BigQueryMeta - Metadata need to query BigQuery
type BigQueryMeta struct {
	Table       string
	ProjectID   string
	Credentials string
}

// QuerySet - A set of base64 encoded password hashes to query with
type QuerySet struct {
	Domain string `json:"domain"`
	Email  string `json:"email"`
	User   string `json:"user"`
	Page   int    `json:"page"`
}

// Result - A single entry in leakdb
type Result struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ResultSet - The set of results from a QuerySet
type ResultSet struct {
	Count   int      `json:"count"`
	Page    int      `json:"page"`
	Pages   int      `json:"pages"`
	Results []Result `json:"results"`
}

// LeakDBDomainQuery - Query the LeakDB
func LeakDBDomainQuery(querySet QuerySet) (ResultSet, error) {
	log.Printf("Query domain = '%s'", querySet.Domain)
	return leakDBQuery(querySet.Domain, querySet.Page, "domain")
}

// LeakDBEmailQuery - Query the LeakDB
func LeakDBEmailQuery(querySet QuerySet) (ResultSet, error) {
	log.Printf("Query email = '%s'", querySet.Email)
	return leakDBQuery(querySet.Email, querySet.Page, "email")
}

// LeakDBUserQuery - Query the LeakDB
func LeakDBUserQuery(querySet QuerySet) (ResultSet, error) {
	log.Printf("Query user = '%s'", querySet.User)
	return leakDBQuery(querySet.User, querySet.Page, "user")
}

func leakDBQuery(queryParam string, queryPage int, field string) (ResultSet, error) {

	bigQueryCtx := context.Background()
	options := option.WithCredentialsJSON([]byte(bigQueryMeta.Credentials))
	bigQueryClient, err := bigquery.NewClient(bigQueryCtx, bigQueryMeta.ProjectID, options)
	if err != nil {
		log.Printf("[error] BigQuery client: %s", err)
		return ResultSet{}, err
	}

	rawQuery := fmt.Sprintf("SELECT email,password FROM `%s` WHERE %s = ?", bigQueryMeta.Table, field)
	query := bigQueryClient.Query(rawQuery)
	query.Parameters = []bigquery.QueryParameter{}
	query.Parameters = append(query.Parameters, bigquery.QueryParameter{
		Value: queryParam,
	})
	row, err := query.Read(bigQueryCtx)
	if err != nil {
		log.Printf("[error] Query '%s' failed: %v", field, err)
		return ResultSet{}, err
	}

	// Pagination logic
	lastPage := int(math.Ceil(float64(row.TotalRows)/pageSize)) - 1 // Zero index
	page := int(math.Abs(float64(queryPage)))                       // 'queryPage' is user controlled
	if lastPage < page {
		page = lastPage
	}
	start := page * pageSize
	stop := start + pageSize
	resultSet := ResultSet{
		Count:   int(row.TotalRows),
		Page:    page,
		Pages:   lastPage,
		Results: []Result{},
	}
	log.Printf("Results = %d, Page = %d, Pages = %d", resultSet.Count, resultSet.Page, resultSet.Pages)

	row.StartIndex = uint64(start)
	log.Printf("Starting at index %d", row.StartIndex)
	for position := start; position < stop; position++ {
		var value []bigquery.Value
		err = row.Next(&value)
		if err == iterator.Done {
			log.Printf("No more results")
			break
		}
		if err != nil {
			log.Printf("[error] Row iterator: %v", err)
			break
		}
		result := Result{
			Email:    fmt.Sprintf("%s", value[0]),
			Password: fmt.Sprintf("%s", value[1]),
		}
		resultSet.Results = append(resultSet.Results, result)
	}

	return resultSet, nil
}
