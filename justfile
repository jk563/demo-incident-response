set shell := ["bash", "-euo", "pipefail", "-c"]
set dotenv-load

# Initialise Terraform working directory
tf-init:
    cd terraform && terraform init \
        -backend-config="bucket=${TF_STATE_BUCKET}" \
        -backend-config="region=${AWS_REGION}" \
        ${AWS_PROFILE:+-backend-config="profile=${AWS_PROFILE}"}

# Generate terraform.tfvars from .env
gen-tfvars:
    @printf 'aws_region   = "%s"\nroute53_zone = "%s"\nsubdomain    = "%s"\ngit_provider = "%s"\ngit_repo     = "%s"\ngitlab_url   = "%s"\n' \
        "${AWS_REGION}" "${ROUTE53_ZONE}" "${SUBDOMAIN}" "${GIT_PROVIDER}" "${GIT_REPO}" "${GITLAB_URL:-https://gitlab.com}" \
        > terraform/terraform.tfvars

# Show Terraform execution plan
tf-plan: gen-tfvars
    cd terraform && terraform plan

# Apply Terraform changes to AWS (interactive approval)
tf-apply *FLAGS: gen-tfvars
    cd terraform && terraform apply {{FLAGS}}

# Destroy all Terraform-managed infrastructure (interactive approval)
tf-destroy *FLAGS:
    cd terraform && terraform destroy {{FLAGS}}

# Show Terraform outputs
tf-output:
    cd terraform && terraform output

# Build steady and inject traffic generator binaries
build-all:
    cd traffic && CGO_ENABLED=0 go build -o ../bin/steady ./cmd/steady
    cd traffic && CGO_ENABLED=0 go build -o ../bin/inject ./cmd/inject

# Seed DynamoDB with initial data
seed:
    ./scripts/seed-data.sh

# Run pre-flight checks before the demo
preflight:
    ./scripts/demo-preflight.sh

# Reset demo state to a clean baseline
reset:
    ./scripts/demo-reset.sh

# Run steady traffic generator
steady: build-all
    ./bin/steady -target "https://${APP_DOMAIN}"

# Inject error traffic to trigger the incident
inject: build-all
    ./bin/inject -target "https://${APP_DOMAIN}"

# Tail the triage agent Lambda logs
tail-agent:
    ./scripts/tail-agent-logs.sh

# Manually invoke the triage agent Lambda for testing
invoke-agent-manual:
    aws lambda invoke \
        --function-name demo-triage-agent \
        --cli-binary-format raw-in-base64-out \
        --payload '{"Records":[{"Sns":{"Message":"{\"AlarmName\":\"demo-incident-response-error-rate\",\"NewStateValue\":\"ALARM\",\"NewStateReason\":\"Manual invocation for testing\"}"}}]}' \
        --region "${AWS_REGION}" \
        ${AWS_PROFILE:+--profile "${AWS_PROFILE}"} \
        /dev/stdout

# Update the Git provider PAT stored in Secrets Manager
update-pat:
    ./scripts/update-git-pat.sh

# Run end-to-end tests against the live API
test-e2e:
    cd test && TEST_TARGET="https://${APP_DOMAIN}" go test -v -count=1 ./...

# Serve the observer UI locally
observer:
    @echo "Observer: http://localhost:9090?api=https://${APP_DOMAIN}&region=${AWS_REGION}&repo=${GIT_REPO}"
    python3 -m http.server 9090 --directory observer

# Full deploy: apply infrastructure, seed data, and run pre-flight checks
deploy: (tf-apply "-auto-approve") seed preflight
