---
marp: true
theme: default
paginate: false
backgroundColor: #FFFFFF
style: |
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');
  section {
    font-family: 'Inter', 'Helvetica Neue', Arial, sans-serif;
    padding: 50px 70px;
    color: #1E293B;
  }
  section.lead {
    text-align: center;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
  section.lead h1 {
    font-size: 2.4em;
    border-bottom: none;
    display: block;
  }
  section.lead h2 {
    font-size: 1.3em;
    margin-top: 0.2em;
  }
  h1 {
    color: #0F172A;
    font-weight: 700;
    font-size: 1.7em;
    margin-bottom: 0.5em;
    border-bottom: 3px solid #3B82F6;
    padding-bottom: 0.15em;
    display: inline-block;
  }
  h2 {
    color: #475569;
    font-weight: 400;
    font-size: 1.15em;
  }
  strong { color: #1E40AF; }
  .subtitle {
    color: #64748B;
    font-size: 0.95em;
    margin-top: 1.5em;
    letter-spacing: 0.02em;
  }
  .kicker {
    margin-top: 1.5rem;
    padding: 14px 20px;
    background: #F1F5F9;
    border-left: 4px solid #3B82F6;
    font-size: 0.92em;
    color: #334155;
  }
  .two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2.5rem;
    margin-top: 1rem;
  }
  .two-col h3 {
    font-size: 0.95em;
    color: #3B82F6;
    font-weight: 700;
    margin-bottom: 0.4em;
  }
  .two-col ul {
    font-size: 0.88em;
    line-height: 1.7;
    padding-left: 1.2em;
  }
  img[alt~="diagram"] {
    display: block;
    margin: 0.8rem auto 0.5rem;
    max-width: 95%;
  }
  .caption {
    text-align: center;
    font-size: 0.85em;
    color: #64748B;
    margin-top: 0.3em;
  }
  .closing-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    margin-top: 1rem;
    font-size: 0.92em;
  }
  .closing-grid h3 {
    font-size: 0.95em;
    font-weight: 700;
    margin-bottom: 0.3em;
  }
  .closing-grid ul {
    font-size: 0.88em;
    line-height: 1.7;
    padding-left: 1.2em;
  }
---

<!-- _class: lead -->

# AI-Powered Incident Response

## From Alert to Root Cause — Autonomously

<div class="subtitle">
AWS-native · Serverless · Built with Strands Agents + Amazon Bedrock
</div>

<!--
Speaker notes:
This is a working demo, not slides-ware. Everything you'll see runs on real AWS infrastructure.
The goal: show that structured triage — the 30-minute tax on every incident — is automatable today.
-->

---

# What If Triage Took Seconds, Not Hours?

![diagram](diagrams/before-after.png)

Triage is investigation, not invention — the steps are structured and repeatable.
If your on-call runbook fits on a page, **an agent can follow it**.

<!--
Speaker notes:
Let the diagram do the talking. The contrast speaks for itself.
Top row: the reality everyone in this room knows — 30-45 minutes of senior engineer time, RCA often written up the next morning with half the context lost.
Bottom row: same investigation, done autonomously in under a minute, with the RCA filed immediately and all evidence attached.
The key phrase: "investigation, not invention." Triage follows a structured process. That's what makes it automatable.
-->

---

# Architecture

![diagram](diagrams/architecture.png)

**Fully AWS-native. Serverless. Event-driven.** Built entirely on services your clients likely already run — CloudWatch, Lambda, Bedrock. No third-party tooling required.

<!--
Speaker notes:
Walk through the flow left-to-right:
1. App fails → CloudWatch picks up the error spike via structured logs and metrics
2. Alarm breaches threshold (>10% error rate over 1 min) → SNS → Lambda
3. Agent uses Strands SDK to reason through the investigation: queries logs, metrics, X-Ray traces, reads the source code
4. Files a structured RCA issue with all evidence attached

Key sell for architects: everything here is IAC (Terraform), reproducible, and uses standard AWS integration patterns.
Key sell for skeptics: the agent uses the same data sources an engineer would. No magic — just automation of a structured process.
-->

---

# The Agent in Action

![diagram](diagrams/agent-flow.png)

The agent follows the same playbook a senior engineer would — querying logs, scoping blast radius, tracing requests, reading the source — and files a structured RCA with all evidence attached.

Every step is **logged and auditable**. No black box.

<!--
Speaker notes:
This is NOT a chatbot answering questions. It's an autonomous agent that:
1. Receives the alarm payload
2. Decides which tools to call and in what order
3. Synthesises the evidence into a root-cause analysis
4. Files it as an actionable engineering issue

The whole process takes a minute or two. We'll see it live in the demo.

For the technically curious: the agent uses Strands Agents SDK (AWS open-source), runs as a container on Lambda, and calls Claude via Amazon Bedrock. Each tool call is a real AWS API call — describe_alarm, query_logs, get_metric_data, etc.
-->

---

# Governance & Security

The question isn't *"can AI do this?"* — it's *"should we trust it to?"*

<div class="two-col">
<div>

### Built for trust
- Every agent action logged to CloudWatch
- Full audit trail — see exactly why it reached each conclusion
- Agent **files an issue, never deploys a fix**
- Human remains the decision-maker

</div>
<div>

### Built on AWS guardrails
- Amazon Bedrock — IAM-controlled, no API keys to manage
- Least-privilege IAM roles per component
- Private subnets; only the ALB is internet-facing

</div>
</div>

<div class="kicker">
AI as the investigator. Human as the decision-maker.
</div>

<!--
Speaker notes:
This slide is for the skeptics in the room — and there should be skeptics.
Key points:
- The agent is READ-ONLY on your infrastructure. It queries observability data. It does not modify anything.
- It files an issue. A human decides what to do about it.
- No data leaves the AWS account — Bedrock runs within your account boundary.
- Every tool invocation is logged, so you can audit exactly what the agent saw and why it reached its conclusion.

For clients in regulated industries: this is the right entry point for AI in operations — low risk, high visibility, auditable, and human-in-the-loop.
-->

---

<!-- _class: lead -->

# Taking It Further

<div class="closing-grid">
<div>

### For clients
- Bolt-on to any AWS environment with CloudWatch
- Customise to their stack — PagerDuty, Datadog, Jira, ServiceNow
- Natural extension of managed services engagements
- Low-risk AI entry point for regulated industries

</div>
<div>

### What's next
- Runbook automation beyond incident response
- Multi-cloud observability (Azure Monitor, GCP Cloud Ops)
- Integration with existing ITSM workflows

</div>
</div>

<div class="kicker" style="text-align: center; margin-top: 1.5rem;">
Live demo follows — let's break something.
</div>

<!--
Speaker notes:
Two angles here:
1. For the commercial-minded: this is a repeatable accelerator. Every client with AWS and an on-call team is a potential engagement. We're not selling AI — we're selling faster incident response with full auditability.

Transition to the live demo. Explain what they'll see: real traffic, a real bug, a real alarm, and the agent triaging it autonomously.
-->
