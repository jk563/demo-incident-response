# Automated Incident Response Demo

A fully live demo showing end-to-end automated incident response: a production application experiences a real failure, AWS-native observability detects it, and an AI agent triages the incident and creates an issue with an engineering RCA.

**Audience**: Technical decision-makers at client/prospect organisations.
**Demo duration**: 3–5 minutes from "healthy" to "issue created with RCA".

## Architecture

```
Traffic Gen ──► Go Order API (ECS Fargate) ──► DynamoDB
                        │
                CloudWatch (Logs + Metrics + X-Ray)
                        │
                CloudWatch Alarm (error rate > 10%)
                        │
                    SNS Topic
                        │
                Lambda (Strands Agent)
                  │           │
            Queries CW/X-Ray   Reads source
                        │
                Creates Issue with RCA
```

See [docs/architecture.md](docs/architecture.md) for full architecture documentation with diagrams and design decisions.

## Prerequisites

- AWS CLI configured (eu-west-2)
- [just](https://github.com/casey/just) command runner
- Go 1.22+
- Python 3.12+
- Docker / docker
- Terraform 1.5+
- Git provider PAT (GitHub or GitLab) with appropriate scopes (store via `just update-pat <token>`)

## Quickstart

```bash
# 1. Deploy infrastructure
just tf-init
just tf-apply

# 2. Build and push images
just build-all

# 3. Seed sample data
just seed

# 4. Run pre-flight checks
just preflight

# 5. Start steady traffic
just steady

# 6. (When ready) Inject the bug
just inject

# 7. Watch the agent work
just tail-agent
```

## Repository Structure

```
├── demo-order-api/          # Go order processing service
│   ├── cmd/api/             # Entry point
│   ├── internal/            # Handlers, store, middleware, discount logic
│   ├── web/                 # Embedded frontend UI
│   └── Dockerfile
├── terraform/               # All AWS infrastructure
├── agent/                   # Strands triage agent (Python Lambda)
│   ├── tools/               # CloudWatch, X-Ray, Git provider tool functions
│   └── prompts/             # Agent system prompt
├── traffic/                 # Go traffic generator (two binaries)
│   └── cmd/                 # steady + inject
├── scripts/                 # Build, seed, preflight, reset helpers
├── slides/                  # Marp presentation deck
├── docs/                    # Architecture, backlog, runbook
├── bin/                     # Compiled Go binaries (gitignored)
├── test/                    # E2E tests (all paths except the WELCOME bug)
└── justfile                 # Primary interface for all operations
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/architecture.md](docs/architecture.md) | System architecture, diagrams, design decisions |
| [docs/backlog.md](docs/backlog.md) | Feature/task backlog (8 epics) |
| [docs/runbook.md](docs/runbook.md) | Demo day runbook and fallback procedures |
| [agent/prompts/system.md](agent/prompts/system.md) | Agent system prompt and RCA template |
| [slides/deck.md](slides/deck.md) | Presentation slide deck |

## The Bug

The Go order API has a deliberate bug: the `WELCOME` discount code maps to tier index 3, but only 3 tiers exist (indices 0–2). This causes an index-out-of-range panic on any order using the WELCOME code. All other discount codes work fine.

This is a realistic coordination bug — someone added a promotional code but forgot to add the corresponding discount tier.

## AWS Resources

| Resource | Detail |
|----------|--------|
| Region | eu-west-2 (London) |
| Domain | Configured via `.env` (`APP_DOMAIN`) |
| Terraform state | Configured via `.env` (`TF_STATE_BUCKET`) |
| ECS cluster | Fargate |
| DynamoDB | On-demand billing |
| Lambda | Container image, 5-min timeout |
| Bedrock model | Configurable (default: Claude Sonnet 4.6) |

## Cost Management

All infrastructure is designed for teardown after use. Run `just tf-destroy` when finished to avoid ongoing costs. No reserved capacity or provisioned resources.
