# Running the HR entity locally

The Risk module's "Risk Identified By: Employee" field gets its employee list
from the WSO2 HR entity GraphQL service (`hr_entity`), which lives in the
**digiops-hr-main** repo — not in this repo, and not in this app's own
database. This file is local-only (gitignored) and just covers how to stand
that dependency up on your machine.

## Two things named "HR entity" — use the right one

`digiops-hr-main` has two different services. Only one of them can do what
this feature needs:

| Service | Location | Can list employees? |
|---|---|---|
| `entity-service` (REST) | `digiops-hr-main/entity-service` | **No** — single lookup by exact email only |
| `entity` (GraphQL, `hr_entity`) | `digiops-hr-main/entity` | **Yes** — `employees(filter, limit, offset)` |

The grc-platform backend talks to `entity`'s `employees` query. It's a
read-only integration — we never write back to HR.

## Local dev: use the mock, not the real service

Running the real `entity` service locally requires `bal run` against four
live MySQL databases (`people_ops_suite`, `hris`, `allocation_db`,
`leave_app`) via a `Config.toml` with real credentials — not something to
point at from a laptop for day-to-day dev.

Instead, use the mock server in this repo at `backend/cmd/hrmock`. It's a
small standalone Go program that serves the exact same wire contract
(`POST /graphql` accepting `{query, variables}`, returning employees shaped
like the real `Employee` type) with ~15 canned fake WSO2 employees. Because
the contract is identical, the grc-platform backend's code is unaware
whether it's talking to the mock or the real service — only a URL changes.

### Start it

```bash
cd apps/grc-platform/backend
go run ./cmd/hrmock
```

Listens on `:9090` by default (override with `PORT=xxxx go run ./cmd/hrmock`).

### Point the backend at it

Already set in `.env` / `.env.example`:

```
HR_ENTITY_GRAPHQL_URL=http://localhost:9090/graphql
```

Then run the backend as usual (`go run ./cmd/server`). Both processes need
to be running at the same time — the backend does not start the mock for
you. If the mock isn't up, `GET /api/v1/employees/search` will fail and the
Add/Edit Risk form will show "Unable to reach the employee directory" and
block the Employee field (by design — no free-text fallback).

### Quick manual test

```bash
curl -s -X POST http://localhost:9090/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query SearchEmployees($filter: EmployeeFilter, $limit: Int) { employees(filter: $filter, limit: $limit) { firstName lastName workEmail } }",
    "variables": {"filter": {"emailSearchString": "asel", "employeeStatus": ["Active"]}, "limit": 20}
  }'
```

Or through the real backend once both are running:

```bash
curl "http://localhost:8080/api/v1/employees/search?q=asel"
```

### Mock data

The fake employees in `cmd/hrmock/main.go` intentionally reuse the same
names as the "Risk Identified By" people already seeded in
`risk_module_data_schema.sql` (Asel Fernando, Nimal Jayasinghe, Dilini
Rathnayake, etc.), so the mock and the seeded dummy risks tell a consistent
story when you're clicking through the UI locally. Add more names to the
`mockEmployees` slice if you need to test a wider search.

## Production (Choreo)

grc-platform and the real `hr_entity` service will be deployed in the same
Choreo project. Set `HR_ENTITY_GRAPHQL_URL` to the real service's URL there
— no code changes needed, same as switching from the mock locally.
