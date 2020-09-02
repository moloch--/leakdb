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
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	// Generic Errors
	errNoBody             = errors.New("No HTTP body")
	errFailedToParseBody  = errors.New("Failed to parse HTTP body")
	errEmptyQuerySet      = errors.New("Empty query set")
	errMethodNotSupported = errors.New("HTTP method not supported")
)

// LambdaError - Error mapped to JSON
type LambdaError struct {
	Error string `json:"error"`
}

// JSONError - Returns an error formatted as an APIGatewayProxyResponse
// if you try to return an actual error the API Gateway just swaps it
// for a generic 500 because why the fuck would you just expect an error
// to get returned to the client if you explicitly return it
// this is also sort of dangerous because we don't always know what the
// error message is going to say but you shouldn't expose this publically
func JSONError(err error) events.APIGatewayProxyResponse {
	msg, _ := json.Marshal(LambdaError{
		Error: fmt.Sprintf("%v", err),
	})
	return events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       string(msg),
	}
}

// RequestHandler - Handle an HTTP request
func RequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.Printf("Processing Lambda request %s\n", request.RequestContext.RequestID)

	// If no name is provided in the HTTP request body, throw an error
	if len(request.Body) < 1 {
		return JSONError(errNoBody), nil
	}

	if request.HTTPMethod != "POST" {
		return JSONError(errMethodNotSupported), nil
	}

	log.Printf("Parsing request body ...")
	var querySet QuerySet
	err := json.Unmarshal([]byte(request.Body), &querySet)
	if err != nil {
		return JSONError(errFailedToParseBody), nil
	}

	if len(querySet.Domain) == 0 && len(querySet.User) == 0 && len(querySet.Email) == 0 {
		return JSONError(errEmptyQuerySet), nil
	}

	var resultSet ResultSet
	if 0 < len(querySet.Domain) {
		resultSet, err = LeakDBDomainQuery(querySet)
		if err != nil {
			log.Printf("[error] %s", err)
			return JSONError(err), nil
		}
	} else if 0 < len(querySet.Email) {
		resultSet, err = LeakDBEmailQuery(querySet)
		if err != nil {
			log.Printf("[error] %s", err)
			return JSONError(err), nil
		}
	} else if 0 < len(querySet.User) {
		resultSet, err = LeakDBUserQuery(querySet)
		if err != nil {
			log.Printf("[error] %s", err)
			return JSONError(err), nil
		}
	}

	response, err := json.Marshal(resultSet)
	if err != nil {
		return JSONError(err), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(response),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RequestHandler)
}
