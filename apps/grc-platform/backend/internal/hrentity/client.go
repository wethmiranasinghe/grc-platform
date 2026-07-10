// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package hrentity is a read-only client for the WSO2 HR entity GraphQL
// service (hr_entity). It is used to look up employees for the Risk module's
// "Risk Identified By: Employee" field — employee data is never stored in
// the GRC platform's own database, only fetched live per search.
package hrentity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Employee is the subset of hr_entity's Employee fields the Risk module needs.
type Employee struct {
	FirstName string
	LastName  string
	WorkEmail string
}

// Client talks to the hr_entity GraphQL service. Pointed at a local mock
// server during development and the real Choreo-hosted service in
// production — only the URL changes, the request/response contract is
// identical either way.
type Client struct {
	graphqlURL string
	httpClient *http.Client
}

// NewClient creates a Client for the hr_entity service at graphqlURL.
func NewClient(graphqlURL string) *Client {
	return &Client{
		graphqlURL: graphqlURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

const employeesQuery = `
query SearchEmployees($filter: EmployeeFilter, $limit: Int) {
	employees(filter: $filter, limit: $limit) {
		firstName
		lastName
		workEmail
	}
}`

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlResponse struct {
	Data struct {
		Employees []struct {
			FirstName *string `json:"firstName"`
			LastName  *string `json:"lastName"`
			WorkEmail *string `json:"workEmail"`
		} `json:"employees"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// SearchActiveEmployees returns active WSO2 employees whose work email
// contains emailSearchString, capped at limit results. hr_entity's
// EmployeeFilter has no name-search field, so matching is by email only.
func (c *Client) SearchActiveEmployees(ctx context.Context, emailSearchString string, limit int) ([]Employee, error) {
	body, err := json.Marshal(graphqlRequest{
		Query: employeesQuery,
		Variables: map[string]any{
			"filter": map[string]any{
				"emailSearchString": emailSearchString,
				"employeeStatus":    []string{"Active"},
			},
			"limit": limit,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal hr entity request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build hr entity request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call hr entity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hr entity returned status %d", resp.StatusCode)
	}

	var gqlResp graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("decode hr entity response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("hr entity returned errors: %s", gqlResp.Errors[0].Message)
	}

	employees := make([]Employee, 0, len(gqlResp.Data.Employees))
	for _, e := range gqlResp.Data.Employees {
		var emp Employee
		if e.FirstName != nil {
			emp.FirstName = *e.FirstName
		}
		if e.LastName != nil {
			emp.LastName = *e.LastName
		}
		if e.WorkEmail != nil {
			emp.WorkEmail = *e.WorkEmail
		}
		employees = append(employees, emp)
	}
	return employees, nil
}
