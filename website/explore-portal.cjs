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
  console.log('Navigating to login...');
  await page.goto(`${BASE}/login`);
  await page.waitForLoadState('networkidle');

  // Find and fill login form
  const emailInput =
    (await page.$('input[type="email"]')) ||
    (await page.$('input[formcontrolname="email"]')) ||
    (await page.$('input[name="email"]'));
  const passInput =
    (await page.$('input[type="password"]')) ||
    (await page.$('input[formcontrolname="password"]')) ||
    (await page.$('input[name="password"]'));

  if (emailInput && passInput) {
    await emailInput.fill(CUSTOMER_EMAIL);
    await passInput.fill(CUSTOMER_PASS);
    await page.click('button[type="submit"]');
  } else {
    console.log('Could not find login fields. Page content:');
    console.log(await page.content().then(c => c.substring(0, 2000)));
  }

  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(3000);

  console.log('Current URL after login:', page.url());
  await page.screenshot({path: `${OUT_DIR}/01-home.png`, fullPage: true});

  // Get all links on the page
  const allLinks = await page.evaluate(() => {
    const results = [];
    document.querySelectorAll('a').forEach(a => {
      const href = a.getAttribute('href') || '';
      const text = a.textContent.trim().substring(0, 80);
      if (text) results.push({href, text});
    });
    return results;
  });
  console.log('\n=== All links on page ===');
  allLinks.forEach(l => console.log(`  ${l.text} -> ${l.href}`));

  // Get sidebar/nav structure more specifically
  const pageStructure = await page.evaluate(() => {
    const result = {};
    // Get all visible text from sidebar area
    const sidebar = document.querySelector(
      'mat-sidenav, nav, [class*="sidebar"], [class*="drawer"], [class*="menu"]',
    );
    if (sidebar) {
      result.sidebarHTML = sidebar.innerHTML.substring(0, 3000);
      result.sidebarText = sidebar.textContent.trim().substring(0, 1000);
    }
    // Get all mat-list-item or similar
    const menuItems = [];
    document
      .querySelectorAll(
        'mat-list-item, [mat-list-item], a[routerlink], [routerlink]',
      )
      .forEach(el => {
        menuItems.push({
          text: el.textContent.trim().substring(0, 80),
          routerLink:
            el.getAttribute('routerlink') ||
            el.getAttribute('ng-reflect-router-link') ||
            '',
          href: el.getAttribute('href') || '',
        });
      });
    result.menuItems = menuItems;
    return result;
  });

  console.log('\n=== Sidebar text ===');
  console.log(pageStructure.sidebarText || '(none found)');
  console.log('\n=== Menu items with routerLinks ===');
  (pageStructure.menuItems || []).forEach(m =>
    console.log(`  ${m.text} -> routerLink: ${m.routerLink} href: ${m.href}`),
  );

  // Try navigating to common customer portal routes
  const routes = [
    '/deployments',
    '/artifacts',
    '/registry',
    '/licenses',
    '/license',
    '/support',
    '/support-bundles',
    '/secrets',
    '/settings',
    '/metrics',
    '/quality',
    '/compliance',
    '/overview',
    '/home',
    '/dashboard',
    '/applications',
    '/app',
    '/entitlements',
  ];

  let idx = 2;
  for (const path of routes) {
    try {
      const url = `${BASE}${path}`;
      const resp = await page.goto(url, {
        waitUntil: 'networkidle',
        timeout: 8000,
      });
      const status = resp?.status();
      const finalUrl = page.url();

      if (
        status === 200 &&
        !finalUrl.includes('/login') &&
        !finalUrl.includes('/404')
      ) {
        console.log(`\n✅ ${path} -> ${finalUrl} (${status})`);
        await page.waitForTimeout(1000);

        // Get page heading/title
        const heading = await page.evaluate(() => {
          const h = document.querySelector(
            'h1, h2, [class*="title"], [class*="header"]',
          );
          return h ? h.textContent.trim().substring(0, 100) : '(no heading)';
        });
        console.log(`   Heading: ${heading}`);

        await page.screenshot({
          path: `${OUT_DIR}/${String(idx).padStart(2, '0')}-${path.replace(/\//g, '') || 'root'}.png`,
          fullPage: true,
        });
        idx++;
      } else {
        console.log(`❌ ${path} -> ${finalUrl} (${status})`);
      }
    } catch (e) {
      console.log(`❌ ${path} -> Error: ${e.message.substring(0, 80)}`);
    }
  }

  await browser.close();
  console.log('\n\nDone! Screenshots saved to:', OUT_DIR);
})();
