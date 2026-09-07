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

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// localTestCSV is a byte-for-byte copy of
// "Risk_Test_Full.xlsx - New Risk Form Structure.csv" in the planning-docs area
// — the hand-cleaned register used for a local -scim-snapshot run. This test is
// the regression lock that it stays fully valid: 5 rows, zero REJECTs, the
// dates / aliases / buckets all landing where the runbook says.
const localTestCSV = `Year,Quarter,Source Register,Risk Title,Risk Description,Security Compliance Reference ,Risk Category,Risk Identified By,Select Employee/ Name of External Person/ Tool,Risk Identified Date,Risk Assigned To,Likelihood,Impact,Impact Description,Implementation Date,Reassessment Date,Assignment Team,Risk Owner,Management Approver,Action Owner,Action Plan Description,Action Steps,Treatment Strategy,Progress,Git Issue URL,Email Subject,Remarks,Exidence Attachments,Workflow Status ,Migration ID
2025,Q3,Asgardeo,Supplier contract renewal delay,Delay in renewing supplier contracts may cause service interruption and unplanned cost increases.,HIPPA,Access Control & Credentials,Employee,Tharushi Pushpakumara,"January 10, 2025",ganguli.pi@wso2.com,3,2,Service interruption and unplanned cost increases for the organisation.,"June 30, 2025",30th Sep 2025,Business,tharushi.pu@wso2.com,asela.ja@wso2.com,atheeq.ra@wso2.com,Targeting completion by end of 2025,"Action Item -1
Action 2",Accept,WIP,https://github.com/wso2/product-risk/issues/101,Risk identification on threat models,Raised during the Q3 threat-model review.,,IN_REMEDIATION,1
2025,Q3,Asgardeo,Employee security training gap,Absence of regular security awareness training increases susceptibility to phishing and social engineering.,BUSINESS,"Logging, Monitoring & Detection",Employee,Asela Jayatilleke,"February 5, 2025",nikolya.fe@wso2.com,2,3,"Higher susceptibility to phishing, social engineering and insider threats.","September 30, 2025",31st Dec 2025,Digi Ops,tharushi.pu@wso2.com,asela.ja@wso2.com,atheeq.ra@wso2.com,Targeting completion by end of 2026,"Action Item -1
Action 2",Transfer,WIP,,Hackathon findings - ABC,,,CLOSED,2
2025,Q4,Choreo,Business continuity plan not tested,Without regular BCP drills the organisation cannot validate critical operations during a disruption.,ISO,Access Control & Credentials,Employee,Atheeq Rahman,"March 15, 2025",nikolya.fe@wso2.com,3,2,Cannot validate the ability to maintain critical operations during a major incident.,2025-12-31,30th Jun 2026,Choreo,tharushi.pu@wso2.com,asela.ja@wso2.com,atheeq.ra@wso2.com,Targeting completion by end of 2027,"Action Item -1
Action 2",Remediate,WIP,,Risk identification on threat models,,,IN_REMEDIATION,3
2026,Q1,Clever Care,Regulatory data localisation requirement,Emerging regulations may mandate that customer data be stored within specific geographic regions.,BUSINESS,Data Exposure & Privacy (PII),Employee,Asela Jayatilleke,"April 1, 2025",ganguli.pi@wso2.com,3,1,Significant infrastructure changes and additional compliance effort.,"March 31, 2026",30th Jun 2026,Digi Ops,tharushi.pu@wso2.com,asela.ja@wso2.com,atheeq.ra@wso2.com,Targeting completion by end of 2028,"Action Item -1
Action 2",Avoid,WIP,,Hackathon findings - ABC,,,CLOSED,4
2026,Q3,Business,Payment gateway single point of failure,Exclusive reliance on a single payment gateway provider risks halting revenue collection on any outage.,SOC2,Process & Documentation Gaps,Employee,Tharushi Pushpakumara,"May 20, 2025",nikolya.fe@wso2.com,3,3,Any provider downtime directly halts revenue collection and customer transactions.,2025-10-31,31st Dec 2025,Asgardeo,tharushi.pu@wso2.com,asela.ja@wso2.com,atheeq.ra@wso2.com,Targeting completion by end of 2029,"Action Item -1
Action 2",Accept,WIP,,External report,,,IN_REMEDIATION,5
`

func TestLocalTestCSV_ParsesCleanAndFullyResolves(t *testing.T) {
	rd := fixtureRefData(t)

	rows, fs, err := parseSheet(strings.NewReader(localTestCSV), rd)
	if err != nil {
		t.Fatalf("parseSheet: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	for _, f := range fs {
		if f.Severity == SevReject {
			t.Errorf("parse REJECT on a supposedly-clean row: %+v", f)
		}
	}

	wantBucket := map[int]string{1: "IN_REMEDIATION", 2: "CLOSED", 3: "IN_REMEDIATION", 4: "CLOSED", 5: "IN_REMEDIATION"}
	for _, r := range rows {
		if r.WorkflowStatus != wantBucket[r.MigrationID] {
			t.Errorf("row %d bucket = %q, want %q", r.MigrationID, r.WorkflowStatus, wantBucket[r.MigrationID])
		}
	}

	r1 := rows[0]
	if r1.RiskIdentifiedDate != "2025-01-10" || r1.ImplementationDate != "2025-06-30" || r1.ReassessmentDate != "2025-09-30" {
		t.Errorf("row 1 dates: id=%q impl=%q re=%q", r1.RiskIdentifiedDate, r1.ImplementationDate, r1.ReassessmentDate)
	}
	if len(r1.ComplianceRefIDs) != 1 || r1.ComplianceRefIDs[0] != 42 {
		t.Errorf("row 1 HIPPA should alias to HIPAA (id 42): %v", r1.ComplianceRefIDs)
	}
	if r1.GitIssueURL != "https://github.com/wso2/product-risk/issues/101" {
		t.Errorf("row 1 real git URL should survive: %q", r1.GitIssueURL)
	}
	if rows[3].TreatmentStrategy != "AVOID" {
		t.Errorf("row 4 'Avoid' should map to AVOID: %q", rows[3].TreatmentStrategy)
	}
	if r4 := rows[3]; r4.ImplementationDate != "2026-03-31" {
		t.Errorf(`row 4 "March 31, 2026" should parse: %q`, r4.ImplementationDate)
	}

	// Resolve against the snapshot file that ships beside the CSV.
	res := NewResolver(nil, snapshotFixtureUsers(), true)
	res.Build()
	rfs, err := res.Apply(context.Background(), rows)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	for _, f := range rfs {
		if f.Severity == SevReject {
			t.Errorf("resolve REJECT — every person should resolve: %+v", f)
		}
	}
}

func snapshotFixtureUsers() []DirectoryUser {
	return []DirectoryUser{
		{Email: "ganguli.pi@wso2.com", UUID: "11111111-1111-4111-8111-111111111111"},
		{Email: "nikolya.fe@wso2.com", UUID: "22222222-2222-4222-8222-222222222222"},
		{Email: "tharushi.pu@wso2.com", UUID: "33333333-3333-4333-8333-333333333333"},
		{Email: "asela.ja@wso2.com", UUID: "44444444-4444-4444-8444-444444444444"},
		{Email: "atheeq.ra@wso2.com", UUID: "55555555-5555-4555-8555-555555555555"},
	}
}

func TestLoadSnapshotFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.csv")
	body := "email,uuid\n" +
		"Alice.Ex@wso2.com , uuid-alice \n" +
		"\n" + // blank line tolerated
		"bob.ex@wso2.com,uuid-bob,ignored-trailing-column\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	users, err := loadSnapshotFile(path)
	if err != nil {
		t.Fatalf("loadSnapshotFile: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2 (header + blank line skipped): %+v", len(users), users)
	}
	if users[0].Email != "alice.ex@wso2.com" || users[0].UUID != "uuid-alice" {
		t.Errorf("row 0 not trimmed/lowercased: %+v", users[0])
	}
	if users[1].Email != "bob.ex@wso2.com" || users[1].UUID != "uuid-bob" {
		t.Errorf("row 1: %+v", users[1])
	}

	if _, err := loadSnapshotFile(filepath.Join(dir, "missing.csv")); err == nil {
		t.Error("a missing file should error")
	}
}
