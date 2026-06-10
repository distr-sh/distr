#!/usr/bin/env node
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';
import {createInterface} from 'node:readline/promises';
import {fileURLToPath} from 'node:url';

import {chromium} from 'playwright';
import sharp from 'sharp';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const websiteRoot = path.resolve(__dirname, '..');

const fullHd = {width: 1920, height: 1080};
const zoom = 1.25;
const viewport = {
  width: Math.round(fullHd.width / zoom),
  height: Math.round(fullHd.height / zoom),
};

const screenshots = [
  {
    id: 'distr-deployments',
    title: 'Deployment Overview Status Dashboard incl. Logs',
    persona: 'vendor',
    route: '/deployments',
    description:
      'Vendor deployment overview with grouped customer deployment targets, application versions, health, CPU/memory gauges, deploy/update/inspect actions, and the current sidebar.',
    checks: [
      /Distr Vendor Portal/,
      /PoC VM/,
      /Production Cluster/,
      /Deploy App/,
    ],
  },
  {
    id: 'distr-artifacts',
    title: 'Registry Downloads',
    persona: 'vendor',
    route: '/artifact-pulls',
    description:
      'Vendor Registry > Downloads table showing pull date, customer, user, address, artifact, and version so it is clear who downloaded what and when.',
    action: async page => {
      await page
        .getByText(/DATE\s+CUSTOMER\s+USER\s+ADDRESS\s+ARTIFACT\s+VERSION/)
        .first()
        .scrollIntoViewIfNeeded({timeout: 5000})
        .catch(() => {});
      await page.waitForTimeout(400);
    },
    checks: [
      /DATE\s+CUSTOMER\s+USER\s+ADDRESS\s+ARTIFACT\s+VERSION/,
      /hello-world|hello-distr/,
    ],
  },
  {
    id: 'distr-artifact-licenses',
    title: 'Artifact Entitlements',
    persona: 'vendor',
    route: '/licenses/ebdc5a9f-ea69-4344-9006-0ec1d882141a',
    description:
      'Vendor license-management page with the artifact-entitlement drawer open and the artifact tag selector expanded.',
    action: async page => {
      const rowEdit = page
        .locator('tr')
        .filter({hasText: 'xcorp'})
        .first()
        .getByRole('button', {name: /^Edit$/});

      if (await rowEdit.count()) {
        await rowEdit.click();
      } else {
        const edits = page.getByRole('button', {name: /^Edit$/});
        if ((await edits.count()) < 9) {
          throw new Error('Could not find artifact entitlement edit button.');
        }
        await edits.nth(8).click();
      }

      await page.waitForTimeout(800);
      const tagButton = page
        .getByRole('button', {name: /tags? selected/i})
        .first();
      if (!(await tagButton.count())) {
        throw new Error('Could not find artifact tag selector.');
      }
      await tagButton.click();
      await page.waitForTimeout(500);
    },
    checks: [/MANAGE ARTIFACT ENTITLEMENT/, /Artifact Tags \*/, /Save/],
  },
  {
    id: 'distr-customer-portal-artifacts',
    title: 'White-labeled Customer Portal Home',
    persona: 'customer',
    route: '/home',
    description:
      'Customer-facing white-label home page with onboarding instructions and install snippets visible.',
    checks: [
      /White Label Example/,
      /Welcome to Your Software Update Hub/,
      /Install dependencies/,
    ],
    copyLightToDefault: true,
  },
  {
    id: 'distr-dashboard',
    title: 'Single pane of glass dashboard',
    persona: 'vendor',
    route: '/dashboard',
    description:
      'Vendor dashboard with support bundles, agent/deployment cards, health states, customer names, and current navigation.',
    checks: [/Support Bundles/, /Agents/, /BYOC Global/, /Healthy/],
  },
  {
    id: 'distr-customer-portal',
    title: 'Customer Portal New Deployment Flow',
    persona: 'customer',
    route: '/deployments',
    description:
      'Customer deployments page with the Create New Deployment modal open on the application-selection step.',
    action: async page => {
      await page.getByRole('button', {name: /New Deployment/i}).click();
      await page.waitForTimeout(1000);
    },
    checks: [
      /Create New Deployment/,
      /Select Application/,
      /Hello Distr/,
      /Continue/,
    ],
  },
  {
    id: 'distr-log-viewer',
    title: 'Deployment target log viewer',
    persona: 'vendor',
    route: '/deployments/2ad1125e-1d38-4457-80bf-5c8d043686a8',
    description:
      'Vendor deployment-target Agent Logs view with timestamps, log levels, source files, metric records, filters, sort, and export controls.',
    checks: [
      /Agent Logs/,
      /docker\/metrics\.go|docker\/main\.go/,
      /Export all/,
    ],
  },
];

function parseArgs(argv) {
  const options = {
    baseUrl: process.env.DISTR_SCREENSHOT_BASE_URL ?? 'https://demo.distr.sh',
    outDir: path.join(websiteRoot, 'src/assets/screenshots/distr'),
    storageDir: path.join(
      process.env.TMPDIR ?? '/tmp',
      'distr-homepage-screenshots',
    ),
    headed: false,
    reuseStorage: false,
    list: false,
    help: false,
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--') continue;
    if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg === '--list') options.list = true;
    else if (arg === '--headed') options.headed = true;
    else if (arg === '--reuse-storage') options.reuseStorage = true;
    else if (arg === '--base-url') options.baseUrl = argv[++i];
    else if (arg === '--out-dir') options.outDir = path.resolve(argv[++i]);
    else if (arg === '--storage-dir')
      options.storageDir = path.resolve(argv[++i]);
    else throw new Error(`Unknown argument: ${arg}`);
  }

  options.baseUrl = options.baseUrl.replace(/\/$/, '');
  return options;
}

function printHelp() {
  console.log(`Capture homepage screenshots from demo.distr.sh.

Usage:
  pnpm screenshots:homepage
  pnpm screenshots:homepage -- --out-dir /tmp/distr-screenshots

Options:
  --base-url <url>      Demo URL. Defaults to https://demo.distr.sh.
  --out-dir <dir>       Output directory. Defaults to src/assets/screenshots/distr.
  --storage-dir <dir>   Temporary Playwright storage-state directory.
  --reuse-storage       Reuse existing storage states in --storage-dir.
  --headed              Run Chromium with a visible browser window.
  --list                Print the screenshot plan and exit.
  --help                Show this help.

Credentials:
  The script prompts for credentials when they are not provided through env vars.
  It never writes credentials to the repo.

Environment variables:
  DISTR_VENDOR_EMAIL
  DISTR_CUSTOMER_EMAIL
  DISTR_VENDOR_PASSWORD
  DISTR_CUSTOMER_PASSWORD
  DISTR_DEMO_PASSWORD       Shared fallback password for both accounts.
  DISTR_SCREENSHOT_BASE_URL Optional base URL.
`);
}

function printList() {
  console.log('Homepage screenshot plan:\n');
  for (const screenshot of screenshots) {
    console.log(`${screenshot.id}`);
    console.log(`  title: ${screenshot.title}`);
    console.log(`  login: ${screenshot.persona}`);
    console.log(`  route: ${screenshot.route}`);
    console.log(`  visible: ${screenshot.description}`);
    console.log('');
  }
}

async function prompt(question) {
  const rl = createInterface({input: process.stdin, output: process.stdout});
  const answer = await rl.question(question);
  rl.close();
  return answer.trim();
}

async function promptSecret(question) {
  if (!process.stdin.isTTY || !process.stdin.setRawMode) {
    return prompt(question);
  }

  process.stdout.write(question);
  process.stdin.setRawMode(true);
  process.stdin.resume();

  return new Promise((resolve, reject) => {
    let value = '';

    const cleanup = () => {
      process.stdin.setRawMode(false);
      process.stdin.off('data', onData);
      process.stdout.write('\n');
    };

    const onData = buffer => {
      for (const text of buffer.toString('utf8')) {
        if (text === '\u0003') {
          cleanup();
          reject(new Error('Aborted.'));
          return;
        }
        if (text === '\r' || text === '\n') {
          cleanup();
          resolve(value);
          return;
        }
        if (text === '\u007f') {
          value = value.slice(0, -1);
          continue;
        }
        value += text;
      }
    };

    process.stdin.on('data', onData);
  });
}

async function getCredentials() {
  const vendorEmail =
    process.env.DISTR_VENDOR_EMAIL ?? (await prompt('Vendor email: '));
  const customerEmail =
    process.env.DISTR_CUSTOMER_EMAIL ?? (await prompt('Customer email: '));

  const sharedPassword = process.env.DISTR_DEMO_PASSWORD;
  const vendorPassword =
    process.env.DISTR_VENDOR_PASSWORD ??
    sharedPassword ??
    (await promptSecret('Vendor password: '));
  const customerPassword =
    process.env.DISTR_CUSTOMER_PASSWORD ?? sharedPassword ?? vendorPassword;

  return {
    vendor: {email: vendorEmail, password: vendorPassword},
    customer: {email: customerEmail, password: customerPassword},
  };
}

async function exists(filePath) {
  try {
    await fs.access(filePath);
    return true;
  } catch {
    return false;
  }
}

async function login({browser, baseUrl, storagePath, email, password}) {
  const context = await browser.newContext({
    viewport,
    deviceScaleFactor: zoom,
  });
  const page = await context.newPage();
  await page.goto(`${baseUrl}/login`, {waitUntil: 'domcontentloaded'});
  await page.getByLabel(/email/i).fill(email);
  await page.getByLabel(/password/i).fill(password);
  await page.getByRole('button', {name: /sign in|log in|login/i}).click();
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(1200);

  if (new URL(page.url()).pathname.includes('/login')) {
    const body = await page
      .locator('body')
      .innerText()
      .catch(() => '');
    throw new Error(
      `Login failed. Page text starts with: ${body.slice(0, 120)}`,
    );
  }

  await context.storageState({path: storagePath});
  await context.close();
}

async function ensureStorageStates({browser, options}) {
  await fs.mkdir(options.storageDir, {recursive: true});
  const storage = {
    vendor: path.join(options.storageDir, 'vendor-storage.json'),
    customer: path.join(options.storageDir, 'customer-storage.json'),
  };

  if (
    options.reuseStorage &&
    (await exists(storage.vendor)) &&
    (await exists(storage.customer))
  ) {
    console.log(`Reusing storage states from ${options.storageDir}`);
    return storage;
  }

  const credentials = await getCredentials();

  console.log('Logging into vendor portal...');
  await login({
    browser,
    baseUrl: options.baseUrl,
    storagePath: storage.vendor,
    ...credentials.vendor,
  });

  console.log('Logging into customer portal...');
  await login({
    browser,
    baseUrl: options.baseUrl,
    storagePath: storage.customer,
    ...credentials.customer,
  });

  return storage;
}

async function createContext({browser, storageState, scheme}) {
  const context = await browser.newContext({
    storageState,
    viewport,
    deviceScaleFactor: zoom,
  });

  await context.addInitScript(selectedScheme => {
    localStorage.setItem(
      'COLOR_SCHEME',
      selectedScheme === 'dark' ? 'dark' : '',
    );
  }, scheme);

  return context;
}

async function assertVisibleText(page, screenshot, scheme) {
  const bodyText = await page.locator('body').innerText({timeout: 5000});
  for (const check of screenshot.checks) {
    if (!check.test(bodyText)) {
      throw new Error(
        `${screenshot.id}-${scheme} missing required text: ${check}`,
      );
    }
  }

  const isDark = await page.evaluate(() =>
    document.body.classList.contains('dark'),
  );
  if (scheme === 'dark' && !isDark) {
    throw new Error(`${screenshot.id}-dark did not enable dark mode.`);
  }
  if (scheme === 'light' && isDark) {
    throw new Error(`${screenshot.id}-light unexpectedly enabled dark mode.`);
  }
}

async function captureScreenshot({
  browser,
  options,
  storage,
  screenshot,
  scheme,
}) {
  const storageState = storage[screenshot.persona];
  const context = await createContext({browser, storageState, scheme});
  const page = await context.newPage();
  await page.goto(`${options.baseUrl}${screenshot.route}`, {
    waitUntil: 'domcontentloaded',
  });
  await page.waitForLoadState('networkidle').catch(() => {});
  await page.waitForTimeout(1400);

  if (screenshot.action) {
    await screenshot.action(page);
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(400);
  }

  await assertVisibleText(page, screenshot, scheme);

  const pngPath = path.join(
    options.storageDir,
    `${screenshot.id}-${scheme}.png`,
  );
  const webpPath = path.join(options.outDir, `${screenshot.id}-${scheme}.webp`);

  await page.screenshot({
    path: pngPath,
    fullPage: false,
    animations: 'disabled',
    scale: 'device',
  });
  await context.close();

  const meta = await sharp(pngPath).metadata();
  if (meta.width !== fullHd.width || meta.height !== fullHd.height) {
    throw new Error(
      `${screenshot.id}-${scheme} is ${meta.width}x${meta.height}; expected ${fullHd.width}x${fullHd.height}.`,
    );
  }

  await sharp(pngPath).webp({quality: 88, effort: 6}).toFile(webpPath);
  await fs.rm(pngPath, {force: true});

  console.log(`Wrote ${path.relative(websiteRoot, webpPath)}`);
}

async function main() {
  const options = parseArgs(process.argv.slice(2));

  if (options.help) {
    printHelp();
    return;
  }

  if (options.list) {
    printList();
    return;
  }

  await fs.mkdir(options.outDir, {recursive: true});
  await fs.mkdir(options.storageDir, {recursive: true});

  const browser = await chromium.launch({headless: !options.headed});
  try {
    const storage = await ensureStorageStates({browser, options});
    for (const scheme of ['light', 'dark']) {
      for (const screenshot of screenshots) {
        await captureScreenshot({
          browser,
          options,
          storage,
          screenshot,
          scheme,
        });
      }
    }

    for (const screenshot of screenshots) {
      if (screenshot.copyLightToDefault) {
        const source = path.join(options.outDir, `${screenshot.id}-light.webp`);
        const target = path.join(options.outDir, `${screenshot.id}.webp`);
        await fs.copyFile(source, target);
        console.log(`Wrote ${path.relative(websiteRoot, target)}`);
      }
    }
  } finally {
    await browser.close();
  }
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
