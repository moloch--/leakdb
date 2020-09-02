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
	"fmt"
	"os"
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
)

var (
	awsRegion      = getEnv("AWS_REGION", "us-west-2")
	athenaDatabase = getEnv("ATHENA_DATABASE", "leakdb")
	athenaTable    = getEnv("ATHENA_TABLE", "leakdb")
	s3ResultBucket = getEnv("S3_RESULT_BUCKET", "s3://leakdb-results")
)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryEmail - Query an email address
func QueryEmail(email string) ([]string, error) {
	db := fmt.Sprintf("%s.%s", athenaDatabase, athenaTable)
	return queryAthena(email, fmt.Sprintf("SELECT email,password FROM %s WHERE email = ?", db))
}

func queryAthena(value string, query string) ([]string, error) {
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

	result, err := svc.StartQueryExecution(&queryInput)
	if err != nil {
		return nil, err
	}
	fmt.Println("StartQueryExecution result:")
	fmt.Println(result.GoString())

	var qri athena.GetQueryExecutionInput
	qri.SetQueryExecutionId(*result.QueryExecutionId)

	var queryOutput *athena.GetQueryExecutionOutput

	for {
		fmt.Println("polling results ...")
		queryOutput, err = svc.GetQueryExecution(&qri)
		if err != nil {
			return nil, err
		}
		if *queryOutput.QueryExecution.Status.State != RUNNING {
			break
		}
		time.Sleep(time.Duration(250) * time.Millisecond)
	}

	if *queryOutput.QueryExecution.Status.State == SUCCEEDED {
		var ip athena.GetQueryResultsInput
		ip.SetQueryExecutionId(*result.QueryExecutionId)
		op, err := svc.GetQueryResults(&ip)
		if err != nil {
			return nil, err
		}
		fmt.Printf("%+v", op)
	} else {
		fmt.Println(*queryOutput.QueryExecution.Status.State)
	}

	return []string{}, nil
}
