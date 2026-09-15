# IndexNow submissions and sitemap lastmod

`scripts/indexnow.mjs` notifies IndexNow about pages that changed in a deploy,
and fills in the sitemap's `lastmod` values from the same state. IndexNow reaches
Bing, Yandex, Seznam, Naver, Yep, Internet Archive and Amazonbot. Google does not
participate, so this speeds up Bing pickup — and with it DuckDuckGo, Copilot and
ChatGPT search — rather than affecting Google rankings.

The protocol wants only genuinely changed URLs; resubmitting unchanged ones
wastes crawl quota and risks a 429. The script therefore hashes every page in
`dist/` and compares those hashes against the previous deploy, so it never has to
map a source file back to a URL or read git history.

## How a deploy uses it

`.github/workflows/website.yaml` runs two commands on `main`, both after the AWS
credentials step so the manifest is reachable:

1. `node scripts/indexnow.mjs prepare`, before the S3 sync. It reads the URLs
   from `dist/sitemap-*.xml`, hashes each page, diffs against the manifest,
   writes `lastmod` into the sitemap files and records the changed URLs in
   `.indexnow/pending.json`.
2. `node scripts/indexnow.mjs submit`, after the sync and the CloudFront
   invalidation, so crawlers arriving immediately see the new content. It posts
   the recorded URLs to `https://api.indexnow.org/indexnow` and stores the new
   manifest.

Candidate URLs come from the built sitemap, which means the `noindex` exclusions
and the redirect map in `astro.config.mjs` are respected without repeating them
here.

## Manifest

`s3://distr.sh/.indexnow/manifest.json`, uploaded without `--acl` so it stays
private while the pages around it are `public-read`:

```json
{
  "version": 1,
  "urls": {
    "https://distr.sh/about/": {
      "hash": "9f86d081884c7d65...",
      "lastmod": "2026-09-02T10:14:07Z"
    }
  }
}
```

`lastmod` stays `null` until the script actually observes that page change. Bing
disregards `lastmod` values that all sit on the same date, which is exactly what
backfilling the first deploy would produce, and IndexNow asks for changes made
after adoption only. So the first run records hashes, submits nothing and emits
no dates; pages pick up a `lastmod` as they change.

Because the previous deploy's state lives in S3, `lastmod` is injected in CI
rather than by `@astrojs/sitemap`. A local `pnpm build` produces a sitemap
without it.

Three consequences worth knowing:

- Editing a shared component such as the footer really does change every page, so
  all of them are submitted. That is accurate and fits in a single request.
- A failed submission leaves the manifest untouched, so the next deploy retries
  the same URLs instead of losing them.
- The homepage can be reported as changed when it was not.
  `src/assets/index.ts` imports both
  `screenshots/distr/distr-customer-portal-artifacts.webp` and its byte-identical
  twin `…-artifacts-light.webp`, so Astro emits one file for the two and which of
  the two names wins shifts with build ordering. Dropping the duplicate would
  settle it, but that belongs to the screenshot pipeline rather than here.

## Running it locally

`--dry-run` keeps everything on disk: the manifest is read from and written to
`website/.indexnow/manifest.json`, and nothing is submitted.

```sh
pnpm build
node scripts/indexnow.mjs prepare --dry-run
```

Running it twice with a rebuild in between is the way to check that the hash
normalization still ignores Astro's fingerprinted asset names — the second run
has to report zero changed pages.

## One-time setup

1. Generate a key with `openssl rand -hex 16`. IndexNow allows `a-z`, `A-Z`,
   `0-9` and `-`, between 8 and 128 characters.
2. Add it as the `INDEXNOW_KEY` repository secret. The deploy writes it to
   `dist/<key>.txt`, so the key is served from `https://distr.sh/<key>.txt`
   without its name appearing in this public repository. Until the secret exists,
   the deploy still maintains the manifest and the sitemap `lastmod` values and
   only skips the submission.
3. Claim distr.sh in [Bing Webmaster Tools](https://www.bing.com/webmasters),
   which can import the site from Google Search Console. Neither IndexNow nor any
   Bing API exposes submission history, so its IndexNow report is the only place
   to confirm that submissions land.
4. Check that the `github-actions-distr.sh` role may `GetObject` and `PutObject`
   on `distr.sh/.indexnow/*`. A read failure is reported and treated as a first
   run, but a write failure fails the step.

Rotating the key leaves the old key file in the bucket, because neither sync uses
`--delete`. Remove it by hand.
