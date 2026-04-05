# Demo Incident Response

Automated incident response demo: a Go order API on ECS Fargate experiences a real failure, CloudWatch detects it, and a Strands AI agent triages and creates a GitHub issue with an engineering RCA.

## Repo Structure

- `demo-order-api/` — The Go order processing service (the app that breaks)
- `terraform/` — All AWS infrastructure (ECS, DynamoDB, CloudWatch, Lambda, etc.)
- `agent/` — Strands triage agent (Python Lambda)
- `traffic/` — Go traffic generator (two binaries: steady + inject)
- `scripts/` — Build, seed, preflight, reset helper scripts (includes `update-git-pat.sh`)
- `slides/` — Marp presentation deck
- `docs/` — Architecture, backlog, and runbook
- `bin/` — Compiled Go binaries (gitignored)
- `test/` — End-to-end tests (all paths except the WELCOME bug)

## Documentation

- `docs/architecture.md` — System architecture with Mermaid diagrams, design decisions, observability stack
- `docs/backlog.md` — Complete feature/task backlog (8 epics, implementation order)
- `docs/runbook.md` — Demo day runbook (pre-flight, demo script, fallbacks, post-demo)
- `agent/prompts/system.md` — Strands agent system prompt with RCA format template
- `slides/deck.md` — Marp presentation deck (10 slides with speaker notes)

## AWS

All environment-specific values (region, profile, domain, state bucket) are in `.env` (gitignored). See `.env.example` for the required keys. Terraform variables are auto-generated from `.env` by `just gen-tfvars` (called automatically before `tf-plan` and `tf-apply`).

## Conventions

- British English throughout (all code comments, docs, UI text, agent output)
- `just` is the primary interface — see `justfile` for all targets
- Terraform backend: S3 with key `demo-incident-response/terraform.tfstate`
- Terraform builds and pushes the app container image automatically on Go source changes
- Git provider PAT stored in Secrets Manager; update with `just update-pat <token>` (also accepts stdin or interactive prompt)
- Python agent code: Python 3.12, strands-agents SDK, boto3
- Go code: standard library where possible, `chi` router for HTTP

## Demo Workflow

Every time the user asks to run the demo, gather configuration first, then deploy and run. This ensures config is never stale.

### Step 0: Gather configuration

Use AskUserQuestion to confirm each setting. If `.env` already exists, show the current value as the default. Always ask — never silently reuse old config.

1. **Ask for the Git repository URL** — e.g. `https://github.com/org/repo` or `https://gitlab.example.com/group/project`
2. **Auto-detect from the URL:**
   - `GIT_PROVIDER`: `github` if the host is `github.com`, otherwise `gitlab`
   - `GIT_REPO`: for GitHub extract `org/repo`; for GitLab extract the project path or ask for the numeric project ID
   - `GITLAB_URL`: set to `https://{host}` if not `github.com` or `gitlab.com` (for self-hosted GitLab)
3. **Ask for AWS details** (with these defaults):
   - AWS region (default: `eu-west-2`)
   - AWS CLI profile name (default: empty)
   - Route53 hosted zone (e.g. `example.com`)
   - Subdomain (e.g. `demo.example.com`, default: `demo.{zone}`)
   - Terraform state S3 bucket name
4. **Ask for the PAT** — explain the required scopes for the detected provider:
   - GitHub: fine-grained PAT with Issues (rw), Contents (rw)
   - GitLab: project access token with `api` scope
5. **Write `.env`** — all runtime env vars (see `.env.example`). `APP_DOMAIN` is always `orders.{subdomain}`. Terraform variables are auto-generated from `.env` at deploy time.
6. **Store the PAT** — run `scripts/update-git-pat.sh` with the provided token
7. **Initialise Terraform** (first time only) — run `just tf-init`

### Step 1–8: Run the demo

1. `just deploy` — applies Terraform (auto-generates tfvars from `.env`), seeds DynamoDB, and runs preflight checks.
2. Verify all 7 preflight checks pass, including the Git PAT. If the PAT is missing or expired, run `just update-pat`.
3. Open browser tabs: App UI (`https://${APP_DOMAIN}`), CloudWatch Dashboard, Issues page.
4. `just steady` — start baseline traffic. **Must be running before inject** so the dashboard shows a healthy baseline for contrast.
5. `just inject` — sends WELCOME discount code requests that trigger the index-out-of-range panic.
6. Watch the CloudWatch dashboard — error rate climbs, alarm fires within ~90 seconds.
7. `just tail-agent` — watch the triage agent gather evidence and write the RCA.
8. Check Issues for the new issue with root-cause analysis.

### Post-demo cleanup

After the demo completes, ask the user whether they want to:
- **Delete created issues** — list any issues created during the demo and offer to close/delete them.
- **Tear down infrastructure** — run `just tf-destroy` to remove all AWS resources and avoid ongoing costs.

### Behavioural notes

- `just deploy` already chains `tf-apply -auto-approve`, `seed`, and `preflight` — no need to run them separately.
- The CloudWatch alarm SNS topic fires on both ALARM and OK state transitions. The agent ignores OK transitions, so expect two Lambda invocations per incident cycle (one acts, one is a no-op).
- Agent typically completes in 50–55 seconds after alarm fires.
- After the demo, stop traffic with Ctrl+C and run `just tf-destroy` to avoid ongoing costs.

## The Intentional Bug

The app has a deliberate bug in `demo-order-api/internal/discount/discount.go`:
`WELCOME` discount code maps to tier index 3, but only 3 tiers exist (indices 0–2).
This causes an index-out-of-range panic. **Do not fix this bug** — it is the demo trigger.
