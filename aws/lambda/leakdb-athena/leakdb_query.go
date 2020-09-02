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
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
)

const (
	emailColumn    = "email"
	userColumn     = "user"
	domainColumn   = "domain"
	passwordColumn = "password"

	// RUNNING - Running state of a query
	RUNNING = "RUNNING"
	// SUCCEEDED - Query successful state
	SUCCEEDED = "SUCCEEDED"

	// QUEUED - Query is queued status
	QUEUED = "QUEUED"
)

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

var (
	awsRegion      = getEnv("AWS_REGION", "us-west-2")
	athenaDatabase = getEnv("ATHENA_DATABASE", "leakdb")
	athenaTable    = getEnv("ATHENA_TABLE", "leakdb")
	s3ResultBucket = getEnv("S3_RESULT_BUCKET", "s3://leakdb-results")

	// WARNING: Since Athena does not support prepaired statements, nor provide
	// any escaping methods we're on our own. Because AWS hates its users and
	// the only thing worse than the API design are the docs.
	// Allowed: alpha-numerics, numbers, '.' and '@'
	matchWhitelist = regexp.MustCompile(`^[A-Za-z0-9@\.]+$`).MatchString

	// Returns if a parameter does not match the whitelist
	errInvalidParameter = errors.New("Parameter contains an invalid character")
)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryEmail - Query an email address
func QueryEmail(email string) (*ResultSet, error) {
	db := fmt.Sprintf("%s.%s", athenaDatabase, athenaTable)
	if matchWhitelist(email) {
		// See above, Athena does not provide any safe query method with user input like a prepared statement.
		return queryAthena(fmt.Sprintf("SELECT email, password FROM %s WHERE email = '%s'", db, email))
	}
	return nil, errInvalidParameter
}

// QueryDomain - Query an domain
func QueryDomain(domain string) (*ResultSet, error) {
	db := fmt.Sprintf("%s.%s", athenaDatabase, athenaTable)
	if matchWhitelist(domain) {
		// See above, Athena does not provide any safe query method with user input like a prepared statement.
		return queryAthena(fmt.Sprintf("SELECT email, password FROM %s WHERE domain = '%s'", db, domain))
	}
	return nil, errInvalidParameter
}

// QueryUser - Query an User
func QueryUser(user string) (*ResultSet, error) {
	db := fmt.Sprintf("%s.%s", athenaDatabase, athenaTable)
	if matchWhitelist(user) {
		// See above, Athena does not provide any safe query method with user input like a prepared statement.
		return queryAthena(fmt.Sprintf("SELECT email, password FROM %s WHERE user = '%s'", db, user))
	}
	return nil, errInvalidParameter
}

func queryAthena(query string) (*ResultSet, error) {
	awsCfg := &aws.Config{}
	awsCfg.WithRegion(awsRegion)

	awsSession := session.Must(session.NewSession(awsCfg))
	svc := athena.New(awsSession, aws.NewConfig().WithRegion(awsRegion))
	var queryInput athena.StartQueryExecutionInput
	queryInput.SetQueryString(query)

	var queryExec athena.QueryExecutionContext
	queryExec.SetDatabase(athenaDatabase)
	queryInput.SetQueryExecutionContext(&queryExec)

	var resultCfg athena.ResultConfiguration
	resultCfg.SetOutputLocation(s3ResultBucket)
	queryInput.SetResultConfiguration(&resultCfg)

	startResponse, err := svc.StartQueryExecution(&queryInput)
	if err != nil {
		return nil, err
	}
	log.Printf("StartQueryExecution result: %s", startResponse.GoString())

	var qri athena.GetQueryExecutionInput
	qri.SetQueryExecutionId(*startResponse.QueryExecutionId)

	var queryOutput *athena.GetQueryExecutionOutput

	for {
		queryOutput, err = svc.GetQueryExecution(&qri)
		if err != nil {
			return nil, err
		}
		queryState := *queryOutput.QueryExecution.Status.State
		log.Printf("Query state is %s\n", queryState)
		if queryState != RUNNING && queryState != QUEUED {
			break
		}
		time.Sleep(time.Duration(250) * time.Millisecond)
	}

	if *queryOutput.QueryExecution.Status.State == SUCCEEDED {
		// Athena has one of the worst APIs I've ever seen.
		var resultsRequest athena.GetQueryResultsInput
		resultsRequest.SetQueryExecutionId(*startResponse.QueryExecutionId)
		cursor, err := svc.GetQueryResults(&resultsRequest)
		if err != nil {
			return nil, err
		}
		resultSet := &ResultSet{
			Count:   len(cursor.ResultSet.Rows) - 1,
			Results: []Result{},
		}
		// The first row is always the column names for some dumb reason
		for _, row := range cursor.ResultSet.Rows[1:] {
			resultSet.Results = append(resultSet.Results, Result{
				Email:    *row.Data[0].VarCharValue,
				Password: *row.Data[1].VarCharValue,
			})
		}
		return resultSet, nil
	}
	queryFailed := *queryOutput.QueryExecution.Status.State
	return nil, fmt.Errorf("Query completed in %s state", queryFailed)
}
