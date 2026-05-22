#!/usr/bin/env node

/**
 * E2E smoke test for the Distr Hub API.
 *
 * Exercises the full user journey: register → login → tutorial flow → verify side effects.
 *
 * Usage:
 *   DISTR_HOST=http://localhost:8080 node hack/e2e-smoke-test.mjs
 *
 * Requires Node.js 18+ (native fetch).
 */

const BASE_URL = (process.env.DISTR_HOST ?? 'http://localhost:8080').replace(/\/$/, '');

const TEST_EMAIL = 'e2e1@smoke.test';
const TEST_PASSWORD = 'E2eSmoke123!';
const TEST_NAME = 'E2E Smoke Test';

async function request(method, path, {body, token} = {}) {
  const headers = {'Content-Type': 'application/json'};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${method} ${path} → ${res.status}: ${text.trim()}`);
  }
  if (res.status === 204) {
    return null;
  }
  return res.json();
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(`Assertion failed: ${message}`);
  }
}

let stepNum = 0;
function step(name) {
  stepNum++;
  console.log(`[${stepNum}] ${name}`);
}

step('Register user');
await request('POST', '/api/v1/auth/register', {
  body: {name: TEST_NAME, email: TEST_EMAIL, password: TEST_PASSWORD},
});

step('Login');
const loginResponse = await request('POST', '/api/v1/auth/login', {
  body: {email: TEST_EMAIL, password: TEST_PASSWORD},
});
const token = loginResponse.token;
assert(token, 'login response must include a token');

step('Verify organization exists');
const org = await request('GET', '/api/v1/organization', {token});
assert(org && org.name, 'organization must have a name');

step('Trigger tutorial (agents/welcome/start)');
const tutorialResult = await request('PUT', '/api/v1/tutorial-progress/agents', {
  token,
  body: {stepId: 'welcome', taskId: 'start'},
});
const tutorialEvent = tutorialResult?.events?.find((e) => e.stepId === 'welcome' && e.taskId === 'start');
assert(tutorialEvent?.value?.deploymentTargetId, 'tutorial response must include an event with deploymentTargetId');

step('Verify hello-distr application was created');
const applications = await request('GET', '/api/v1/applications', {token});
assert(
  applications.some((a) => a.name === 'hello-distr'),
  'hello-distr application must exist'
);

step('Verify hello-distr-tutorial deployment target was created with a deployment');
const targets = await request('GET', '/api/v1/deployment-targets', {token});
const helloTarget = targets.find((t) => t.name === 'hello-distr-tutorial');
assert(helloTarget, 'hello-distr-tutorial deployment target must exist');
assert(helloTarget.deployments?.length > 0, 'hello-distr-tutorial must have at least one deployment');

console.log(`\nAll ${stepNum} smoke test steps passed.`);
