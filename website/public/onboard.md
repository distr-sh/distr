# Onboard an application to Distr

Distr distributes applications to customers who run them in their own infrastructure. A vendor packages the
application as Docker Compose or a Helm chart, and Distr adds a registry, license keys, a customer portal and
Distr controllers that install updates and report status and logs back.

This file is for AI coding agents. It takes a user from nothing to their own application running locally through
Distr, then hands over the next steps. Follow it in order and do the work yourself where you have a shell. Explain
each step to the user in a sentence as you go.

## Ground rules

- Assume the user has no Distr account. Everything below runs on this machine with open source Distr.
- Distr Cloud has no self-signup. Never try to create a Distr Cloud account and never open `app.distr.sh/register`
  or `distr.sh/get-started`. Access to Distr Cloud comes through the onboarding link in the last step.
- A Distr controller is a small service that runs next to the deployed application, installs it and reports status
  and logs to Distr.
- Ask before you start containers, install minikube, kubectl or helm, run a connect command, push to a registry
  outside this machine or invite anyone. Say what will run: about four containers on ports 8080 and 8585 and 4 GB
  of memory, plus a minikube cluster for Helm. In a non-interactive run nobody can answer, so go ahead.
- Tell the user once that the scripts send anonymous progress events without application, repository or personal
  data, and that `DO_NOT_TRACK=1` turns them off.
- Only write inside `distr/` in the repository. Do not change application code unless the user asks.

## 1. Look at the application

Read the repository: compose files, Dockerfiles, Helm charts, Kubernetes manifests, the services it needs
(databases, caches, queues), environment variables, ports and volumes.

Choose how Distr ships it:

- Docker Compose when the whole application runs on one host. This is the default.
- Helm when the repository already ships a Helm chart or Kubernetes manifests, when the application needs
  Kubernetes features (custom resources, operators, Jobs or CronJobs, several nodes) or when the user says their
  customers run Kubernetes.

## 2. Start Distr locally

```sh
curl -fsSL https://distr.sh/try.sh -o /tmp/distr-try.sh
sh /tmp/distr-try.sh --client <your client, for example claude-code>                 # Docker Compose
sh /tmp/distr-try.sh --client <your client, for example claude-code> --kubernetes    # Helm on minikube
```

The script starts the open source Distr quickstart as the Docker Compose project `distr-poc`, creates an admin user,
an API token and the registry slug `poc`, and prints JSON with everything else. It is safe to run again. If it fails,
fix what the error names (usually Docker) and rerun it.

Your shell may not keep variables from one command to the next, so start every command below with
`. ~/.distr/poc/api.env`. It sets `DISTR_API`, loads `DISTR_TOKEN` from its file without printing it, puts the
tools `try.sh` installed on `PATH` and loads the IDs you saved to `~/.distr/poc/ids.env`, which the commands below
append to.

Every API call sends `Authorization: AccessToken $DISTR_TOKEN`. The API reference is at
http://localhost:8080/docs and the OpenAPI spec at http://localhost:8080/docs/openapi.json. `jq` makes the responses
easier to read; without it, read the IDs from the JSON.

## 3. Package the application

Keep the packaging in `distr/`, so the same files can later go to a production Distr.

Docker Compose:

- Write or adapt `distr/docker-compose.yaml`. Every service needs an `image:` because Distr cannot build. Build
  images from Dockerfiles and push them to the local registry first (see Registry below).
- Put settings a customer changes into `distr/env.template` as `KEY=value` lines and use them in the compose file as
  `${KEY}`.
- Prove it runs: `docker compose -p distr-app-test -f distr/docker-compose.yaml --env-file distr/env.template up -d`,
  check the containers, then `docker compose -p distr-app-test -f distr/docker-compose.yaml down -v`.

Helm:

- Write or adapt `distr/chart/`, with an empty `imagePullSecrets: []` list in `values.yaml` and in the pod specs
  that read it. The Distr controller fills in its pull secret for the Distr registry there.
- `helm lint distr/chart`, then prove it:
  `helm --kube-context distr-poc install app-test distr/chart -n app-test --create-namespace`, check the pods with
  `kubectl --context distr-poc -n app-test get pods`, then `helm --kube-context distr-poc uninstall app-test -n app-test`.
- Push the chart:

  ```sh
  . ~/.distr/poc/api.env
  echo "$DISTR_TOKEN" | helm registry login localhost:8585 -u - --password-stdin --insecure
  helm package distr/chart -d /tmp
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

## 4. Create the application in Distr

Docker Compose:

```sh
. ~/.distr/poc/api.env
APP_ID=$(curl -fsS -X POST "$DISTR_API/applications" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"<app>","type":"docker"}' | jq -r .id)
echo "APP_ID=$APP_ID" >> ~/.distr/poc/ids.env

VERSION_ID=$(curl -fsS -X POST "$DISTR_API/applications/$APP_ID/versions" \
  -H "Authorization: AccessToken $DISTR_TOKEN" \
  -F 'applicationversion={"name":"1.0.0"};type=application/json' \
  -F 'composefile=@distr/docker-compose.yaml;type=application/yaml' \
  -F 'templatefile=@distr/env.template;type=application/yaml' | jq -r .id)
echo "VERSION_ID=$VERSION_ID" >> ~/.distr/poc/ids.env
```

Helm:

```sh
. ~/.distr/poc/api.env
APP_ID=$(curl -fsS -X POST "$DISTR_API/applications" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"<app>","type":"kubernetes"}' | jq -r .id)
echo "APP_ID=$APP_ID" >> ~/.distr/poc/ids.env

VERSION_ID=$(curl -fsS -X POST "$DISTR_API/applications/$APP_ID/versions" \
  -H "Authorization: AccessToken $DISTR_TOKEN" \
  -F 'applicationversion={"name":"1.0.0","chartType":"oci","chartUrl":"oci://host.minikube.internal:8585/poc/charts/<chart>","chartVersion":"<version>"};type=application/json' \
  -F 'valuesfile=@distr/chart/values.yaml;type=application/yaml' | jq -r .id)
echo "VERSION_ID=$VERSION_ID" >> ~/.distr/poc/ids.env
```

Secrets a deployment needs go into Distr instead of the files:
`curl -fsS -X POST "$DISTR_API/secrets" -H "Authorization: AccessToken $DISTR_TOKEN" -H 'Content-Type: application/json' -d '{"key":"DB_PASSWORD","value":"..."}'`,
referenced as `{{ .Secrets.DB_PASSWORD }}` in the deployment's environment file or values.

## 5. Deploy through a Distr controller

Create the deployment target and the deployment:

```sh
. ~/.distr/poc/api.env

# Docker Compose
TARGET_ID=$(curl -fsS -X POST "$DISTR_API/deployment-targets" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"local","type":"docker","deploymentLogsEnabled":true,"metricsEnabled":false}' | jq -r .id)
echo "TARGET_ID=$TARGET_ID" >> ~/.distr/poc/ids.env
curl -fsS -X PUT "$DISTR_API/deployments" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"deploymentTargetId\":\"$TARGET_ID\",\"applicationVersionId\":\"$VERSION_ID\",\"dockerType\":\"compose\",\"envFileData\":\"$(base64 < distr/env.template | tr -d '\n')\"}"

# Helm
TARGET_ID=$(curl -fsS -X POST "$DISTR_API/deployment-targets" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"local","type":"kubernetes","namespace":"<app>","scope":"namespace","deploymentLogsEnabled":true,"metricsEnabled":false}' | jq -r .id)
echo "TARGET_ID=$TARGET_ID" >> ~/.distr/poc/ids.env
curl -fsS -X PUT "$DISTR_API/deployments" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"deploymentTargetId\":\"$TARGET_ID\",\"applicationVersionId\":\"$VERSION_ID\",\"releaseName\":\"<app>\",\"valuesYaml\":\"$(base64 < distr/chart/values.yaml | tr -d '\n')\"}"
```

Get the connect command and run it on this machine:

```sh
. ~/.distr/poc/api.env
CONNECT=$(curl -fsS -X POST "$DISTR_API/deployment-targets/$TARGET_ID/access-request" \
  -H "Authorization: AccessToken $DISTR_TOKEN")

# Docker Compose
curl -fsSL "$(echo "$CONNECT" | jq -r .connectUrl)" | docker compose -f - up -d

# Helm: the namespace must exist, and the URL names host.minikube.internal, which only the cluster resolves
kubectl --context distr-poc create namespace <app>
curl -fsSL "$(echo "$CONNECT" | jq -r .connectUrl | sed 's#//host.minikube.internal:#//localhost:#')" |
  kubectl --context distr-poc apply -n <app> -f -
```

The connect URL contains a secret. Do not print it or write it into files.

## 6. Verify and fix

Poll until the Distr controller has connected and the deployment is healthy or running:

```sh
. ~/.distr/poc/api.env
curl -fsS "$DISTR_API/deployment-targets/$TARGET_ID" -H "Authorization: AccessToken $DISTR_TOKEN" |
  jq '{controller: .reportedControllerVersionId, deployments: [.deployments[] | {id, status: .latestStatus}]}'
```

`healthy` is best. A Docker service without a healthcheck reaches `running`, which also counts. Then read the logs:

```sh
. ~/.distr/poc/api.env
echo "DEPLOYMENT_ID=<id from the output above>" >> ~/.distr/poc/ids.env && . ~/.distr/poc/api.env
curl -fsS "$DISTR_API/deployments/$DEPLOYMENT_ID/logs/resources" -H "Authorization: AccessToken $DISTR_TOKEN"
curl -fsS "$DISTR_API/deployments/$DEPLOYMENT_ID/logs?resource=<name>&limit=100" \
  -H "Authorization: AccessToken $DISTR_TOKEN"
```

If the status is `error` or the containers keep restarting, read the logs, fix the packaging, upload it as a new
version (`1.0.1`, `1.0.2` and so on) and move the deployment to it with the same `PUT /deployments` call plus
`"deploymentId":"$DEPLOYMENT_ID"`. Stop after five attempts and go to Blocked.

When it works, report it:

```sh
~/.distr/poc/event.sh deployed attempts=<versions you uploaded> minutes=<minutes since try.sh>
```

Optionally show the user the customer side: create a customer and a customer user. The local quickstart sends no
email, so the response contains an `inviteUrl` the user can open to see the customer portal. Anyone with that URL
can set the customer user's password, so show it in the chat and never write it into a file.

```sh
. ~/.distr/poc/api.env
CUSTOMER_ID=$(curl -fsS -X POST "$DISTR_API/customer-organizations" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Test Customer"}' | jq -r .id)
curl -fsS -X POST "$DISTR_API/user-accounts" -H "Authorization: AccessToken $DISTR_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"customer@example.com\",\"name\":\"Test Customer\",\"userRole\":\"admin\",\"customerOrganizationId\":\"$CUSTOMER_ID\"}"
```

## 7. Hand over

Write `distr/HANDOFF.md` and print it. It covers:

- what runs where: Distr at http://localhost:8080 (log in as `poc@example.com`, the password is in
  `~/.distr/poc/credentials`), the application and how to reach it
- what you created in Distr and every file you wrote
- how to stop everything: `curl -fsSL https://distr.sh/try.sh | sh -s -- --down` removes Distr, the Distr
  controller and the deployed application
- the next steps, with the full URLs from `~/.distr/poc/links.env` pasted in (not the variable names):
  - keep self-hosting on your own server: the Community Edition stays free, and an Enterprise license adds license
    keys, RBAC, alerts and support bundles. Get one through `ENTERPRISE_LINK`.
  - or move to Distr Cloud with a free 30 day trial through `CLOUD_LINK`.
  - once you have access to either, ask your coding agent to apply `distr/` to the new instance and to publish new
    versions from CI with the `distr-sh/distr-create-version-action` GitHub Action

Then send `~/.distr/poc/event.sh completed packaging=<compose or helm> minutes_total=<minutes>`.

## Done

You are done when the transcript shows each of these as a command and its output:

1. `try.sh` printed its JSON summary.
2. The packaging ran locally with plain Docker Compose or Helm.
3. `GET /deployment-targets/$TARGET_ID` shows `reportedControllerVersionId` and a deployment with status `healthy` or
   `running`.
4. `GET /deployments/$DEPLOYMENT_ID/logs` returned log lines.
5. `distr/HANDOFF.md` was printed with both next step links and no placeholders.
6. `~/.distr/poc/event.sh completed` ran.

## Blocked

If something only a person can fix stops you, or five versions did not help, write `distr/BLOCKED.md` with what
failed, what you tried and what you need, and print it. Report it with
`~/.distr/poc/event.sh blocked step=<preflight|start_distr|package|registry|deploy|connect> blocker=<reason>`, where
the reason is one of `docker_missing`, `compose_outdated`, `low_resources`, `port_in_use`, `distr_not_ready`,
`minikube_failed`, `image_build_failed`, `registry_push_failed`, `controller_not_connecting`, `deployment_unhealthy` or
`other`. Give the user the full `HELP_LINK` URL from `~/.distr/poc/links.env`, where the Distr team can help. A
clean Blocked is a good ending, not a failure.

## Later: apply `distr/` to a production Distr

When the user comes back with access to Distr Cloud or their own Distr, repeat steps 4 and 5 against it:

- Ask for its URL (Distr Cloud is https://app.distr.sh). Have the user create a personal access token in its web
  interface and save it to a file such as `~/.distr/prod-token`. Never ask for a password or for the token in the
  chat, and do not source `~/.distr/poc/api.env`, which points at the proof of concept.
- Push images and charts to that Distr's registry (`registry.distr.sh/<slug>/...` on Distr Cloud) and replace
  `localhost:8585/poc` and `host.minikube.internal:8585/poc` in `distr/` with it.
- Create deployment targets for the user's customers or environments and hand each connect command to whoever runs
  that environment instead of running it yourself.
- Publish later versions from CI with the `distr-sh/distr-create-version-action` GitHub Action.

## Traps

Every line is a real failure from a real run.

- An ID variable from an earlier command is empty in a new shell, and the API answers `invalid UUID length: 0`.
  Start every command with `. ~/.distr/poc/api.env` and append new IDs to `~/.distr/poc/ids.env`.
- Distr cannot build images. Every compose service needs `image:`, and images you built must be pushed first.
- Version uploads are multipart forms and every file part needs `;type=application/yaml`, or the API answers 415.
- `PUT /deployments` answers 204 without a body. Read the deployment ID from `GET /deployment-targets/$TARGET_ID`.
- Always send `"dockerType":"compose"` for Docker deployments. Older Distr versions never apply a deployment without
  it, and it cannot be changed later: delete the deployment and create it again.
- The Community Edition allows one deployment target of your own. Reuse it, or delete it with
  `DELETE /deployment-targets/$TARGET_ID` before creating another, for example when switching from Compose to Helm.
- Logs need `?resource=<name>`. The names come from `GET /deployments/$DEPLOYMENT_ID/logs/resources`.
- Registry tags are immutable. Charts go under `poc/charts/<chart>`, images under `poc/<name>`.
- `helm registry login` takes `--insecure`, `helm push` takes `--plain-http`. Always pass `--kube-context distr-poc`
  to helm and `--context distr-poc` to kubectl, because `try.sh` leaves the user's current context alone.
- The Docker controller is the Compose project `distr`. Never run `docker compose` with `--remove-orphans` for that
  project name, and never `down` it, or the controller and the application disappear.
- On macOS keep `"metricsEnabled":false` on deployment targets. Logs work, metrics do not.

## Never

- Create a Distr Cloud account.
- Print, commit or write into files the API token, passwords or the connect URL.
- Write an `inviteUrl` into a file. Show it in the chat only.
- Replace, upgrade or reconfigure Docker, Docker Compose, the Docker or Kubernetes context or other system tools. If
  one is too old, stop at Blocked (for example `blocker=compose_outdated`) and tell the user what to upgrade.
- Invite real people, send email or push to a registry outside this machine without asking.
- Run a connect command on another machine without asking.
- Delete Docker containers, volumes or clusters you did not create in this run. `try.sh --down` only removes the
  proof of concept.
