#!/usr/bin/env node

/**
 * E2E test for the Distr Hub authentication flows.
 *
 * Exercises every credential-setup path against the API and verifies the side effects
 * (issued JWTs and the emails delivered to Mailpit):
 *
 *   - Registration            (verification required → email_verified=false token + verification mail)
 *   - Resend verification mail (/auth/verify/request delivers a new verification mail)
 *   - Invite via email link    (email_verified=true token → accept logs in directly, email already verified)
 *   - Invite via response URL  (email_verified=false token → accept sends a verification mail, then /verify)
 *   - Password reset           (reset token → /auth/reset/confirm sets password, verifies email, logs in)
 *   - Token scope enforcement  (reset/confirm and invite/accept reject unscoped and mismatched tokens)
 *   - Resend invitation        (a second invitation mail is delivered)
 *
 * The script adapts to whether USER_EMAIL_VERIFICATION_REQUIRED is enabled (detected at runtime).
 *
 * Usage:
 *   DISTR_HOST=http://localhost:8080 MAILPIT_HOST=http://localhost:8025 node hack/e2e-auth-test.mjs
 *
 * Requires Node.js 18+ (native fetch) and a running Hub + Mailpit.
 */

const BASE_URL = (process.env.DISTR_HOST ?? 'http://localhost:8080').replace(/\/$/, '');
const MAILPIT_URL = (process.env.MAILPIT_HOST ?? 'http://localhost:8025').replace(/\/$/, '');

const RUN_ID = `${Date.now()}-${Math.random().toString(16).slice(2)}`;

const PASSWORD = 'E2eAuth123!';
const NEW_PASSWORD = 'E2eAuth456!';

const VERIFY_SUBJECT = 'Verify your Distr account';
const INVITE_SUBJECT = 'Welcome to';
const RESET_SUBJECT = 'Password reset';

function email(label) {
  return `e2e-auth-${label}-${RUN_ID}@smoke.test`;
}

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

// Asserts that a request is refused. When `status` is given the response status must match exactly,
// which distinguishes an auth/scope rejection (401) from a request-body validation error (400).
async function expectRejected(method, path, {body, token, status} = {}) {
  const headers = {'Content-Type': 'application/json'};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  assert(!res.ok, `${method} ${path} must be rejected, but got ${res.status}`);
  if (status !== undefined) {
    assert(res.status === status, `${method} ${path} expected status ${status}, got ${res.status}`);
  }
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(`Assertion failed: ${message}`);
  }
}

function decodeJwt(token) {
  const payload = token.split('.')[1];
  return JSON.parse(Buffer.from(payload, 'base64url').toString('utf8'));
}

function extractJwt(text) {
  const match = /[?&]jwt=([A-Za-z0-9._-]+)/.exec(text ?? '');
  assert(match, 'expected a jwt link in the email body');
  return match[1];
}

async function mailpit(path) {
  const res = await fetch(`${MAILPIT_URL}${path}`);
  if (!res.ok) {
    throw new Error(`Mailpit ${path} → ${res.status}`);
  }
  return res.json();
}

// Polls Mailpit for the newest message to `to` whose subject starts with `subject` and whose ID is not in
// `excludeIds`, then returns the full message detail (incl. Text/HTML bodies).
async function waitForEmail({to, subject, excludeIds = new Set(), timeoutMs = 10_000}) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const {messages} = await mailpit('/api/v1/messages?limit=200');
    const hit = messages.find(
      (m) =>
        m.To?.some((addr) => addr.Address?.toLowerCase() === to.toLowerCase()) &&
        m.Subject?.startsWith(subject) &&
        !excludeIds.has(m.ID)
    );
    if (hit) {
      return mailpit(`/api/v1/message/${hit.ID}`);
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`timed out waiting for email to ${to} with subject "${subject}"`);
}

async function messageIdsFor({to, subject}) {
  const {messages} = await mailpit('/api/v1/messages?limit=200');
  return new Set(
    messages
      .filter(
        (m) => m.To?.some((addr) => addr.Address?.toLowerCase() === to.toLowerCase()) && m.Subject?.startsWith(subject)
      )
      .map((m) => m.ID)
  );
}

// Registers a user, confirms their email if verification is required, and returns a logged-in token.
async function registerVerifiedUser(addr) {
  const reg = await request('POST', '/api/v1/auth/register', {
    body: {name: 'E2E Auth', email: addr, password: PASSWORD},
  });
  assert(reg?.token, 'register must return a token');
  if (decodeJwt(reg.token).email_verified === false) {
    const mail = await waitForEmail({to: addr, subject: VERIFY_SUBJECT});
    await request('POST', '/api/v1/auth/verify/confirm', {token: extractJwt(mail.Text)});
  }
  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(login?.token, 'login must return a token');
  return login.token;
}

async function invite(adminToken, addr, name) {
  const res = await request('POST', '/api/v1/user-accounts', {
    token: adminToken,
    body: {email: addr, name, userRole: 'admin'},
  });
  assert(res?.inviteUrl, 'invite must return an inviteUrl');
  return res;
}

const results = [];
async function step(name, fn) {
  process.stdout.write(`• ${name} ... `);
  try {
    await fn();
    console.log('PASS');
    results.push({name, ok: true});
  } catch (e) {
    console.log('FAIL');
    console.log(`    ${e.message}`);
    results.push({name, ok: false, error: e});
  }
}

console.log(`Distr auth E2E test — run ${RUN_ID}`);
console.log(`  hub:     ${BASE_URL}`);
console.log(`  mailpit: ${MAILPIT_URL}\n`);

// The inviter admin is set up by the registration flow and reused by the invite/resend flows.
let adminToken;
let verificationRequired = true;

await step('registration: token + verification mail, email confirmation, login', async () => {
  const addr = email('register');
  const reg = await request('POST', '/api/v1/auth/register', {
    body: {name: 'Register Flow', email: addr, password: PASSWORD},
  });
  assert(reg?.token, 'register response must include a login token');
  verificationRequired = decodeJwt(reg.token).email_verified === false;
  console.log(`\n    (email verification required: ${verificationRequired})`);

  const status = await request('GET', '/api/v1/auth/status', {token: reg.token});
  assert(status?.active === true, 'status must report active=true after registration');

  if (verificationRequired) {
    const mail = await waitForEmail({to: addr, subject: VERIFY_SUBJECT});
    const verifyToken = extractJwt(mail.Text);
    assert(decodeJwt(verifyToken).email_verified === true, 'verification token must carry email_verified=true');
    assert(mail.Text.includes('copy and paste'), 'verification mail must contain the copy/paste URL section');
    await request('POST', '/api/v1/auth/verify/confirm', {token: verifyToken});
  } else {
    assert(decodeJwt(reg.token).email_verified === true, 'verification disabled → token already verified');
  }

  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(decodeJwt(login.token).email_verified === true, 'after confirmation the login token is verified');
  adminToken = login.token;
});

await step('resend verification mail: /auth/verify/request delivers a new verification mail', async () => {
  const addr = email('verify-resend');
  const reg = await request('POST', '/api/v1/auth/register', {
    body: {name: 'Verify Resend', email: addr, password: PASSWORD},
  });
  assert(reg?.token, 'register must return a token');

  // registration already sent one verification mail; capture it so we can detect the resend
  const before = await messageIdsFor({to: addr, subject: VERIFY_SUBJECT});
  await request('POST', '/api/v1/auth/verify/request', {token: reg.token});
  const mail = await waitForEmail({to: addr, subject: VERIFY_SUBJECT, excludeIds: before});
  const verifyToken = extractJwt(mail.Text);
  assert(decodeJwt(verifyToken).email_verified === true, 'resent verification token must be verified');

  // once the email is verified, requesting again is a no-op (the user is already verified)
  await request('POST', '/api/v1/auth/verify/confirm', {token: verifyToken});
  await request('POST', '/api/v1/auth/verify/request', {token: reg.token});
  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(decodeJwt(login.token).email_verified === true, 'user is verified after confirming the resent mail');
});

await step('invite via EMAIL link (email_verified=true): accept logs in directly', async () => {
  const addr = email('invite-mail');
  const created = await invite(adminToken, addr, 'Invitee Mail');
  assert(
    decodeJwt(extractJwt(created.inviteUrl)).email_verified === false,
    'response inviteUrl token must be email_verified=false'
  );

  const mail = await waitForEmail({to: addr, subject: INVITE_SUBJECT});
  assert(mail.Text.includes('copy and paste'), 'invite mail must contain the copy/paste URL section');
  const emailToken = extractJwt(mail.Text);
  assert(decodeJwt(emailToken).email_verified === true, 'email invite token must be email_verified=true');

  const accept = await request('POST', '/api/v1/auth/invite/accept', {
    token: emailToken,
    body: {name: 'Invitee Mail', password: PASSWORD},
  });
  assert(accept?.token, 'accept must return a login token');
  assert(decodeJwt(accept.token).email_verified === true, 'accept token must be verified (came via email)');

  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(decodeJwt(login.token).email_verified === true, 'invitee can log in with the verified account');
});

await step('invite via RESPONSE url (email_verified=false): accept triggers verification mail', async () => {
  const addr = email('invite-resp');
  const verifyMailsBefore = await messageIdsFor({to: addr, subject: VERIFY_SUBJECT});
  const created = await invite(adminToken, addr, 'Invitee Resp');
  const responseToken = extractJwt(created.inviteUrl);
  assert(decodeJwt(responseToken).email_verified === false, 'response inviteUrl token must be email_verified=false');

  const accept = await request('POST', '/api/v1/auth/invite/accept', {
    token: responseToken,
    body: {name: 'Invitee Resp', password: PASSWORD},
  });
  assert(accept?.token, 'accept must return a login token');

  if (verificationRequired) {
    assert(
      decodeJwt(accept.token).email_verified === false,
      'accept token must be unverified → frontend routes to /verify'
    );
    const mail = await waitForEmail({to: addr, subject: VERIFY_SUBJECT, excludeIds: verifyMailsBefore});
    const verifyToken = extractJwt(mail.Text);
    assert(decodeJwt(verifyToken).email_verified === true, 'auto-sent verification token must be verified');
    await request('POST', '/api/v1/auth/verify/confirm', {token: verifyToken});
  } else {
    assert(decodeJwt(accept.token).email_verified === true, 'verification disabled → logged in directly');
  }

  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(decodeJwt(login.token).email_verified === true, 'invitee can log in after verifying');
});

await step('password reset: confirm sets new password, verifies email, logs in', async () => {
  const addr = email('reset');
  await registerVerifiedUser(addr);

  await request('POST', '/api/v1/auth/reset', {body: {email: addr}});
  const mail = await waitForEmail({to: addr, subject: RESET_SUBJECT});
  const claims = decodeJwt(extractJwt(mail.Text));
  assert(claims.scope === 'password_reset', 'reset token must carry scope=password_reset');
  assert(claims.email_verified === true, 'reset token must carry email_verified=true');

  const confirm = await request('POST', '/api/v1/auth/reset/confirm', {
    token: extractJwt(mail.Text),
    body: {password: NEW_PASSWORD},
  });
  assert(confirm?.token, 'reset confirm must return a login token');
  assert(decodeJwt(confirm.token).email_verified === true, 'reset confirm token must be verified');

  const ok = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: NEW_PASSWORD}});
  assert(ok?.token, 'login with the new password must succeed');

  let oldLoginFailed = false;
  try {
    await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  } catch {
    oldLoginFailed = true;
  }
  assert(oldLoginFailed, 'login with the old password must fail');
});

await step('token scope: reset/invite endpoints reject unscoped and mismatched tokens', async () => {
  // An unscoped, org-scoped login token (a regular session) must not be usable to change a password.
  const loginAddr = email('scope-login');
  const loginToken = await registerVerifiedUser(loginAddr);
  assert(decodeJwt(loginToken).scope === undefined, 'a regular login token must not carry a scope claim');

  // A password-reset-scoped token.
  const resetAddr = email('scope-reset');
  await registerVerifiedUser(resetAddr);
  await request('POST', '/api/v1/auth/reset', {body: {email: resetAddr}});
  const resetMail = await waitForEmail({to: resetAddr, subject: RESET_SUBJECT});
  const resetToken = extractJwt(resetMail.Text);
  assert(decodeJwt(resetToken).scope === 'password_reset', 'reset token must carry scope=password_reset');

  // An invite-scoped token.
  const inviteAddr = email('scope-invite');
  const created = await invite(adminToken, inviteAddr, 'Scope Invitee');
  const inviteToken = extractJwt(created.inviteUrl);
  assert(decodeJwt(inviteToken).scope === 'invite', 'invite token must carry scope=invite');

  // The unscoped login token is refused (401) by both password-setting endpoints.
  await expectRejected('POST', '/api/v1/auth/reset/confirm', {
    token: loginToken,
    body: {password: NEW_PASSWORD},
    status: 401,
  });
  await expectRejected('POST', '/api/v1/auth/invite/accept', {
    token: loginToken,
    body: {name: 'Scope Login', password: NEW_PASSWORD},
    status: 401,
  });

  // A reset token cannot accept an invite, and an invite token cannot confirm a reset.
  await expectRejected('POST', '/api/v1/auth/invite/accept', {
    token: resetToken,
    body: {name: 'Scope Reset', password: NEW_PASSWORD},
    status: 401,
  });
  await expectRejected('POST', '/api/v1/auth/reset/confirm', {
    token: inviteToken,
    body: {password: NEW_PASSWORD},
    status: 401,
  });

  // The correctly scoped tokens are still accepted by their own endpoints.
  const accepted = await request('POST', '/api/v1/auth/invite/accept', {
    token: inviteToken,
    body: {name: 'Scope Invitee', password: PASSWORD},
  });
  assert(accepted?.token, 'invite token must be accepted by /auth/invite/accept');

  const confirmed = await request('POST', '/api/v1/auth/reset/confirm', {
    token: resetToken,
    body: {password: NEW_PASSWORD},
  });
  assert(confirmed?.token, 'reset token must be accepted by /auth/reset/confirm');
});

await step('resend invitation: a second invitation mail is delivered', async () => {
  const addr = email('resend');
  const created = await invite(adminToken, addr, 'Invitee Resend');
  const firstMail = await waitForEmail({to: addr, subject: INVITE_SUBJECT});

  const resend = await request('POST', `/api/v1/user-accounts/${created.user.id}/invite`, {token: adminToken});
  assert(resend?.inviteUrl, 'resend must return an inviteUrl');

  const secondMail = await waitForEmail({to: addr, subject: INVITE_SUBJECT, excludeIds: new Set([firstMail.ID])});
  assert(secondMail.ID !== firstMail.ID, 'resend must deliver a new invitation mail');
});

await step('password reset for an orphaned user (no org) resurrects them with a personal org', async () => {
  // Create the user via invitation into the admin org, then remove them from it → the account persists with
  // zero organizations. A reset must still succeed (mirroring regular login, which creates a personal org).
  const addr = email('orphan');
  const created = await invite(adminToken, addr, 'Orphan User');
  await request('DELETE', `/api/v1/user-accounts/${created.user.id}`, {token: adminToken});

  await request('POST', '/api/v1/auth/reset', {body: {email: addr}});
  const mail = await waitForEmail({to: addr, subject: RESET_SUBJECT});
  const confirm = await request('POST', '/api/v1/auth/reset/confirm', {
    token: extractJwt(mail.Text),
    body: {password: PASSWORD},
  });
  assert(confirm?.token, 'reset confirm must return a login token even when the user had no organization');
  assert(decodeJwt(confirm.token).org, 'login token must be scoped to a (newly created) organization');

  const login = await request('POST', '/api/v1/auth/login', {body: {email: addr, password: PASSWORD}});
  assert(login?.token, 'orphaned user can log in after reset');
});

const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} steps passed.`);
if (failed.length > 0) {
  process.exit(1);
}
