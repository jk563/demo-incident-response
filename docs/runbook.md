# Demo Day Runbook — Automated Incident Response

## Overview

A 3–5 minute live demo showing end-to-end automated incident response. The target audience is technical decision-makers.

The demo presents a real Go order-processing API running on ECS Fargate. A buggy promotional discount code is injected, causing the service to fail. CloudWatch detects the elevated error rate and fires an alarm. A Strands AI triage agent is invoked automatically via SNS, gathers evidence from logs, traces, and source code, then creates a GitHub issue containing a full root-cause analysis (RCA).

The entire flow — from fault injection to actionable engineering issue — completes in under three minutes with zero human intervention.

## Timing Targets

| Phase | Target |
|---|---|
| Alarm fires | Within 90 seconds of inject start |
| Agent completes | Within 60 seconds of alarm |
| Total demo flow | 3–5 minutes |

## Pre-flight Checklist (30 minutes before)

1. Run `just deploy` — ensure all infrastructure is up and healthy.
2. Run `just preflight` — validate all smoke tests pass.
3. Run `just seed` — populate DynamoDB with sample orders.
4. Open browser tabs: App UI (`orders.${SUBDOMAIN}`), CloudWatch Dashboard, Issues page.
5. Open terminals: traffic generator, agent log tail (`just tail-agent`), command terminal.
6. Test `just steady` briefly — confirm the dashboard shows green metrics.
7. Stop steady traffic and clear any test alarms.
8. Verify the Git PAT is valid and the repo is accessible (`just update-pat` to refresh — pass the token as an argument or it will prompt interactively).
9. Check the Lambda is warm (invoke once manually if needed).

## Demo Script

### 1. Show the app UI

> "Here's our order processing service at orders.${SUBDOMAIN}. Real application, real orders, running on ECS Fargate."

### 2. Show the CloudWatch dashboard

> "Full AWS-native observability — metrics, logs, alarms, X-Ray tracing. All built in."

### 3. Start steady traffic

```bash
just steady
```

> "Generating normal production traffic — orders with standard discount codes, reads, refunds."

### 4. Narrate the dashboard

> "All green. Roughly 10 requests per second, zero errors, healthy latency."

### 5. The trigger

Pause for effect.

> "Marketing have just launched a new promotional code — WELCOME — for new customers..."

### 6. Inject buggy traffic

```bash
just inject
```

The audience sees you type this command live.

### 7. Watch the dashboard

Error rate climbs on `POST /orders`.

> "There it is. Errors climbing on the create-order endpoint."

### 8. Alarm fires

> "CloudWatch alarm triggered — error rate has crossed our 10% threshold."

### 9. Switch to agent logs

```bash
just tail-agent
```

> "The alarm fired to SNS, which invoked our triage agent. Let's watch it think..."

### 10. Narrate agent reasoning

The agent queries logs, pulls traces, and reads source code.

> "It's gathering evidence — structured logs, X-Ray traces, even reading the source file from the repository."

### 11. Switch to GitHub

> "And there's our issue, complete with a full engineering RCA."

### 12. Walk through the issue

Highlight the root cause, evidence (trace IDs, stack traces), and the suggested fix.

## Fallback Procedures

| Failure | Recovery |
|---|---|
| Alarm doesn't fire in time | `just invoke-agent-manual` — manually trigger the agent with a test payload |
| Agent Lambda errors | Show pre-recorded agent output screenshot, explain what would have happened |
| Dashboard slow to refresh | Narrate expected behaviour — "metrics will catch up in a moment" |
| App not responding | Check ECS service — `aws ecs describe-services`. If down, skip to pre-recorded demo |
| Git provider API rate limited | Agent will retry; if persistent, show pre-recorded issue |

## Post-Demo

- Stop traffic generators (`Ctrl+C`).
- Answer questions with the dashboard still visible.
- Optional: show the agent reasoning chain in detail.
- Optional: show the Terraform code — "everything as code, fully reproducible."
- Run `just tf-destroy` when finished to avoid ongoing costs.

## Key Talking Points

- Everything is AWS-native (no third-party observability tools).
- The agent reasons transparently — you can see every step.
- The RCA format matches what an experienced engineer would write.
- Total time from alert to actionable issue: under three minutes.
- Fully reproducible: `just deploy` from scratch.
