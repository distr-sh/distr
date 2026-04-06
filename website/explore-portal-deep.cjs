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

  // Click "View License" on second license (the scoped one)
  console.log('\n=== License Detail ===');
  await page.goto(`${BASE}/license-keys`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  const viewButtons = await page.$$(
    'button:has-text("View License"), a:has-text("View License")',
  );
  console.log(`Found ${viewButtons.length} View License buttons`);

  if (viewButtons.length > 1) {
    await viewButtons[1].click(); // click the scoped one
    await page.waitForTimeout(2000);
    console.log('License detail URL:', page.url());

    const licenseDetail = await page.evaluate(() => {
      return document.body.textContent.trim().substring(0, 2000);
    });
    console.log('Content:', licenseDetail);
    await page.screenshot({
      path: `${OUT_DIR}/deep-license-detail.png`,
      fullPage: true,
    });
  }

  // Click on artifact to see detail/tags
  console.log('\n=== Artifact Detail ===');
  await page.goto(`${BASE}/artifacts`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  // Try clicking the artifact name/row
  const artifactEl = await page.$('text=hello-distr/proxy');
  if (artifactEl) {
    await artifactEl.click();
    await page.waitForTimeout(2000);
    console.log('After clicking artifact:', page.url());

    const artifactDetail = await page.evaluate(() => {
      return document.body.textContent.trim().substring(0, 2000);
    });
    console.log('Content:', artifactDetail);
    await page.screenshot({
      path: `${OUT_DIR}/deep-artifact-detail.png`,
      fullPage: true,
    });
  } else {
    console.log('Could not find artifact to click');
    // Try clicking any row
    const row = await page.$('mat-card, tr, [class*="row"], [class*="item"]');
    if (row) {
      await row.click();
      await page.waitForTimeout(2000);
      console.log('After clicking row:', page.url());
      await page.screenshot({
        path: `${OUT_DIR}/deep-artifact-detail.png`,
        fullPage: true,
      });
    }
  }

  // Check the deployments page - click on an actual deployment if one exists
  console.log('\n=== Deployments Detail ===');
  await page.goto(`${BASE}/deployments`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  const deploymentsContent = await page.evaluate(() => {
    return document.body.textContent.trim().substring(0, 2000);
  });
  console.log('Deployments content:', deploymentsContent);
  await page.screenshot({
    path: `${OUT_DIR}/deep-deployments.png`,
    fullPage: true,
  });

  // Try clicking on user menu (the avatar icon in top-right)
  console.log('\n=== User Menu ===');
  await page.goto(`${BASE}/home`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1000);

  // Look for all buttons in the header area
  const headerButtons = await page.evaluate(() => {
    const buttons = [];
    document.querySelectorAll('button, [role="button"]').forEach(b => {
      buttons.push({
        text: b.textContent.trim().substring(0, 50),
        ariaLabel: b.getAttribute('aria-label') || '',
        className: b.className.substring(0, 100),
        matTooltip:
          b.getAttribute('mattooltip') ||
          b.getAttribute('ng-reflect-message') ||
          '',
      });
    });
    return buttons;
  });
  console.log('All buttons:');
  headerButtons.forEach(b =>
    console.log(
      `  "${b.text}" aria="${b.ariaLabel}" class="${b.className}" tooltip="${b.matTooltip}"`,
    ),
  );

  // Try to find and click the user menu
  const menuBtn = await page.$(
    'button[aria-label*="user" i], button[aria-label*="menu" i], button[mattooltip*="user" i], button:has(mat-icon:text("person")), button:has(mat-icon:text("account_circle"))',
  );
  if (menuBtn) {
    await menuBtn.click();
    await page.waitForTimeout(1000);
    console.log('User menu opened');

    const menuContent = await page.evaluate(() => {
      const overlay = document.querySelector('.cdk-overlay-container');
      return overlay
        ? overlay.textContent.trim().substring(0, 500)
        : '(no overlay)';
    });
    console.log('Menu content:', menuContent);
    await page.screenshot({
      path: `${OUT_DIR}/deep-user-menu.png`,
      fullPage: true,
    });
  } else {
    // Click by tooltip text
    const allBtns = await page.$$('button');
    for (const btn of allBtns) {
      const label = await btn.getAttribute('aria-label');
      const tooltip = await btn.getAttribute('mattooltip');
      if (label && label.toLowerCase().includes('user')) {
        await btn.click();
        await page.waitForTimeout(1000);
        await page.screenshot({
          path: `${OUT_DIR}/deep-user-menu.png`,
          fullPage: true,
        });
        break;
      }
    }
  }

  // Try to find PAT/token generation
  console.log('\n=== PAT / Token Settings ===');
  // Check if there's a settings or tokens page in user menu
  const settingsRoutes = [
    '/settings',
    '/tokens',
    '/pat',
    '/access-tokens',
    '/personal-access-tokens',
    '/profile',
  ];
  for (const route of settingsRoutes) {
    const resp = await page.goto(`${BASE}${route}`, {
      waitUntil: 'networkidle',
      timeout: 5000,
    });
    const finalUrl = page.url();
    if (!finalUrl.includes('/home') && !finalUrl.includes('/login')) {
      console.log(`Found settings at ${route} -> ${finalUrl}`);
      await page.waitForTimeout(1000);
      const content = await page.evaluate(() =>
        document.body.textContent.trim().substring(0, 1000),
      );
      console.log('Content:', content);
      await page.screenshot({
        path: `${OUT_DIR}/deep-settings.png`,
        fullPage: true,
      });
      break;
    }
  }

  await browser.close();
  console.log('\nDone!');
})();
