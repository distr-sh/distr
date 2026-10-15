# Onboard an application to Distr

Distr distributes applications to customers who run them in their own infrastructure. A vendor packages the
application as Docker Compose or a Helm chart, and Distr adds a registry, license keys, a customer portal and
Distr controllers that install updates and report status and logs back.

This file is for AI coding agents. It takes a user from nothing to Hello Distr and then their own application running
locally through Distr for a test customer, then hands over the next steps. Follow it in order and do the work yourself
where you have a shell. Explain each step to the user in a sentence as you go.

## Ground rules

- Assume the user has no Distr account. Everything below runs on this machine with open source Distr.
- Distr Cloud has no self-signup. Never try to create a Distr Cloud account and never open `app.distr.sh/register`
  or `distr.sh/get-started`. Access to Distr Cloud comes through the onboarding link in the last step.
- A Distr controller is a small service that runs next to the deployed application, installs it and reports status
  and logs to Distr.
- Ask before you start containers, install minikube, kubectl or helm, connect a Distr controller, push to a registry
  outside this machine or invite anyone. Say what will run: about four containers on ports 8080 and 8585 and 4 GB
  of memory, then Hello Distr on ports 80 and 443, plus a minikube cluster for Helm. In a non-interactive run nobody
  can answer, so go ahead.
- Tell the user once that the scripts send anonymous progress events without application, repository or personal
  data, and that `DO_NOT_TRACK=1` turns them off.
- Write the packaging and the handoff into `deploy/distr/` in the repository, so they can be reviewed and committed.
  If `git check-ignore -q deploy/distr` succeeds, the directory is ignored: pick another directory that is not, and
  use it wherever this file says `deploy/distr/`. Do not change application code unless the user asks.

## 1. Start Distr locally

Both ways below run the published Distr Community Edition image `ghcr.io/distr-sh/distr-ce`. Do not build Distr from
source to onboard; that is only for changing Distr itself (see Development at the end).

Inside a Distr checkout (the repository root contains `deploy/docker/quickstart/try.sh`), use the files of the
checkout: read this runbook from `website/public/onboard.md`, fetch nothing from distr.sh and run every command from the
repository root.

```sh
sh deploy/docker/quickstart/try.sh --client <your client, for example claude-code>
```

Anywhere else:

```sh
curl -fsSL https://distr.sh/try.sh -o /tmp/distr-try.sh
sh /tmp/distr-try.sh --client <your client, for example claude-code>
```

Add `--kubernetes` to also start a minikube cluster for Helm charts.

The script checks Docker, Docker Compose, ports 8080 and 8585, free disk space and the commands it needs before it
downloads anything, and stops with what to fix. Then it downloads the images with progress and retries, starts the
Compose project `distr-poc`, waits until Distr is ready, creates an admin user, an API token and the registry slug
`poc` and prints JSON with everything else. The first run downloads about 500 MB and can take several minutes, later
runs start within a minute. If your tool times out, run `sh ~/.distr/poc/try.sh --pull` until it finishes, which only
downloads, and then `sh ~/.distr/poc/try.sh`. Every run resumes where the last one stopped.

After the first run, always use the copy at `~/.distr/poc/try.sh`. If port 8080 is taken, do not stop what holds it:
rerun with `DISTR_PORT=<free port>` in front of `sh`, which later runs remember. Distr is then at the `distrUrl` from
the JSON instead of http://localhost:8080. The JSON lists the commands to manage it under `commands`:

| What                          | Command                                            |
| ----------------------------- | -------------------------------------------------- |
| Start again                   | `sh ~/.distr/poc/compose.sh up -d --pull never`    |
| Check readiness               | `curl -fsS http://localhost:8080/ready`            |
| Logs of Distr                 | `sh ~/.distr/poc/compose.sh logs --tail 200 distr` |
| Stop                          | `sh ~/.distr/poc/compose.sh stop`                  |
| Diagnostics, secrets redacted | `sh ~/.distr/poc/try.sh --diagnose`                |
| Remove everything             | `sh ~/.distr/poc/try.sh --down`                    |

`compose.sh` runs `docker compose` with the project name, environment file and compose files Distr was started with,
so stopping and removing always match the start. `--down` also removes the Distr controllers and the applications
they deployed, and keeps `~/.distr/poc` when something could not be removed.

Your shell may not keep variables from one command to the next, so every command below starts with
`. ~/.distr/poc/api.env`. It sets `DISTR_API`, loads `DISTR_TOKEN` from its file without printing it, puts the tools
`try.sh` installed on `PATH` and loads the IDs saved to `~/.distr/poc/ids.env`, which the commands below append to.
Every API call sends `Authorization: AccessToken $DISTR_TOKEN`.

## 2. Deploy Hello Distr to a test customer

This proves the whole loop on this machine before you touch the user's application: an application with a version, a
customer, a deployment target that belongs to the customer, a Distr controller and a healthy deployment. Run the
blocks in order. Each one is safe to run again and ends with an error naming what is missing if it fails. Run it in the
Docker proof of concept: with `--kubernetes`, Distr hands out host names only minikube resolves, so skip to step 3.

<!-- hello-distr:start -->

Get the Hello Distr files and write the environment of the test customer's deployment. The database password is a
secret in Distr, so the file only references it:

```sh
mkdir -p ~/.distr/poc/hello-distr
if [ -f internal/resources/embedded/apps/hello-distr/docker-compose.yaml ]; then
  cp internal/resources/embedded/apps/hello-distr/docker-compose.yaml \
    internal/resources/embedded/apps/hello-distr/template.env ~/.distr/poc/hello-distr/
else
  for file in docker-compose.yaml template.env; do
    curl -fsSL "https://raw.githubusercontent.com/distr-sh/distr/main/internal/resources/embedded/apps/hello-distr/$file" \
      -o ~/.distr/poc/hello-distr/$file
  done
fi
cat > ~/.distr/poc/hello-distr/deployment.env <<'EOF'
HELLO_DISTR_HOST=localhost
HELLO_DISTR_PROTOCOL=http
HELLO_DISTR_DB_NAME=hello-distr
HELLO_DISTR_DB_USER=hello-distr
HELLO_DISTR_DB_PASSWORD={{ .Secrets.HELLO_DISTR_DB_PASSWORD }}
EOF
```

Create the application `hello-distr-test`, or select it when it exists:

```sh
. ~/.distr/poc/api.env
HELLO_APP_ID=$(curl --fail-with-body -sS "$DISTR_API/applications" -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq -r '[.[] | select(.name == "hello-distr-test")][0].id // empty')
if [ -z "$HELLO_APP_ID" ]; then
  HELLO_APP_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' \
    -d '{"name":"hello-distr-test","type":"docker"}' | jq -r .id)
fi
: "${HELLO_APP_ID:?the application was not created, read the response above}"
echo "HELLO_APP_ID=$HELLO_APP_ID" >> ~/.distr/poc/ids.env
```

Publish version `0.4.9`, or select it when it exists. Every file part needs `;type=application/yaml`:

```sh
. ~/.distr/poc/api.env
HELLO_VERSION_ID=$(curl --fail-with-body -sS "$DISTR_API/applications/$HELLO_APP_ID" \
  -H "Authorization: AccessToken $DISTR_TOKEN" | jq -r '[(.versions // [])[] | select(.name == "0.4.9")][0].id // empty')
if [ -z "$HELLO_VERSION_ID" ]; then
  HELLO_VERSION_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications/$HELLO_APP_ID/versions" \
    -H "Authorization: AccessToken $DISTR_TOKEN" \
    -F 'applicationversion={"name":"0.4.9","linkTemplate":"http://localhost"};type=application/json' \
    -F "composefile=@$HOME/.distr/poc/hello-distr/docker-compose.yaml;type=application/yaml" \
    -F "templatefile=@$HOME/.distr/poc/hello-distr/template.env;type=application/yaml" | jq -r .id)
fi
: "${HELLO_VERSION_ID:?the version was not created, read the response above}"
echo "HELLO_VERSION_ID=$HELLO_VERSION_ID" >> ~/.distr/poc/ids.env
```

Create the customer `Hello Distr Test Customer`, or select it when it exists:

```sh
. ~/.distr/poc/api.env
CUSTOMER_ID=$(curl --fail-with-body -sS "$DISTR_API/customer-organizations" \
  -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq -r '[.[] | select(.name == "Hello Distr Test Customer")][0].id // empty')
if [ -z "$CUSTOMER_ID" ]; then
  CUSTOMER_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/customer-organizations" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' \
    -d '{"name":"Hello Distr Test Customer"}' | jq -r .id)
fi
: "${CUSTOMER_ID:?the customer was not created, read the response above}"
echo "CUSTOMER_ID=$CUSTOMER_ID" >> ~/.distr/poc/ids.env
```

Store a random database password as a secret of that customer. Deployments to a customer's target only see the
customer's secrets, and the value never appears in the output:

```sh
. ~/.distr/poc/api.env
if ! curl --fail-with-body -sS "$DISTR_API/secrets" -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq -e --arg customer "$CUSTOMER_ID" \
    'any(.[]; .key == "HELLO_DISTR_DB_PASSWORD" and .customerOrganizationId == $customer)' >/dev/null; then
  jq -n --arg customer "$CUSTOMER_ID" --arg value "$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')" \
    '{key: "HELLO_DISTR_DB_PASSWORD", value: $value, customerOrganizationId: $customer}' |
    curl --fail-with-body -sS -o /dev/null -X POST "$DISTR_API/secrets" \
      -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @- &&
    echo "secret HELLO_DISTR_DB_PASSWORD created"
fi
```

Create the customer's Docker deployment target, or select it when it exists. `customerOrganizationId` makes the
customer own it, and the Community Edition allows one target per customer:

```sh
. ~/.distr/poc/api.env
TARGET_ID=$(curl --fail-with-body -sS "$DISTR_API/deployment-targets" -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq -r --arg customer "$CUSTOMER_ID" \
    '[.[] | select(.customerOrganizationId == $customer and .type == "docker")][0].id // empty')
if [ -z "$TARGET_ID" ]; then
  TARGET_ID=$(jq -n --arg customer "$CUSTOMER_ID" \
    '{name: "hello-distr-test", type: "docker", customerOrganizationId: $customer,
      deploymentLogsEnabled: true, metricsEnabled: false}' |
    curl --fail-with-body -sS -X POST "$DISTR_API/deployment-targets" \
      -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @- |
    jq -r .id)
fi
: "${TARGET_ID:?the deployment target was not created, read the response above}"
echo "TARGET_ID=$TARGET_ID" >> ~/.distr/poc/ids.env
```

Deploy the version to the target. `PUT /deployments` answers 204 without a body, so the deployment ID is read from
the target afterwards:

```sh
. ~/.distr/poc/api.env
deployment_of_app() {
  curl --fail-with-body -sS "$DISTR_API/deployment-targets/$TARGET_ID" -H "Authorization: AccessToken $DISTR_TOKEN" |
    jq -r --arg app "$HELLO_APP_ID" '[(.deployments // [])[] | select(.applicationId == $app)][0].id // empty'
}
jq -n --arg target "$TARGET_ID" --arg version "$HELLO_VERSION_ID" --arg deployment "$(deployment_of_app)" \
  --arg env "$(base64 < ~/.distr/poc/hello-distr/deployment.env | tr -d '\n')" \
  '{deploymentTargetId: $target, applicationVersionId: $version, dockerType: "compose", envFileData: $env}
    + if $deployment == "" then {} else {deploymentId: $deployment} end' |
  curl --fail-with-body -sS -X PUT "$DISTR_API/deployments" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @-
HELLO_DEPLOYMENT_ID=$(deployment_of_app)
: "${HELLO_DEPLOYMENT_ID:?the deployment was not created, read the response above}"
echo "HELLO_DEPLOYMENT_ID=$HELLO_DEPLOYMENT_ID" >> ~/.distr/poc/ids.env
```

Check that ports 80 and 443 are free for Hello Distr, then start the target's Distr controller on this machine. The
connect URL contains the target secret, so `--connect` fetches and applies it without printing it and waits until the
controller reports to Distr:

```sh
. ~/.distr/poc/api.env
sh ~/.distr/poc/try.sh --check-ports 80 443 && sh ~/.distr/poc/try.sh --connect --target "$TARGET_ID"
```

Wait until the deployment is healthy. The controller pulls the Hello Distr images first, which takes a few minutes on
the first run:

```sh
. ~/.distr/poc/api.env
attempt=0
while [ "$attempt" -lt 60 ]; do
  STATUS=$(curl --fail-with-body -sS "$DISTR_API/deployment-targets/$TARGET_ID" \
    -H "Authorization: AccessToken $DISTR_TOKEN" |
    jq -r --arg deployment "$HELLO_DEPLOYMENT_ID" \
      '[(.deployments // [])[] | select(.id == $deployment) | .latestStatus.type][0] // "pending"')
  echo "Hello Distr: $STATUS"
  [ "$STATUS" = healthy ] && break
  attempt=$((attempt + 1))
  sleep 10
done
[ "$STATUS" = healthy ]
```

Verify through the API that the customer owns the target, the controller reported and the deployment is healthy:

```sh
. ~/.distr/poc/api.env
TARGET=$(curl --fail-with-body -sS "$DISTR_API/deployment-targets/$TARGET_ID" \
  -H "Authorization: AccessToken $DISTR_TOKEN")
printf '%s' "$TARGET" | jq --arg deployment "$HELLO_DEPLOYMENT_ID" '{
  target: .name, customer: .customerOrganization.name, customerOrganizationId,
  controllerVersion: .reportedControllerVersionId,
  deployment: [(.deployments // [])[] | select(.id == $deployment) |
    {id, version: .applicationVersionName, status: .latestStatus.type}][0]}'
printf '%s' "$TARGET" | jq -e --arg customer "$CUSTOMER_ID" --arg deployment "$HELLO_DEPLOYMENT_ID" '
  .customerOrganizationId == $customer and .reportedControllerVersionId != null and
  any((.deployments // [])[]; .id == $deployment and .latestStatus.type == "healthy")' >/dev/null &&
  echo "verified"
```

Verify Hello Distr itself. Its health endpoint answers `{"status":"healthy","database":"connected"}`:

```sh
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  curl -fsS -m 5 http://localhost/api/health >/dev/null 2>&1 && break
  sleep 5
done
curl -fsS -m 5 http://localhost/api/health | jq -e '.status == "healthy" and .database == "connected"'
```

Read the deployment logs that the controller sent to Distr, which is where to look first when anything above fails:

```sh
. ~/.distr/poc/api.env
for resource in $(curl --fail-with-body -sS "$DISTR_API/deployments/$HELLO_DEPLOYMENT_ID/logs/resources" \
  -H "Authorization: AccessToken $DISTR_TOKEN" | jq -r '.active[]?'); do
  echo "== $resource"
  curl --fail-with-body -sS -G "$DISTR_API/deployments/$HELLO_DEPLOYMENT_ID/logs" \
    -H "Authorization: AccessToken $DISTR_TOKEN" --data-urlencode "resource=$resource" --data-urlencode limit=20 |
    jq -r '.[] | "\(.timestamp) \(.body)"'
done
```

<!-- hello-distr:end -->

If the status stays `progressing` or turns `error`, also read `.latestStatus.message` from the target, the controller
with `sh ~/.distr/poc/try.sh --diagnose` and the containers with `docker ps`. Show the user Hello Distr at
http://localhost and the customer and its target in Distr at the `distrUrl`, then report the milestone:

```sh
~/.distr/poc/event.sh deployed app=hello-distr minutes=<minutes since try.sh>
```

When the user has no application of their own, or the repository is Distr itself, go to step 7.

## 3. Look at the application

Read the repository: compose files, Dockerfiles, Helm charts, Kubernetes manifests, the services it needs
(databases, caches, queues), environment variables, ports and volumes.

Choose how Distr ships it:

- Docker Compose when the whole application runs on one host. This is the default.
- Helm when the repository already ships a Helm chart or Kubernetes manifests, when the application needs
  Kubernetes features (custom resources, operators, Jobs or CronJobs, several nodes) or when the user says their
  customers run Kubernetes. Rerun `sh ~/.distr/poc/try.sh --kubernetes` to add the minikube cluster.

## 4. Package the application

Docker Compose:

- Write or adapt `deploy/distr/docker-compose.yaml`. Every service needs an `image:` because Distr cannot build.
  Build images from Dockerfiles and push them to the local registry first (see Registry below).
- Put settings a customer changes into `deploy/distr/env.template` as `KEY=value` lines and use them in the compose
  file as `${KEY}`. Passwords become Distr secrets of the customer, like `HELLO_DISTR_DB_PASSWORD` in step 2.
- Prove it runs:
  `docker compose -p distr-app-test -f deploy/distr/docker-compose.yaml --env-file deploy/distr/env.template up -d`,
  check the containers, then `docker compose -p distr-app-test -f deploy/distr/docker-compose.yaml down -v`.

Helm:

- Write or adapt `deploy/distr/chart/`, with an empty `imagePullSecrets: []` list in `values.yaml` and in the pod
  specs that read it. The Distr controller fills in its pull secret for the Distr registry there. To test the chart
  before that, create the secret yourself with
  `kubectl --context distr-poc -n app-test create secret docker-registry distr-pull --docker-server=host.minikube.internal:8585 --docker-username=- --docker-password="$DISTR_TOKEN"`
  after `. ~/.distr/poc/api.env` and pass `--set 'imagePullSecrets[0].name=distr-pull'`.
- `helm lint deploy/distr/chart`, then prove it:
  `helm --kube-context distr-poc install app-test deploy/distr/chart -n app-test --create-namespace`, check the pods
  with `kubectl --context distr-poc -n app-test get pods`, then
  `helm --kube-context distr-poc uninstall app-test -n app-test`.
- Push the chart:

  ```sh
  . ~/.distr/poc/api.env
  echo "$DISTR_TOKEN" | helm registry login localhost:8585 -u - --password-stdin --insecure
  helm package deploy/distr/chart -d /tmp
  helm push /tmp/<chart>-<version>.tgz oci://localhost:8585/poc/charts --plain-http
  ```

Registry:

```sh
. ~/.distr/poc/api.env
echo "$DISTR_TOKEN" | docker login localhost:8585 -u - --password-stdin
docker build -t localhost:8585/poc/<name>:<tag> .
docker push localhost:8585/poc/<name>:<tag>
```

In compose files reference images as `localhost:8585/poc/<name>:<tag>`. In Helm values use
`host.minikube.internal:8585/poc/<name>:<tag>`, the name the cluster resolves. Charts live under `poc/charts/` so
they never share a repository and tag with an image.

Then report the milestone:

```sh
~/.distr/poc/event.sh packaged packaging=compose built_images=true generated=true
```

Use `packaging=helm` for Helm, and `built_images` and `generated` to say whether you built images and whether you
wrote the packaging from scratch.

## 5. Create the application in Distr

Use the same requests as step 2 with the user's files, and save the IDs as `APP_ID` and `VERSION_ID`. Docker Compose:

```sh
. ~/.distr/poc/api.env
APP_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"<app>","type":"docker"}' | jq -r .id)
: "${APP_ID:?}" && echo "APP_ID=$APP_ID" >> ~/.distr/poc/ids.env

VERSION_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications/$APP_ID/versions" \
  -H "Authorization: AccessToken $DISTR_TOKEN" \
  -F 'applicationversion={"name":"1.0.0"};type=application/json' \
  -F 'composefile=@deploy/distr/docker-compose.yaml;type=application/yaml' \
  -F 'templatefile=@deploy/distr/env.template;type=application/yaml' | jq -r .id)
: "${VERSION_ID:?}" && echo "VERSION_ID=$VERSION_ID" >> ~/.distr/poc/ids.env
```

Helm:

```sh
. ~/.distr/poc/api.env
APP_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"<app>","type":"kubernetes"}' | jq -r .id)
: "${APP_ID:?}" && echo "APP_ID=$APP_ID" >> ~/.distr/poc/ids.env

VERSION_ID=$(curl --fail-with-body -sS -X POST "$DISTR_API/applications/$APP_ID/versions" \
  -H "Authorization: AccessToken $DISTR_TOKEN" \
  -F 'applicationversion={"name":"1.0.0","chartType":"oci","chartUrl":"oci://host.minikube.internal:8585/poc/charts/<chart>","chartVersion":"<version>"};type=application/json' \
  -F 'valuesfile=@deploy/distr/chart/values.yaml;type=application/yaml' | jq -r .id)
: "${VERSION_ID:?}" && echo "VERSION_ID=$VERSION_ID" >> ~/.distr/poc/ids.env
```

Store passwords as secrets of the test customer with the secret request from step 2, and reference them as
`{{ .Secrets.<KEY> }}` in the deployment's environment file or values.

## 6. Deploy to the test customer

Docker Compose: deploy to the test customer's target from step 2, which can run several deployments. Hello Distr
holds ports 80 and 443, so when the application needs one of them, first remove Hello Distr with
`curl --fail-with-body -sS -X DELETE "$DISTR_API/deployments/$HELLO_DEPLOYMENT_ID" -H "Authorization: AccessToken $DISTR_TOKEN"`.
Write the customer's environment to `deploy/distr/deployment.env` and send it like step 2 does:

```sh
. ~/.distr/poc/api.env
jq -n --arg target "$TARGET_ID" --arg version "$VERSION_ID" \
  --arg env "$(base64 < deploy/distr/deployment.env | tr -d '\n')" \
  '{deploymentTargetId: $target, applicationVersionId: $version, dockerType: "compose", envFileData: $env}' |
  curl --fail-with-body -sS -X PUT "$DISTR_API/deployments" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @-
DEPLOYMENT_ID=$(curl --fail-with-body -sS "$DISTR_API/deployment-targets/$TARGET_ID" \
  -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq -r --arg app "$APP_ID" '[(.deployments // [])[] | select(.applicationId == $app)][0].id // empty')
: "${DEPLOYMENT_ID:?}" && echo "DEPLOYMENT_ID=$DEPLOYMENT_ID" >> ~/.distr/poc/ids.env
```

Helm: a customer has one target, so create a second test customer for Kubernetes the way step 2 created the first,
with the name `<app> Test Customer`, and give it a Kubernetes target. Save its IDs as `CUSTOMER_ID` and `TARGET_ID`,
which later commands then use:

```sh
. ~/.distr/poc/api.env
TARGET_ID=$(jq -n --arg customer "$CUSTOMER_ID" \
  '{name: "<app>-test", type: "kubernetes", namespace: "<app>", scope: "namespace", customerOrganizationId: $customer,
    deploymentLogsEnabled: true, metricsEnabled: false}' |
  curl --fail-with-body -sS -X POST "$DISTR_API/deployment-targets" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @- | jq -r .id)
: "${TARGET_ID:?}" && echo "TARGET_ID=$TARGET_ID" >> ~/.distr/poc/ids.env
jq -n --arg target "$TARGET_ID" --arg version "$VERSION_ID" \
  --arg values "$(base64 < deploy/distr/chart/values.yaml | tr -d '\n')" \
  '{deploymentTargetId: $target, applicationVersionId: $version, releaseName: "<app>", valuesYaml: $values}' |
  curl --fail-with-body -sS -X PUT "$DISTR_API/deployments" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @-
sh ~/.distr/poc/try.sh --connect --target "$TARGET_ID"
```

`--connect` creates the namespace, applies the controller and waits until it reports. Read `DEPLOYMENT_ID` from the
target like the Docker Compose block does.

## 7. Verify and fix

Use the wait, verify and logs blocks of step 2 with `$DEPLOYMENT_ID` in place of `$HELLO_DEPLOYMENT_ID`. `healthy` is
best. A Docker service without a healthcheck reaches `running`, which also counts.

If the status is `error` or the containers keep restarting, read the logs, fix the packaging, upload it as a new
version (`1.0.1`, `1.0.2` and so on) and move the deployment to it with the same `PUT /deployments` request plus
`deploymentId: $DEPLOYMENT_ID`. Stop after five attempts and go to Blocked.

When it works, report it:

```sh
~/.distr/poc/event.sh deployed attempts=<versions you uploaded> minutes=<minutes since try.sh>
```

Optionally show the user the customer side: create a user for the test customer. The local quickstart sends no
email, so the response contains an `inviteUrl` the user can open to see the customer portal. Anyone with that URL
can set the customer user's password, so show it in the chat and never write it into a file.

```sh
. ~/.distr/poc/api.env
jq -n --arg customer "$CUSTOMER_ID" \
  '{email: "customer@example.com", name: "Test Customer", userRole: "admin", customerOrganizationId: $customer}' |
  curl --fail-with-body -sS -X POST "$DISTR_API/user-accounts" \
    -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' --data-binary @-
```

## 8. Hand over

Write `deploy/distr/HANDOFF.md` and print it. It covers:

- what runs where: Distr at the `distrUrl` (log in as `poc@example.com`, the password is in
  `~/.distr/poc/credentials`), the test customer, its deployment target and how to reach the application
- what you created in Distr and every file you wrote
- how to stop and remove everything: `sh ~/.distr/poc/compose.sh stop` stops Distr, and
  `sh ~/.distr/poc/try.sh --down` removes Distr, the Distr controllers and the deployed applications
- the next steps, with the full URLs from `~/.distr/poc/links.env` pasted in (not the variable names):
  - keep self-hosting on your own server: the Community Edition stays free, and an Enterprise license adds license
    keys, RBAC, alerts and support bundles. Get one through `ENTERPRISE_LINK`.
  - or move to Distr Cloud with a free 30 day trial through `CLOUD_LINK`.
  - once you have access to either, ask your coding agent to apply `deploy/distr/` to the new instance and to publish
    new versions from CI with the `distr-sh/distr-create-version-action` GitHub Action

Then send `~/.distr/poc/event.sh completed packaging=<compose or helm> minutes_total=<minutes>`.

## Done

You are done when the transcript shows each of these as a command and its output:

1. `try.sh` printed its JSON summary.
2. Hello Distr passed the verify block and its health endpoint of step 2.
3. The user's packaging ran locally with plain Docker Compose or Helm.
4. `GET /deployment-targets/$TARGET_ID` shows the test customer's `customerOrganizationId`,
   `reportedControllerVersionId` and the user's deployment with status `healthy` or `running`.
5. `GET /deployments/$DEPLOYMENT_ID/logs` returned log lines.
6. `deploy/distr/HANDOFF.md` was printed with both next step links and no placeholders.
7. `~/.distr/poc/event.sh completed` ran.

Without an application of the user's own, items 3 to 5 are covered by Hello Distr.

## Blocked

If something only a person can fix stops you, or five versions did not help, write `deploy/distr/BLOCKED.md` with what
failed, what you tried and what you need, and print it together with the output of
`sh ~/.distr/poc/try.sh --diagnose`, which redacts every secret. Report it with
`~/.distr/poc/event.sh blocked step=<preflight|start_distr|package|registry|deploy|connect> blocker=<reason>`, where
the reason is one of `docker_missing`, `docker_outdated`, `compose_outdated`, `low_resources`, `port_in_use`,
`image_pull_failed`, `distr_not_ready`, `minikube_failed`, `image_build_failed`, `registry_push_failed`,
`controller_conflict`, `controller_not_connecting`, `deployment_unhealthy` or `other`. Give the user the full
`HELP_LINK` URL from `~/.distr/poc/links.env`, where the Distr team can help. A clean Blocked is a good ending, not a
failure.

## Later: apply `deploy/distr/` to a production Distr

When the user comes back with access to Distr Cloud or their own Distr, repeat steps 5 and 6 against it:

- Ask for its URL (Distr Cloud is https://app.distr.sh). Have the user create a personal access token in its web
  interface and save it to a file such as `~/.distr/prod-token`. Never ask for a password or for the token in the
  chat, and do not source `~/.distr/poc/api.env`, which points at the proof of concept.
- Push images and charts to that Distr's registry (`registry.distr.sh/<slug>/...` on Distr Cloud) and replace
  `localhost:8585/poc` and `host.minikube.internal:8585/poc` in `deploy/distr/` with it.
- Create the real customers and their deployment targets and hand each connect command to whoever runs that
  environment instead of running it yourself.
- Publish later versions from CI with the `distr-sh/distr-create-version-action` GitHub Action.

## Development: run a locally built Distr

Only when you change Distr itself and need code that no release contains. In a Distr checkout, with Docker as the only
prerequisite:

```sh
sh hack/build-ce-image.sh                                         # builds distr-ce:local from the checkout
DISTR_IMAGE=distr-ce:local sh deploy/docker/quickstart/try.sh
```

The first build downloads the Node.js and Go images and the dependencies, which takes a few minutes, and later builds
reuse the Docker build cache. `try.sh` remembers `DISTR_IMAGE` until `sh ~/.distr/poc/try.sh --down`. `mise run test:onboarding`
runs step 1 and step 2 end to end and tears everything down.

## Traps

Every line is a real failure from a real run.

- An ID variable from an earlier command is empty in a new shell, and the API answers `invalid UUID length: 0`.
  Start every command with `. ~/.distr/poc/api.env` and append new IDs to `~/.distr/poc/ids.env`.
- Distr cannot build images. Every compose service needs `image:`, and images you built must be pushed first.
- Version uploads are multipart forms and every file part needs `;type=application/yaml`, or the API answers 415.
- `PUT /deployments` answers 204 without a body. Read the deployment ID from `GET /deployment-targets/$TARGET_ID`.
- Always send `"dockerType":"compose"` for Docker deployments. Older Distr versions never apply a deployment without
  it, and it cannot be changed later: delete the deployment and create it again.
- The Community Edition allows one deployment target per customer. Reuse it, or delete it with
  `DELETE /deployment-targets/$TARGET_ID` before creating another.
- A deployment to a customer's target only resolves the customer's secrets. Create them with
  `customerOrganizationId`, or the deployment fails on the missing key.
- Logs need `?resource=<name>`. The names come from `GET /deployments/$DEPLOYMENT_ID/logs/resources`.
- Registry tags are immutable. Charts go under `poc/charts/<chart>`, images under `poc/<name>`.
- `helm registry login` takes `--insecure`, `helm push` takes `--plain-http`. Always pass `--kube-context distr-poc`
  to helm and `--context distr-poc` to kubectl, because `try.sh` leaves the user's current context alone.
- The Docker controller is the Compose project `distr`. Never run `docker compose` with `--remove-orphans` for that
  project name, and never `down` it, or the controller and the application disappear. `try.sh --connect` refuses to
  start a second controller next to one it did not start.
- On macOS keep `"metricsEnabled":false` on deployment targets. Logs work, metrics do not.

## Never

- Create a Distr Cloud account.
- Print, commit or write into files the API token, passwords or the connect URL. Share diagnostics only from
  `try.sh --diagnose`.
- Write an `inviteUrl` into a file. Show it in the chat only.
- Replace, upgrade or reconfigure Docker, Docker Compose, the Docker or Kubernetes context or other system tools. If
  one is too old, stop at Blocked (for example `blocker=compose_outdated`) and tell the user what to upgrade.
- Invite real people, send email or push to a registry outside this machine without asking.
- Run a connect command on another machine without asking.
- Delete Docker containers, volumes or clusters you did not create in this run. `try.sh --down` only removes the
  proof of concept.
