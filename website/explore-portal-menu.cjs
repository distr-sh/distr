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

  // Click the "Open user menu" button
  console.log('\n=== User Menu ===');
  const userMenuBtn = await page.$(
    'button[aria-label="Open user menu"], button:has-text("Open user menu")',
  );
  if (!userMenuBtn) {
    // Try finding it by the flex/rounded class
    const btns = await page.$$('button.flex.rounded-full');
    if (btns.length > 0) {
      console.log('Found user menu button by class');
      await btns[0].click();
    }
  } else {
    console.log('Found user menu button by aria-label');
    await userMenuBtn.click();
  }
  await page.waitForTimeout(1000);

  // Check for dropdown/menu
  const dropdownContent = await page.evaluate(() => {
    // Look for any dropdown/popover that appeared
    const elements = document.querySelectorAll(
      '[role="menu"], [class*="dropdown"], [class*="popover"], [class*="overlay"]',
    );
    const results = [];
    elements.forEach(el => {
      if (el.textContent.trim()) {
        results.push({
          tag: el.tagName,
          class: el.className.substring(0, 100),
          text: el.textContent.trim().substring(0, 500),
        });
      }
    });
    return results;
  });
  console.log('Dropdown elements:', JSON.stringify(dropdownContent, null, 2));
  await page.screenshot({
    path: `${OUT_DIR}/deep-user-menu-open.png`,
    fullPage: true,
  });

  // Now let's also check what happens when we click on the "Hello Distr" app in deployments
  console.log('\n=== Deployment Flow ===');
  await page.goto(`${BASE}/deployments`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  // Click "New Deployment"
  const newDeployBtn = await page.$('button:has-text("New Deployment")');
  if (newDeployBtn) {
    await newDeployBtn.click();
    await page.waitForTimeout(1500);
    await page.screenshot({
      path: `${OUT_DIR}/deep-deploy-step1.png`,
      fullPage: true,
    });

    // Click on Hello Distr app card, then Continue
    const appCard = await page.$('text=Hello Distr');
    if (appCard) {
      await appCard.click();
      await page.waitForTimeout(500);
      const continueBtn = await page.$('button:has-text("Continue")');
      if (continueBtn) {
        await continueBtn.click();
        await page.waitForTimeout(1500);
        console.log('Step 2 URL:', page.url());
        await page.screenshot({
          path: `${OUT_DIR}/deep-deploy-step2.png`,
          fullPage: true,
        });
      }
    }
  }

  // Check for support bundle creation form
  console.log('\n=== Support Bundle Creation ===');
  await page.goto(`${BASE}/support`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1500);

  const newBundleBtn = await page.$(
    'button:has-text("New Support Bundle"), button:has-text("Support Bundle")',
  );
  if (newBundleBtn) {
    await newBundleBtn.click();
    await page.waitForTimeout(2000);

    const bundleForm = await page.evaluate(() => {
      return document.body.textContent.trim().substring(0, 2000);
    });
    console.log('Bundle form content:', bundleForm);
    await page.screenshot({
      path: `${OUT_DIR}/deep-support-bundle-form.png`,
      fullPage: true,
    });
  } else {
    console.log('No support bundle button found');
  }

  // Check the login page itself (for the screenshot in docs)
  console.log('\n=== Login Page ===');
  // Logout first
  await page.goto(`${BASE}/login`, {waitUntil: 'networkidle'});
  await page.waitForTimeout(1000);
  await page.screenshot({path: `${OUT_DIR}/deep-login.png`, fullPage: true});
  console.log('Login page URL:', page.url());

  await browser.close();
  console.log('\nDone!');
})();
