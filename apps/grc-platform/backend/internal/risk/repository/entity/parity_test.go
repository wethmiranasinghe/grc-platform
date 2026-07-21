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

// Parity tests compare each migrated repository against the MySQL
// implementation it replaces, calling both against live local services and
// asserting the results are identical. They are the primary check that a
// migration preserved behaviour.
//
// They need MySQL and the Compliance Entity running, so they skip unless
// RISK_PARITY=1 — CI and ordinary `go test ./...` runs stay green:
//
//	RISK_PARITY=1 go test -count=1 ./internal/risk/repository/entity/ -v
//
// -count=1 is required, not optional. Go's test cache keys on the test binary,
// the files read and the environment consulted — it cannot see HTTP responses.
// Without it these tests report "ok (cached)" without contacting the entity at
// all, so they would keep passing after an entity-side change broke parity.
//
// Run the whole package, not a single test: every repository migrated so far is
// re-verified, which is how a later entity-side change that breaks an earlier
// repository gets caught.
//
// These tests are deleted along with internal/risk/repository/mysql once the
// migration is complete — they have nothing left to compare against.
package entity

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wso2-open-operations/grc-tools/apps/grc-platform/backend/internal/shared/entityclient"
)

const (
	defaultDSN           = "root:root1234@tcp(localhost:3306)/grc_platform?parseTime=true"
	defaultEntityBaseURL = "http://localhost:8081"
)

// skipUnlessParity skips the calling test unless RISK_PARITY=1, so the suite is
// safe to run anywhere the local services are not up.
func skipUnlessParity(t *testing.T) {
	t.Helper()
	if os.Getenv("RISK_PARITY") != "1" {
		t.Skip("parity tests need live MySQL and Compliance Entity; set RISK_PARITY=1 to run")
	}
}

// parityDB opens the same database the MySQL repositories use. Override with
// PARITY_DB_DSN when the local database differs from the default.
func parityDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PARITY_DB_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db (is MySQL running?): %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// parityClient builds an entity client pointed at the local Compliance Entity.
// Override with PARITY_ENTITY_BASE_URL.
func parityClient(t *testing.T) *entityclient.Client {
	t.Helper()
	base := os.Getenv("PARITY_ENTITY_BASE_URL")
	if base == "" {
		base = defaultEntityBaseURL
	}
	return entityclient.New(base)
}

// derefString renders a *string for failure messages.
func derefString(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
