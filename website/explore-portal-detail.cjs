const {chromium} = require('playwright');
const {mkdirSync} = require('fs');

const BASE = 'https://demo.distr.sh';
const CUSTOMER_EMAIL = 'donna+customer@glasskube.com';
const CUSTOMER_PASS = 'passwort';
const OUT_DIR =
  '/Users/james/.openclaw/workspace/distr-knowledge/portal-screenshots';

mkdirSync(OUT_DIR, {recursive: true});

(async () => {
  const browser = await chromium.launch({headless: true});
  const context = await browser.newContext({
    viewport: {width: 1400, height: 900},
  });
  const page = await context.newPage();

  // Login
  console.log('Logging in...');
  await page.goto(`${BASE}/login`);
  await page.waitForLoadState('networkidle');
  const emailInput =
    (await page.$('input[type="email"]')) ||
    (await page.$('input[formcontrolname="email"]'));
  const passInput =
    (await page.$('input[type="password"]')) ||
    (await page.$('input[formcontrolname="password"]'));
  await emailInput.fill(CUSTOMER_EMAIL);
  await passInput.fill(CUSTOMER_PASS);
  await page.click('button[type="submit"]');
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(3000);
  console.log('Logged in:', page.url());

  // Navigate to license-keys page (found in nav)
  console.log('\n=== License Keys ===');
  await page.goto(`${BASE}/license-keys`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);
  console.log('URL:', page.url());

  // Get page content
  const licenseContent = await page.evaluate(() => {
    return (
      document
        .querySelector('main, [class*="content"], [class*="main"]')
        ?.textContent?.trim()
        ?.substring(0, 500) ||
      document.body.textContent.trim().substring(0, 500)
    );
  });
  console.log('Content:', licenseContent);
  await page.screenshot({
    path: `${OUT_DIR}/detail-licenses.png`,
    fullPage: true,
  });

  // Navigate to Users page
  console.log('\n=== Users ===');
  await page.goto(`${BASE}/users`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);
  console.log('URL:', page.url());
  const usersContent = await page.evaluate(() => {
    return (
      document
        .querySelector('main, [class*="content"], [class*="main"]')
        ?.textContent?.trim()
        ?.substring(0, 500) ||
      document.body.textContent.trim().substring(0, 500)
    );
  });
  console.log('Content:', usersContent);
  await page.screenshot({path: `${OUT_DIR}/detail-users.png`, fullPage: true});

  // Click on the artifact to see details
  console.log('\n=== Artifact Detail ===');
  await page.goto(`${BASE}/artifacts`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  // Click on the first artifact
  const artifactLink = await page.$(
    'a[href*="artifact"], [class*="artifact"] a, mat-card a, table a, .mat-mdc-row',
  );
  if (artifactLink) {
    await artifactLink.click();
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500);
    console.log('Artifact detail URL:', page.url());
    await page.screenshot({
      path: `${OUT_DIR}/detail-artifact.png`,
      fullPage: true,
    });
  } else {
    // Try clicking any clickable row
    const clickable = await page.$(
      '[routerlink], tr[class*="clickable"], .clickable',
    );
    if (clickable) {
      await clickable.click();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1500);
      console.log('Artifact detail URL:', page.url());
    } else {
      console.log('No clickable artifact found');
    }
    await page.screenshot({
      path: `${OUT_DIR}/detail-artifact.png`,
      fullPage: true,
    });
  }

  // Get artifact page content
  const artifactContent = await page.evaluate(() => {
    return document.body.textContent.trim().substring(0, 1000);
  });
  console.log('Content:', artifactContent);

  // Check user settings/profile area (top-right user menu)
  console.log('\n=== User Menu / Settings ===');
  await page.goto(`${BASE}/home`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1000);

  // Click user avatar/profile icon
  const userMenu = await page.$(
    '[class*="user-menu"], [class*="avatar"], button[mat-icon-button]:last-child, [aria-label*="user"], [aria-label*="menu"]',
  );
  if (userMenu) {
    await userMenu.click();
    await page.waitForTimeout(1000);

    // Get menu items
    const menuItems = await page.evaluate(() => {
      const items = [];
      document
        .querySelectorAll(
          '[role="menuitem"], mat-menu-item, [mat-menu-item], .mat-menu-item, .cdk-overlay-container button, .cdk-overlay-container a',
        )
        .forEach(el => {
          items.push(el.textContent.trim().substring(0, 80));
        });
      return items;
    });
    console.log('User menu items:', menuItems);
    await page.screenshot({
      path: `${OUT_DIR}/detail-user-menu.png`,
      fullPage: true,
    });
  } else {
    console.log('No user menu button found');
  }

  // Check support page again in detail
  console.log('\n=== Support Page Detail ===');
  await page.goto(`${BASE}/support`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);
  console.log('URL:', page.url());
  const supportContent = await page.evaluate(() => {
    return document.body.textContent.trim().substring(0, 1000);
  });
  console.log('Content:', supportContent);

  // Check if support is a sub-section or has tabs
  const supportNav = await page.evaluate(() => {
    const tabs = [];
    document
      .querySelectorAll('[role="tab"], mat-tab, .mat-tab-label, [mat-tab-link]')
      .forEach(el => {
        tabs.push(el.textContent.trim());
      });
    return tabs;
  });
  console.log('Support tabs:', supportNav);

  await browser.close();
  console.log('\nDone!');
})();
