// @ts-check
import fs from 'node:fs/promises';

/**
 * Writes s3-redirects.txt next to the build output, one line per redirect with
 * the S3 key of the page Astro builds for it and the redirect target. The
 * deploy sets each target as the object's website redirect, so the S3 website
 * endpoint answers with a 301 instead of serving the meta refresh page.
 *
 * @returns {import('astro').AstroIntegration}
 */
export function s3Redirects() {
  /** @type {import('astro').AstroConfig['redirects']} */
  let redirects = {};
  return {
    name: 's3-redirects',
    hooks: {
      'astro:config:done': ({config}) => {
        redirects = config.redirects;
      },
      'astro:build:done': async ({dir}) => {
        const lines = Object.entries(redirects).map(
          ([from, to]) =>
            `${from.replace(/^\//, '')}index.html ${typeof to === 'string' ? to : to.destination}`,
        );
        await fs.writeFile(
          new URL('../s3-redirects.txt', dir),
          lines.join('\n') + '\n',
        );
      },
    },
  };
}
