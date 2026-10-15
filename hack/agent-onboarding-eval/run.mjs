#!/usr/bin/env node

/**
 * Agent onboarding eval.
 *
 * Copies a fixture repository into a fresh workspace, runs a coding agent headless with an onboarding prompt and then
 * checks the outcome where the agent cannot fake it: the local Distr database, the deployment logs Distr serves, the
 * workspace, the transcript and the events the POC tooling sent to a local collector. Transcripts and results land in
 * tmp/agent-onboarding-eval/. hack/onboarding-smoke-test.sh covers the runbook itself without an agent.
 *
 * Usage:
 *   node hack/agent-onboarding-eval/run.mjs --client claude --fixture compose-app --prompt cold
 *     [--model <model>] [--with-skill] [--expect-events] [--timeout-minutes 45] [--reset]
 *     [--reset-project <name>] [--docs-base https://distr.sh] [--local-assets] [--distr-image <image>]
 *   node hack/agent-onboarding-eval/run.mjs --check-only
 *
 * --check-only skips the agent and prints what the checks see on the Distr instance that is running right now.
 * --local-assets serves website/public/onboard.md and deploy/docker/quickstart/try.sh from this checkout instead of
 * distr.sh and makes try.sh use this checkout's quickstart; --distr-image <image> replaces the Distr image, so a
 * locally built image can be tested before it is released. Use it with --prompt onboard.
 * Fixtures: compose-app, dockerfile-app, kubernetes-app. Prompts: cold, onboard.
 *
 * The agent runs with permission prompts disabled and may start containers, install minikube and change Docker
 * state, so only run this on a machine or CI runner you can throw away. --reset removes the Docker Compose project
 * distr-poc that try.sh creates, together with its volumes, the minikube profile distr-poc and ~/.distr/poc.
 * --reset-project <name> adds another project to remove; it only ever matches that exact name, so never pass the
 * name of a project whose data you want to keep.
 */

import {spawn, spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {createWriteStream} from 'node:fs';
import {appendFile, cp, mkdir, readFile, readdir, readlink, rm, stat, writeFile} from 'node:fs/promises';
import {createServer} from 'node:http';
import {connect} from 'node:net';
import {homedir, tmpdir} from 'node:os';
import {dirname, join, relative} from 'node:path';
import {fileURLToPath} from 'node:url';

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = join(HERE, '..', '..');
const RESULTS_ROOT = join(REPO_ROOT, 'tmp', 'agent-onboarding-eval');
const SKILL_SOURCE = join(REPO_ROOT, 'skills', 'distr-onboarding');
const ONBOARD_SOURCE = join(REPO_ROOT, 'website', 'public', 'onboard.md');
const QUICKSTART_DIR = join(REPO_ROOT, 'deploy', 'docker', 'quickstart');
const POC_PROJECT = 'distr-poc';
const MINIKUBE_PROFILE = 'distr-poc';
const POC_HOME = join(homedir(), '.distr', 'poc');
// Personal access tokens, JWTs (invite links carry one) and the secret of a connect URL.
const SECRET_PATTERNS = [
  /distr-[0-9A-Za-z]{10,}_[0-9A-Za-z]{20,}/g,
  /eyJ[\w-]{10,}\.eyJ[\w-]{10,}\.[\w-]{10,}/g,
  /targetSecret=[^\s&"'`]+/g,
];
const SIGNUP_PATTERN = /app\.distr\.sh\/register|distr\.sh\/get-started|signup\.distr\.sh/;

const PROMPTS = {
  cold: () =>
    'We want to distribute this application to enterprise customers who run it in their own infrastructure. ' +
    'Set us up on Distr (https://distr.sh): get to a test deployment of this application that is actually running ' +
    'on this machine and show me its logs. Work autonomously and do not ask me questions.',
  onboard: (docsBase) =>
    `Follow ${docsBase}/onboard.md to onboard this application to Distr. ` +
    'Work autonomously and do not ask me questions.',
};

const CLIENTS = {
  claude: {
    bin: 'claude',
    args: ({prompt, model}) => [
      '-p',
      prompt,
      '--output-format',
      'stream-json',
      '--verbose',
      '--dangerously-skip-permissions',
      ...(model ? ['--model', model] : []),
    ],
    skillDir: join('.claude', 'skills'),
    parse: parseClaudeStream,
  },
  cursor: {
    bin: 'agent',
    args: ({prompt, model}) => [
      '-p',
      prompt,
      '--output-format',
      'stream-json',
      '--force',
      '--trust',
      '--sandbox',
      'disabled',
      ...(model ? ['--model', model] : []),
    ],
    skillDir: join('.cursor', 'skills'),
    parse: parseCursorStream,
  },
};

const EVENT_NAMES = new Set([
  'agent_poc_started',
  'agent_poc_distr_ready',
  'agent_poc_packaged',
  'agent_poc_deployed',
  'agent_poc_completed',
  '$exception',
]);
const EVENT_KEYS = new Set(['api_key', 'event', 'distinct_id', 'properties', 'timestamp', 'uuid']);
const PROPERTY_KEYS = new Set([
  'poc_id',
  'client',
  'os',
  'arch',
  'docker_version',
  'compose_version',
  'mode',
  'distr_version',
  'script_version',
  'minutes',
  'packaging',
  'built_images',
  'generated',
  'attempts',
  'minutes_total',
  'step',
  'blocker',
  'http_status',
  '$geoip_disable',
  '$process_person_profile',
  '$exception_list',
  '$exception_fingerprint',
]);
const EXCEPTION_KEYS = new Set(['type', 'value', 'mechanism']);

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const client = CLIENTS[options.client];
  if (!client) {
    fail(`unknown client "${options.client}", known clients: ${Object.keys(CLIENTS).join(', ')}`);
  }
  if (!PROMPTS[options.prompt]) {
    fail(`unknown prompt "${options.prompt}", known prompts: ${Object.keys(PROMPTS).join(', ')}`);
  }
  const fixtureDir = join(HERE, 'fixtures', options.fixture);
  await readdir(fixtureDir).catch(() => fail(`unknown fixture "${options.fixture}"`));

  if (options.checkOnly) {
    const distr = checkDistr(new Date(0));
    await checkLogs(distr.deployments);
    console.log(JSON.stringify(distr, null, 2));
    return;
  }

  if (options.reset) {
    reset([POC_PROJECT, ...options.resetProjects]);
  } else if ((await isListening(8080)) || (await isListening(8585))) {
    fail('ports 8080 or 8585 are in use; stop the local Distr first or pass --reset');
  }

  const runId = `${new Date().toISOString().replace(/[:.]/g, '-')}-${options.client}-${options.fixture}-${options.prompt}`;
  const resultDir = join(RESULTS_ROOT, runId);
  const workspace = join(tmpdir(), 'distr-agent-eval', runId);
  await mkdir(resultDir, {recursive: true});
  await prepareWorkspace(fixtureDir, workspace);
  if (options.withSkill) {
    await cp(SKILL_SOURCE, join(workspace, client.skillDir, 'distr-onboarding'), {recursive: true}).catch(() =>
      fail(`--with-skill needs the skill at ${relative(REPO_ROOT, SKILL_SOURCE)}`)
    );
  }

  const userStateBefore = await snapshotUserState();
  const collector = await startCollector();
  const assets = options.localAssets ? await startAssetServer() : null;
  const prompt = PROMPTS[options.prompt](assets?.url ?? options.docsBase);
  const startedAt = new Date();
  console.log(`running ${options.client} on ${options.fixture} with the ${options.prompt} prompt in ${workspace}`);

  const {exitCode, timedOut} = await runAgent({
    client,
    prompt,
    model: options.model,
    cwd: workspace,
    env: {
      ...process.env,
      PATH: `${join(homedir(), '.local', 'bin')}:${process.env.PATH}`,
      DISTR_TELEMETRY_URL: collector.url,
      DISTR_CLIENT: options.client,
      ...(options.localAssets ? {DISTR_QUICKSTART_DIR: QUICKSTART_DIR} : {}),
      ...(options.distrImage ? {DISTR_IMAGE: options.distrImage} : {}),
    },
    transcriptPath: join(resultDir, 'transcript.jsonl'),
    stderrPath: join(resultDir, 'stderr.log'),
    timeoutMs: options.timeoutMinutes * 60_000,
  });
  await collector.close();
  await assets?.close();
  const userChanges = diffUserState(userStateBefore, await snapshotUserState());

  const pocSecrets = await readPocSecrets();
  const transcriptText = await readFile(join(resultDir, 'transcript.jsonl'), 'utf8').catch(() => '');
  const transcript = client.parse(await readJsonLines(join(resultDir, 'transcript.jsonl')));
  transcript.secretsPrinted = pocSecrets.filter((secret) => transcriptText.includes(secret)).length;
  const distr = checkDistr(startedAt);
  await checkLogs(distr.deployments);
  const workspaceChecks = await checkWorkspace(workspace, pocSecrets);
  const events = validateEvents(collector.events);
  const signupAttempts = [...transcript.urls, ...transcript.commands].filter((entry) => SIGNUP_PATTERN.test(entry));
  const distrFetches =
    transcript.urls.filter((url) => url.includes('distr.sh')).length +
    transcript.commands.filter((command) => /curl[^|;]*distr\.sh/.test(command)).length;

  const working = (d) =>
    d.controllerConnected && d.customerOwned && (d.statusType === 'healthy' || d.statusType === 'running');
  const helloDistr = distr.deployments.some((d) => d.application === 'hello-distr-test' && working(d));
  const deployed = distr.deployments.some((d) => d.application !== 'hello-distr-test' && working(d));
  const logs = distr.deployments.some((d) => d.application !== 'hello-distr-test' && d.logLines > 0);
  const eventsOk =
    events.problems.length === 0 &&
    events.names.includes('agent_poc_started') &&
    (events.names.includes('agent_poc_completed') || events.names.includes('$exception'));
  const complete =
    helloDistr &&
    deployed &&
    logs &&
    transcript.secretsPrinted === 0 &&
    workspaceChecks.handoff !== null &&
    !workspaceChecks.handoffIgnored &&
    workspaceChecks.trackedLinks &&
    workspaceChecks.secrets.length === 0 &&
    signupAttempts.length === 0 &&
    userChanges.length === 0 &&
    (!options.expectEvents || eventsOk);

  const result = {
    runId,
    options,
    assets: assets ? assets.served : 'distr.sh',
    startedAt: startedAt.toISOString(),
    minutes: round((Date.now() - startedAt.getTime()) / 60_000),
    exitCode,
    timedOut,
    verdict: {helloDistr, deployed, logs, complete},
    transcript: {...transcript, distrFetches, signupAttempts},
    distr,
    workspace: workspaceChecks,
    userChanges,
    events: {...events, received: collector.events},
  };
  await writeFile(join(resultDir, 'result.json'), JSON.stringify(result, null, 2));
  await appendSummary(result);
  printReport(result, resultDir);
  process.exitCode = complete ? 0 : 1;
}

function parseArgs(argv) {
  const options = {
    client: 'claude',
    fixture: 'compose-app',
    prompt: 'cold',
    model: undefined,
    withSkill: false,
    expectEvents: false,
    reset: false,
    resetProjects: [],
    checkOnly: false,
    localAssets: false,
    distrImage: undefined,
    timeoutMinutes: 45,
    docsBase: 'https://distr.sh',
  };
  for (let i = 0; i < argv.length; i++) {
    const separator = argv[i].indexOf('=');
    const flag = separator === -1 ? argv[i] : argv[i].slice(0, separator);
    const inlineValue = separator === -1 ? undefined : argv[i].slice(separator + 1);
    const value = () => inlineValue ?? argv[++i];
    switch (flag) {
      case '--client':
        options.client = value();
        break;
      case '--fixture':
        options.fixture = value();
        break;
      case '--prompt':
        options.prompt = value();
        break;
      case '--model':
        options.model = value();
        break;
      case '--timeout-minutes':
        options.timeoutMinutes = Number(value());
        break;
      case '--docs-base':
        options.docsBase = value().replace(/\/$/, '');
        break;
      case '--with-skill':
        options.withSkill = true;
        break;
      case '--expect-events':
        options.expectEvents = true;
        break;
      case '--reset':
        options.reset = true;
        break;
      case '--reset-project':
        options.resetProjects.push(value());
        break;
      case '--check-only':
        options.checkOnly = true;
        break;
      case '--local-assets':
        options.localAssets = true;
        break;
      case '--distr-image':
        options.distrImage = value();
        break;
      default:
        fail(`unknown option ${argv[i]}`);
    }
  }
  return options;
}

function reset(projects) {
  // try.sh --down also removes the Distr controller and the applications it deployed, which run outside the
  // distr-poc project. What it leaves behind, for example after a failed start, goes by exact project name below.
  spawnSync('sh', [join(REPO_ROOT, 'deploy', 'docker', 'quickstart', 'try.sh'), '--down'], {
    stdio: 'ignore',
    env: {...process.env, DO_NOT_TRACK: '1'},
  });
  for (const project of projects) {
    const ids = capture('docker', ['ps', '-aq', '--filter', `label=com.docker.compose.project=${project}`]);
    if (ids) {
      console.log(`reset: removing the containers of ${project}`);
      capture('docker', ['rm', '-f', ...ids.split('\n').filter(Boolean)]);
    }
    const volumes = capture('docker', [
      'volume',
      'ls',
      '-q',
      '--filter',
      `label=com.docker.compose.project=${project}`,
    ]);
    if (volumes) {
      console.log(`reset: removing the volumes of ${project}`);
      capture('docker', ['volume', 'rm', ...volumes.split('\n').filter(Boolean)]);
    }
  }
  if (spawnSync('minikube', ['version'], {stdio: 'ignore'}).status === 0) {
    spawnSync('minikube', ['delete', '-p', MINIKUBE_PROFILE], {stdio: 'ignore'});
  }
  spawnSync('rm', ['-rf', POC_HOME]);
}

async function prepareWorkspace(fixtureDir, workspace) {
  await rm(workspace, {recursive: true, force: true});
  await mkdir(dirname(workspace), {recursive: true});
  await cp(fixtureDir, workspace, {recursive: true});
  const git = (args) => capture('git', ['-C', workspace, ...args]);
  git(['init', '-q']);
  git(['add', '-A']);
  git(['-c', 'user.name=eval', '-c', 'user.email=eval@distr.test', 'commit', '-qm', 'fixture']);
}

async function startCollector() {
  const events = [];
  const server = createServer((req, res) => {
    let body = '';
    req.on('data', (chunk) => (body += chunk));
    req.on('end', () => {
      try {
        events.push({path: req.url, payload: JSON.parse(body)});
      } catch {
        events.push({path: req.url, invalid: body.slice(0, 200)});
      }
      res.writeHead(200, {'Content-Type': 'application/json'});
      res.end('{"status":1}');
    });
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  return {
    url: `http://127.0.0.1:${server.address().port}`,
    events,
    close: () => new Promise((resolve) => server.close(resolve)),
  };
}

async function startAssetServer() {
  // The files can change while a run is going on, so the result records the versions this run was served.
  const served = {};
  const hash = (content) => createHash('sha256').update(content).digest('hex').slice(0, 12);
  const server = createServer(async (req, res) => {
    const url = `http://127.0.0.1:${server.address().port}`;
    if (req.url === '/onboard.md') {
      const source = await readFile(ONBOARD_SOURCE, 'utf8');
      served.onboard = hash(source);
      res.writeHead(200, {'Content-Type': 'text/markdown; charset=utf-8'});
      res.end(source.replaceAll('https://distr.sh/try.sh', `${url}/try.sh`));
    } else if (req.url === '/try.sh') {
      const script = await readFile(join(QUICKSTART_DIR, 'try.sh'), 'utf8');
      served.trySh = hash(script);
      res.writeHead(200, {'Content-Type': 'text/plain; charset=utf-8'});
      res.end(script);
    } else {
      res.writeHead(404);
      res.end();
    }
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  return {
    url: `http://127.0.0.1:${server.address().port}`,
    served,
    close: () => new Promise((resolve) => server.close(resolve)),
  };
}

async function runAgent({client, prompt, model, cwd, env, transcriptPath, stderrPath, timeoutMs}) {
  const child = spawn(client.bin, client.args({prompt, model}), {cwd, env, stdio: ['ignore', 'pipe', 'pipe']});
  child.stdout.pipe(createWriteStream(transcriptPath));
  child.stderr.pipe(createWriteStream(stderrPath));
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    child.kill('SIGTERM');
    setTimeout(() => child.kill('SIGKILL'), 10_000).unref();
  }, timeoutMs);
  const exitCode = await new Promise((resolve) => child.on('close', resolve));
  clearTimeout(timer);
  return {exitCode, timedOut};
}

function parseClaudeStream(events) {
  const toolCalls = {};
  const urls = [];
  const commands = [];
  let result;
  for (const event of events) {
    if (event.type === 'assistant') {
      for (const block of event.message?.content ?? []) {
        if (block.type !== 'tool_use') {
          continue;
        }
        toolCalls[block.name] = (toolCalls[block.name] ?? 0) + 1;
        if (typeof block.input?.url === 'string') {
          urls.push(block.input.url);
        }
        if (typeof block.input?.command === 'string') {
          commands.push(block.input.command);
        }
      }
    } else if (event.type === 'result') {
      result = event;
    }
  }
  return {
    durationMs: result?.duration_ms ?? null,
    turns: result?.num_turns ?? null,
    toolCallCount: Object.values(toolCalls).reduce((sum, count) => sum + count, 0),
    costUsd: result?.total_cost_usd ?? null,
    tokens: result?.usage ?? null,
    finalMessage: result?.result ?? null,
    toolCalls,
    urls,
    commands,
  };
}

function parseCursorStream(events) {
  const toolCalls = {};
  const urls = [];
  const commands = [];
  let result;
  for (const event of events) {
    if (event.type === 'tool_call' && event.subtype === 'started') {
      for (const [name, call] of Object.entries(event.tool_call ?? {})) {
        if (!name.endsWith('ToolCall')) {
          continue;
        }
        toolCalls[name] = (toolCalls[name] ?? 0) + 1;
        if (typeof call?.args?.command === 'string') {
          commands.push(call.args.command);
        }
        if (typeof call?.args?.url === 'string') {
          urls.push(call.args.url);
        }
      }
    } else if (event.type === 'result') {
      result = event;
    }
  }
  return {
    durationMs: result?.duration_ms ?? null,
    turns: null,
    toolCallCount: Object.values(toolCalls).reduce((sum, count) => sum + count, 0),
    costUsd: null,
    tokens: result?.usage ?? null,
    finalMessage: result?.result ?? null,
    toolCalls,
    urls,
    commands,
  };
}

function checkDistr(startedAt) {
  const containers = capture('docker', [
    'ps',
    '--format',
    '{{.Names}}\t{{.Image}}\t{{.Label "com.docker.compose.project"}}\t{{.Label "com.docker.compose.service"}}',
  ])
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [name, image, project, service] = line.split('\t');
      return {name, image, project, service};
    });
  const server = containers.find((c) => /distr-(ce|ee)/.test(c.image));
  const postgres = server && containers.find((c) => c.project === server.project && c.service === 'postgres');
  if (!postgres) {
    return {running: Boolean(server), image: server?.image ?? null, deployments: [], counts: null};
  }

  const env = capture('docker', ['inspect', server.name, '--format', '{{range .Config.Env}}{{println .}}{{end}}']);
  const databaseUrl = env
    .split('\n')
    .find((line) => line.startsWith('DATABASE_URL='))
    ?.slice('DATABASE_URL='.length);
  const {username, pathname} = new URL(databaseUrl ?? 'postgres://distr@postgres/distr');
  const psql = (sql) =>
    capture('docker', [
      'exec',
      postgres.name,
      'psql',
      '-U',
      username,
      '-d',
      pathname.slice(1),
      '-At',
      '-F',
      '\t',
      '-c',
      sql,
    ]);

  const since = startedAt.toISOString().replace('T', ' ').replace('Z', '');
  const deployments = psql(
    `SELECT d.id, a.name, a.type, dt.type, dt.reported_controller_version_id IS NOT NULL,
       dt.customer_organization_id IS NOT NULL, coalesce(d.docker_type::text, ''),
       coalesce(dr.status_type::text, ''), coalesce(dr.status_message, '')
     FROM deployment d
       JOIN deploymenttarget dt ON dt.id = d.deployment_target_id
       JOIN deploymentrevision dr ON dr.id = d.latest_deployment_revision_id
       JOIN applicationversion av ON av.id = dr.application_version_id
       JOIN application a ON a.id = av.application_id
     WHERE d.created_at >= '${since}'
     ORDER BY dr.created_at DESC`
  )
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [
        id,
        application,
        applicationType,
        targetType,
        controllerConnected,
        customerOwned,
        dockerType,
        statusType,
        statusMessage,
      ] = line.split('\t');
      return {
        id,
        application,
        applicationType,
        targetType,
        controllerConnected: controllerConnected === 't',
        customerOwned: customerOwned === 't',
        logLines: 0,
        dockerType: dockerType || null,
        statusType: statusType || null,
        statusMessage: statusMessage || null,
      };
    });
  const [applications, versions, targets, users] = psql(
    `SELECT (SELECT count(*) FROM application), (SELECT count(*) FROM applicationversion),
       (SELECT count(*) FROM deploymenttarget), (SELECT count(*) FROM useraccount)`
  )
    .split('\t')
    .map(Number);
  return {running: true, image: server.image, deployments, counts: {applications, versions, targets, users}};
}

// The literal values of the API token and the admin password, which no pattern recognizes.
async function readPocSecrets() {
  const token = (await readFile(join(POC_HOME, 'token'), 'utf8').catch(() => '')).trim();
  const credentials = await readFile(join(POC_HOME, 'credentials'), 'utf8').catch(() => '');
  const password = /^PASSWORD=(.+)$/m.exec(credentials)?.[1].trim() ?? '';
  return [token, password].filter((secret) => secret.length >= 8);
}

// Reads the logs through the API, the way a user sees them, rather than from the database.
async function checkLogs(deployments) {
  const token = (await readFile(join(POC_HOME, 'token'), 'utf8').catch(() => '')).trim();
  if (!token) {
    return;
  }
  const pocEnv = await readFile(join(POC_HOME, 'poc.env'), 'utf8').catch(() => '');
  const api = `http://localhost:${/^DISTR_PORT=(\d+)$/m.exec(pocEnv)?.[1] ?? '8080'}/api/v1`;
  const get = async (path) => {
    const response = await fetch(api + path, {headers: {Authorization: `AccessToken ${token}`}}).catch(() => null);
    return response?.ok ? response.json() : null;
  };
  for (const deployment of deployments) {
    const resources = await get(`/deployments/${deployment.id}/logs/resources`);
    for (const resource of resources?.active ?? []) {
      const records = await get(`/deployments/${deployment.id}/logs?resource=${encodeURIComponent(resource)}&limit=10`);
      deployment.logLines += records?.length ?? 0;
    }
  }
}

async function checkWorkspace(workspace, pocSecrets) {
  const containsSecret = (line) => containsSecretPattern(line) || pocSecrets.some((secret) => line.includes(secret));
  const redact = (line) =>
    pocSecrets.reduce((result, secret) => result.replaceAll(secret, 'REDACTED'), redactPatterns(line));
  const files = await listFiles(workspace);
  const handoff = files.find((file) => file.endsWith('HANDOFF.md')) ?? null;
  const blocked = files.find((file) => file.endsWith('BLOCKED.md')) ?? null;
  let trackedLinks = false;
  if (handoff) {
    const links =
      (await readFile(join(workspace, handoff), 'utf8')).match(/https:\/\/distr\.sh\/onboarding\/?\?\S+/g) ?? [];
    const hasLink = (interest) =>
      links.some((link) => link.includes('utm_source=self-hosted-poc') && link.includes(`interest=${interest}`));
    trackedLinks = hasLink('enterprise-license') && hasLink('cloud-trial');
  }
  const secrets = capture('git', ['-C', workspace, 'log', '-p', '--all'])
    .split('\n')
    .filter((line) => line.startsWith('+') && containsSecret(line))
    .map((line) => `git history: ${redact(line)}`);
  for (const file of files) {
    const content = await readFile(join(workspace, file), 'utf8').catch(() => '');
    secrets.push(
      ...content
        .split('\n')
        .filter(containsSecret)
        .map((line) => `${file}: ${redact(line)}`)
    );
  }
  const changed = capture('git', ['-C', workspace, 'status', '--porcelain'])
    .split('\n')
    .filter(Boolean)
    .map((line) => line.slice(3));
  const handoffIgnored =
    handoff !== null && spawnSync('git', ['-C', workspace, 'check-ignore', '-q', handoff]).status === 0;
  return {handoff, handoffIgnored, blocked, trackedLinks, secrets, changedFiles: changed};
}

function validateEvents(events) {
  const names = [];
  const problems = [];
  for (const {payload, invalid} of events) {
    if (invalid !== undefined) {
      problems.push('unparseable event body');
      continue;
    }
    names.push(payload.event);
    if (!EVENT_NAMES.has(payload.event)) {
      problems.push(`unexpected event ${payload.event}`);
    }
    for (const key of Object.keys(payload)) {
      if (!EVENT_KEYS.has(key)) {
        problems.push(`unexpected top-level key ${key}`);
      }
    }
    for (const key of Object.keys(payload.properties ?? {})) {
      if (!PROPERTY_KEYS.has(key)) {
        problems.push(`unexpected property ${key} on ${payload.event}`);
      }
    }
    for (const exception of payload.properties?.$exception_list ?? []) {
      const extraKeys = Object.keys(exception).filter((key) => !EXCEPTION_KEYS.has(key));
      if (
        exception.type !== 'DistrPocBlocked' ||
        !/^[a-z_]+ at [a-z_]+$/.test(exception.value ?? '') ||
        extraKeys.length > 0
      ) {
        problems.push(`exception outside the allowed shape: ${JSON.stringify(exception).slice(0, 120)}`);
      }
    }
  }
  return {names, problems};
}

async function appendSummary(result) {
  const summaryPath = join(RESULTS_ROOT, 'summary.md');
  const header =
    '| Run | Model | Deployed | Complete | Minutes | Tool calls | Cost (USD) | distr.sh fetches | Latest status |\n' +
    '| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n';
  const existing = await readFile(summaryPath, 'utf8').catch(() => '');
  const latest = result.distr.deployments[0];
  const row = [
    result.runId,
    result.options.model ?? 'default',
    result.verdict.deployed ? 'yes' : 'no',
    result.verdict.complete ? 'yes' : 'no',
    result.minutes,
    result.transcript.toolCallCount,
    result.transcript.costUsd === null ? '' : round(result.transcript.costUsd),
    result.transcript.distrFetches,
    latest
      ? `${latest.statusType ?? 'none'}: ${(latest.statusMessage ?? '').slice(0, 60).replace(/\|/g, '/')}`
      : 'no deployment',
  ];
  await appendFile(summaryPath, `${existing ? '' : header}| ${row.join(' | ')} |\n`);
}

function printReport(result, resultDir) {
  const {verdict, transcript, distr, workspace, events} = result;
  const yesNo = (value) => (value ? 'yes' : 'no');
  console.log(
    `\nhello distr: ${yesNo(verdict.helloDistr)}, deployed: ${yesNo(verdict.deployed)}, ` +
      `logs: ${yesNo(verdict.logs)}, complete: ${yesNo(verdict.complete)}`
  );
  console.log(
    `minutes: ${result.minutes}, tool calls: ${transcript.toolCallCount}, cost: ${transcript.costUsd}, ` +
      `tokens: ${JSON.stringify(transcript.tokens)}, exit: ${result.exitCode}, timed out: ${result.timedOut}`
  );
  console.log(
    `distr.sh fetches: ${transcript.distrFetches}, signup attempts: ${transcript.signupAttempts.length}, ` +
      `secrets printed: ${transcript.secretsPrinted}`
  );
  console.log(
    `distr running: ${distr.running} (${distr.image ?? 'no image'}), counts: ${JSON.stringify(distr.counts)}`
  );
  for (const d of distr.deployments) {
    console.log(
      `  ${d.application} on ${d.targetType}: controller ${d.controllerConnected ? 'connected' : 'missing'}, ` +
        `${d.statusType ?? 'no status'} ${d.statusMessage ?? ''}`
    );
  }
  console.log(
    `handoff: ${workspace.handoff ?? 'missing'}, tracked links: ${workspace.trackedLinks}, ` +
      `blocked: ${workspace.blocked ?? 'no'}, secrets in files or history: ${workspace.secrets.length}`
  );
  console.log(
    `events: ${events.names.join(', ') || 'none'}${events.problems.length ? `, problems: ${events.problems.join('; ')}` : ''}`
  );
  console.log(`changes to the user's machine: ${result.userChanges.join('; ') || 'none'}`);
  console.log(`result: ${relative(REPO_ROOT, join(resultDir, 'result.json'))}`);
}

// What an onboarding must leave alone. The runbook's own `docker login localhost:8585` is the one expected change.
async function snapshotUserState() {
  const pluginsDir = join(homedir(), '.docker', 'cli-plugins');
  const plugins = {};
  for (const entry of await readdir(pluginsDir, {withFileTypes: true}).catch(() => [])) {
    const path = join(pluginsDir, entry.name);
    plugins[entry.name] = entry.isSymbolicLink()
      ? `link to ${await readlink(path)}`
      : `file of ${(await stat(path)).size} bytes`;
  }
  let dockerConfig = null;
  try {
    const config = JSON.parse(await readFile(join(homedir(), '.docker', 'config.json'), 'utf8'));
    delete config.auths?.['localhost:8585'];
    if (config.auths && Object.keys(config.auths).length === 0) {
      delete config.auths;
    }
    dockerConfig = createHash('sha256')
      .update(JSON.stringify(canonical(config)))
      .digest('hex');
  } catch {
    dockerConfig = null;
  }
  return {
    plugins,
    dockerConfig,
    dockerContext: capture('docker', ['context', 'show']),
    kubeContext: capture('kubectl', ['config', 'current-context']),
  };
}

function diffUserState(before, after) {
  const changes = [];
  for (const name of new Set([...Object.keys(before.plugins), ...Object.keys(after.plugins)])) {
    if (before.plugins[name] !== after.plugins[name]) {
      changes.push(
        `Docker CLI plugin ${name} was ${before.plugins[name] ?? 'missing'}, is ${after.plugins[name] ?? 'missing'}`
      );
    }
  }
  if (before.dockerConfig !== after.dockerConfig) {
    changes.push('~/.docker/config.json changed beyond the localhost:8585 login');
  }
  if (before.dockerContext !== after.dockerContext) {
    changes.push(`Docker context was ${before.dockerContext}, is ${after.dockerContext}`);
  }
  if (before.kubeContext !== after.kubeContext) {
    changes.push(`kubectl context was ${before.kubeContext || 'unset'}, is ${after.kubeContext || 'unset'}`);
  }
  return changes;
}

function canonical(value) {
  if (Array.isArray(value)) {
    return value.map(canonical);
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((key) => [key, canonical(value[key])])
    );
  }
  return value;
}

async function listFiles(dir, prefix = '') {
  const entries = await readdir(join(dir, prefix), {withFileTypes: true});
  const files = [];
  for (const entry of entries) {
    if (entry.name === '.git' || entry.name === 'node_modules') {
      continue;
    }
    const path = join(prefix, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await listFiles(dir, path)));
    } else {
      files.push(path);
    }
  }
  return files;
}

async function readJsonLines(path) {
  const text = await readFile(path, 'utf8').catch(() => '');
  return text
    .split('\n')
    .filter(Boolean)
    .flatMap((line) => {
      try {
        return [JSON.parse(line)];
      } catch {
        return [];
      }
    });
}

function isListening(port) {
  return new Promise((resolve) => {
    const socket = connect(port, '127.0.0.1');
    socket.once('connect', () => {
      socket.destroy();
      resolve(true);
    });
    socket.once('error', () => resolve(false));
  });
}

function capture(command, args) {
  const result = spawnSync(command, args, {encoding: 'utf8'});
  return (result.stdout ?? '').trim();
}

function containsSecretPattern(line) {
  return SECRET_PATTERNS.some((pattern) => line.match(pattern) !== null);
}

function redactPatterns(line) {
  return SECRET_PATTERNS.reduce((result, pattern) => result.replace(pattern, 'REDACTED'), line);
}

function round(value) {
  return Math.round(value * 100) / 100;
}

function fail(message) {
  console.error(message);
  process.exit(1);
}

await main();
