# Automated Incident Response Demo — Backlog

All tasks for the automated incident response demo. Organised by epic, with a recommended implementation order at the bottom.

---

## Epic 1: Go Order API (`demo-order-api/`)

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 1.1 | Scaffold Go project | P0 | Done | `go mod init`, `cmd/api/main.go`, project structure |
| 1.2 | Define domain models (Order, Item, DiscountTier) | P0 | Done | |
| 1.3 | Implement DynamoDB store layer (create, get, list, update) | P0 | Done | aws-sdk-go-v2 |
| 1.4 | Implement discount tier logic with the intentional bug | P0 | Done | WELCOME maps to index 3, tiers has 3 entries |
| 1.5 | Implement HTTP handlers (create, get, list, refund, health) | P0 | Done | chi router |
| 1.6 | Structured logging middleware (slog, JSON) | P0 | Done | request_id, trace_id, method, path, status_code, duration_ms |
| 1.7 | Recovery middleware (catch panics → structured error log + 500) | P0 | Done | Must log full stack trace |
| 1.8 | CloudWatch custom metrics middleware | P0 | Done | RequestCount, ErrorCount, Latency with endpoint dimensions |
| 1.9 | X-Ray instrumentation | P1 | Done | HTTP handler + DynamoDB client |
| 1.10 | Embedded frontend UI (web/ directory) | P1 | Done | Alpine.js, go:embed, serves at GET / |
| 1.11 | Dockerfile (multi-stage build, ARM64) | P0 | Done | GOARCH=arm64 for Fargate. Fixed: golang:1.24-alpine base. |
| 1.12 | Unit tests | P1 | Done | Valid discount codes, handler happy paths, store marshalling. Deliberately omit WELCOME test. |
| 1.13 | Integration tests (Docker Compose + LocalStack) | P2 | Won't do | Covered by E2E tests against live API |
| 1.14 | CLAUDE.md for app repo | P0 | Done | |

---

## Epic 2: Terraform Infrastructure (`terraform/`)

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 2.1 | Provider config + S3 backend | P0 | Done | S3 backend with partial config via justfile |
| 2.2 | VPC, public/private subnets, NAT gateway | P0 | Deployed | 2 AZs, single NAT |
| 2.3 | ALB + Route53 record | P0 | Deployed | ACM cert for *.${SUBDOMAIN}, HTTP→HTTPS redirect |
| 2.4 | ECR repository for app image | P0 | Deployed | force_delete enabled |
| 2.5 | ECS cluster + Fargate task def + service | P0 | Deployed | ARM64, 256 CPU, 512 MiB — running and healthy |
| 2.6 | DynamoDB orders table | P0 | Deployed | On-demand, PK: id, GSI: status-index |
| 2.7 | CloudWatch log group + metric filters | P0 | Deployed | LogErrorCount + LogRequestCount from structured logs |
| 2.8 | CloudWatch alarms (error rate >10%, 1-min) | P0 | Deployed | Math expression alarm, action: SNS |
| 2.9 | CloudWatch dashboard | P1 | Deployed | Request rate, error rate %, error count, latency, error logs, alarm widgets |
| 2.10 | X-Ray sampling rule (100%) | P1 | Deployed | |
| 2.11 | SNS topic + alarm subscription | P0 | Deployed | |
| 2.12 | ECR repository for agent image | P0 | Deployed | force_delete enabled |
| 2.13 | Lambda for Strands agent (container image) | P0 | Deployed | 5-min timeout, 1024 MB, ARM64 |
| 2.14 | IAM roles (ECS task, Lambda, CW read, X-Ray read, DynamoDB, Secrets Manager, GitHub) | P0 | Deployed | Least privilege |
| 2.15 | Secrets Manager entries (Git PAT) | P0 | Deployed | Value set via scripts/update-git-pat.sh |
| 2.16 | Terraform outputs (app URL, dashboard URL, log group, etc.) | P0 | Deployed | 9 outputs |
| 2.17 | Auto-build app container on Go source changes | P0 | Deployed | build.tf with source hash trigger, docker, --provenance=false |

---

## Epic 3: Strands Triage Agent (`agent/`)

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 3.1 | Agent system prompt (prompts/system.md) | P0 | Done | RCA format template, reasoning instructions |
| 3.2 | describe_alarm tool | P0 | Done | boto3 cloudwatch.describe_alarms() |
| 3.3 | query_logs tool | P0 | Done | boto3 logs.start_query()/get_query_results() (Logs Insights) |
| 3.4 | get_metric_data tool | P0 | Done | boto3 cloudwatch.get_metric_data() |
| 3.5 | get_xray_traces tool | P1 | Done | boto3 xray.get_trace_summaries() + batch_get_traces() |
| 3.6 | get_source_file tool | P1 | Done | GitHub API, reads from main branch |
| 3.7 | create_issue tool | P0 | Done | GitHub API, structured RCA body |
| 3.8 | Lambda handler (parse SNS → extract alarm → invoke agent) | P0 | Done | Ignores OK state, only investigates ALARM |
| 3.9 | Dockerfile (Python 3.12 + strands-agents + boto3) | P0 | Done | ARM64 Lambda base image. Fixed: removed non-existent strands-agents-bedrock dep. |
| 3.10 | Unit tests (mocked boto3) | P1 | Not started | |
| 3.11 | Integration test (sample alarm payload) | P1 | Not started | |

---

## Epic 4: Traffic Generator (`traffic/`)

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 4.1 | steady binary | P0 | Done | Creates, reads, refunds with valid codes. -target, -rate flags. Tested live. |
| 4.2 | inject binary | P0 | Done | POST /orders with WELCOME. -target, -rate, -duration flags. Tested live (5 500s). |
| 4.3 | --target and --rate flags | P0 | Done | Shared internal/client package with HTTP client + random order generation |
| 4.4 | Structured terminal output | P1 | Done | Periodic stats every 5s, summary on exit |

---

## Epic 5: Scripts & Justfile

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 5.1 | build-and-push.sh | P0 | Superseded | App build now handled by Terraform (build.tf). Agent build TBD. |
| 5.2 | seed-data.sh | P0 | Done | 20–30 sample orders with SAVE5/10/15 codes via `aws dynamodb put-item` |
| 5.3 | demo-preflight.sh | P0 | Done | All smoke checks, colour-coded, exit non-zero on failure |
| 5.4 | tail-agent-logs.sh | P0 | Done | `aws logs tail` with formatting |
| 5.5 | demo-reset.sh | P1 | Done | destroy → apply → seed → preflight |
| 5.6 | Justfile with all targets | P0 | Done | All targets wired up including test-e2e |
| 5.7 | update-git-pat.sh | P0 | Done | Stores PAT in Secrets Manager, includes required scope docs |

---

## Epic 6: Slides & Presentation (`slides/`)

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 6.1 | Marp config + theme | P1 | Done | |
| 6.2 | Slide content (10 slides) | P1 | Done | |
| 6.3 | Architecture diagrams for slides | P1 | Done | |
| 6.4 | Speaker notes per slide | P2 | Done | |

---

## Epic 7: Documentation & Project Setup

| # | Task | Priority | Status | Notes |
|---|------|----------|--------|-------|
| 8.1 | Initialise demo-incident-response git repo | P0 | Done | |
| 8.2 | CLAUDE.md for this repo | P0 | Done | |
| 8.3 | CLAUDE.md for app repo | P0 | Done | |
| 8.4 | README.md for both repos | P1 | Not started | |
| 8.5 | Architecture documentation | P1 | Done | docs/architecture.md with Mermaid diagrams |
| 8.6 | Demo day runbook | P1 | Done | docs/runbook.md |
| 8.7 | Slides deck | P1 | Done | slides/deck.md (Marp) |
| 8.8 | Agent system prompt | P0 | Done | agent/prompts/system.md |

---

## Implementation Order

Recommended session-by-session approach:

1. **Session 1** — Epic 8 (setup, docs) + Epic 1.1–1.5 (Go API core) ✅
2. **Session 2** — Epic 1.6–1.9 (observability) + Epic 1.11 (Dockerfile) + Epic 1.12 (tests) ✅
3. **Session 3** — Epic 2 (Terraform) + Epic 5.7 (PAT script) ✅ — validated, not yet applied
4. **Session 4** — Epic 1.10 (frontend UI) + Epic 3 (Strands agent) ✅
5. **Session 5** — `terraform apply` ✅ (57 resources deployed) + Epic 4 (traffic gen) ✅ + UI testing
6. **Session 6** — Epic 5 (scripts, Justfile) ✅ + Epic 6 (slides) ✅ + E2E tests ✅
7. **Next** — Timing calibration + rehearsal
8. **Later** — P1 remaining (3.10–3.11, 7.4)
