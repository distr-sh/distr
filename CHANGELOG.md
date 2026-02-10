# Changelog

## [2.10.0](https://github.com/distr-sh/distr/compare/2.9.0...2.10.0) (2026-02-10)


### Features

* add downloaded by customer organizations ([#1783](https://github.com/distr-sh/distr/issues/1783)) ([0a59eaa](https://github.com/distr-sh/distr/commit/0a59eaaebc3fc93896899f6dd5a1a143cd48c4cf))
* add superadmin user with extended read-only capabilities ([#1782](https://github.com/distr-sh/distr/issues/1782)) ([69ef346](https://github.com/distr-sh/distr/commit/69ef34600e2164a1a1d319dfb5d113930427031d))


### Bug Fixes

* add permissions check for notification config api ([#1793](https://github.com/distr-sh/distr/issues/1793)) ([f184f17](https://github.com/distr-sh/distr/commit/f184f170cd6cb96416aa6c21311bb57cb262a045))
* **backend:** do not report status "progressing" as recovered from error ([#1792](https://github.com/distr-sh/distr/issues/1792)) ([df618c7](https://github.com/distr-sh/distr/commit/df618c73eb8aa407f5a26c2b99495a86ec0136bf))
* **deps:** update module golang.org/x/crypto to v0.48.0 ([#1795](https://github.com/distr-sh/distr/issues/1795)) ([ca88fd9](https://github.com/distr-sh/distr/commit/ca88fd96fbb7264e8151e517a773d8113c237f2f))
* **deps:** update module golang.org/x/oauth2 to v0.35.0 ([#1787](https://github.com/distr-sh/distr/issues/1787)) ([4554faf](https://github.com/distr-sh/distr/commit/4554faf97b990c96a38cabc93164e752329e236d))
* **ui:** update user name without profile image ([#1796](https://github.com/distr-sh/distr/issues/1796)) ([2e849df](https://github.com/distr-sh/distr/commit/2e849df1c4150a50aa2f685a15c096fb8d33c62e))


### Other

* **deps:** update angular-cli monorepo to v21.1.3 ([#1780](https://github.com/distr-sh/distr/issues/1780)) ([a833ed9](https://github.com/distr-sh/distr/commit/a833ed90e73355d00fe0ae979021d992b3b482a0))
* **deps:** update codemirror ([#1784](https://github.com/distr-sh/distr/issues/1784)) ([26b7516](https://github.com/distr-sh/distr/commit/26b75169c60bcab9b549e05ac1eb3efbe166beef))
* **deps:** update dependency pnpm to v10.29.2 ([#1786](https://github.com/distr-sh/distr/issues/1786)) ([0303c02](https://github.com/distr-sh/distr/commit/0303c02e1d3ddbc46f1375e263ef4a095ce7a907))
* **deps:** update dependency posthog-js to v1.343.0 ([#1788](https://github.com/distr-sh/distr/issues/1788)) ([b19bc39](https://github.com/distr-sh/distr/commit/b19bc39c648810189aaff12a59c75644867c5880))
* **deps:** update dependency typedoc-plugin-markdown to v4.10.0 ([#1785](https://github.com/distr-sh/distr/issues/1785)) ([058ca5c](https://github.com/distr-sh/distr/commit/058ca5c93905aacf9199ff3c5d82df606e356382))
* **deps:** update sentry-javascript monorepo to v10.38.0 ([#1789](https://github.com/distr-sh/distr/issues/1789)) ([5f385f4](https://github.com/distr-sh/distr/commit/5f385f46c7aebc02d197705d62f2fe69ff98d8b7))
* do not report "context cancelled" ([#1791](https://github.com/distr-sh/distr/issues/1791)) ([02a5e0b](https://github.com/distr-sh/distr/commit/02a5e0bc835c007c04d12ce69f5f425cbbbff5ae))
* rename DeploymentStatusNotificationConfiguration to AlertConfiguration ([#1794](https://github.com/distr-sh/distr/issues/1794)) ([f2293fc](https://github.com/distr-sh/distr/commit/f2293fc8c4fc0610c6c355eec2dbd24385eccfc1))
* rework deployment target card ([#1775](https://github.com/distr-sh/distr/issues/1775)) ([b5f3368](https://github.com/distr-sh/distr/commit/b5f33681975381bf1ac9c9e2e7cc664aaaa2f3e6))


### Performance

* **backend:** add NotificationRecord index ([#1790](https://github.com/distr-sh/distr/issues/1790)) ([7ee9c9b](https://github.com/distr-sh/distr/commit/7ee9c9b38ca9f163fc6229f424243287762fed31))

## [2.9.0](https://github.com/distr-sh/distr/compare/2.8.2...2.9.0) (2026-02-05)


### Features

* deployment status notifications ([#1717](https://github.com/distr-sh/distr/issues/1717)) ([16444be](https://github.com/distr-sh/distr/commit/16444be8fe2cededf04321c83da45c7d4b8b6d90))


### Other

* **deps:** update dependency semver to v7.7.4 ([#1778](https://github.com/distr-sh/distr/issues/1778)) ([f5615fa](https://github.com/distr-sh/distr/commit/f5615fa1d6aa1267a139a26f53dc0b30a622d2ad))

## [2.8.2](https://github.com/distr-sh/distr/compare/2.8.1...2.8.2) (2026-02-05)


### Bug Fixes

* prevent template errors for mailsending ([#1774](https://github.com/distr-sh/distr/issues/1774)) ([a10e1a7](https://github.com/distr-sh/distr/commit/a10e1a782f948ba1ce8afca18e75bde24483d26a))

## [2.8.1](https://github.com/distr-sh/distr/compare/2.8.0...2.8.1) (2026-02-05)


### Bug Fixes

* **deps:** update module github.com/go-chi/chi/v5 to v5.2.5 ([#1773](https://github.com/distr-sh/distr/issues/1773)) ([eff9e4a](https://github.com/distr-sh/distr/commit/eff9e4a4d13486d96693f3d73fc43886e758e958))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.145.0 ([#1763](https://github.com/distr-sh/distr/issues/1763)) ([30ab955](https://github.com/distr-sh/distr/commit/30ab95500b9ab209e053b946e1a50e66daa94b3e))
* **deps:** update module go.opentelemetry.io/collector/confmap to v1.51.0 ([#1752](https://github.com/distr-sh/distr/issues/1752)) ([a4e57a4](https://github.com/distr-sh/distr/commit/a4e57a42cee4c446ef0b0c0887a88b893c57040a))
* **frontend:** correctly display vendor/customer state of an organization in the navbar ([#1767](https://github.com/distr-sh/distr/issues/1767)) ([a6707ef](https://github.com/distr-sh/distr/commit/a6707effd9be8e5f3b5778ef028901c354c65cae))


### Other

* **deps:** bump the npm_and_yarn group across 2 directories with 1 update ([#1764](https://github.com/distr-sh/distr/issues/1764)) ([81d3471](https://github.com/distr-sh/distr/commit/81d3471a02b7e0e91a1bd15879c7c3771afbfd90))
* **deps:** update anchore/sbom-action action to v0.22.2 ([#1768](https://github.com/distr-sh/distr/issues/1768)) ([57808e9](https://github.com/distr-sh/distr/commit/57808e92aa5a5f755ba85ed982fd7fd61c974637))
* **deps:** update angular monorepo to v21.1.3 ([#1771](https://github.com/distr-sh/distr/issues/1771)) ([8a72d34](https://github.com/distr-sh/distr/commit/8a72d34bb92719d92ae9c8e535c7f4c571b28cb8))
* **deps:** update dependency @angular/cdk to v21.1.3 ([#1770](https://github.com/distr-sh/distr/issues/1770)) ([572ea40](https://github.com/distr-sh/distr/commit/572ea40e3f66fb901ab08b4231dd2a6f7c9e3d33))
* **deps:** update dependency go to v1.25.7 ([#1769](https://github.com/distr-sh/distr/issues/1769)) ([e5ca5ca](https://github.com/distr-sh/distr/commit/e5ca5ca8925145c1840b13759dd73ed3a77c7f03))
* **deps:** update dependency ngx-markdown to v21.1.0 ([#1766](https://github.com/distr-sh/distr/issues/1766)) ([de8e152](https://github.com/distr-sh/distr/commit/de8e152b574d7a154c21e9c74eb6e3c7c88b7939))
* **deps:** update docker docker tag to v29.2.1 ([#1765](https://github.com/distr-sh/distr/issues/1765)) ([f79de3f](https://github.com/distr-sh/distr/commit/f79de3fd7f9272175ebbcd8b6292db1788445621))


### Refactoring

* merge user invitation email templates ([#1772](https://github.com/distr-sh/distr/issues/1772)) ([3d38174](https://github.com/distr-sh/distr/commit/3d381743126cf221b474824ef234ecace5b8fed1))

## [2.8.0](https://github.com/distr-sh/distr/compare/2.7.1...2.8.0) (2026-02-03)


### Features

* **registry:** implement OCI manifest deletion by tag ([#1742](https://github.com/distr-sh/distr/issues/1742)) ([94303ad](https://github.com/distr-sh/distr/commit/94303ad57d26e88597c7105711bb6fb2a5c644cd))
* totp multi-factor authentication ([#1734](https://github.com/distr-sh/distr/issues/1734)) ([e69179b](https://github.com/distr-sh/distr/commit/e69179bd25e2878cce1d3ec2c41cdb36a9bcdb32))


### Bug Fixes

* **backend:** set width on email logos ([#1758](https://github.com/distr-sh/distr/issues/1758)) ([d2d6c55](https://github.com/distr-sh/distr/commit/d2d6c55617ef21923784862f72e30de3183b764b))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.96.0 ([#1748](https://github.com/distr-sh/distr/issues/1748)) ([12ae198](https://github.com/distr-sh/distr/commit/12ae1988e23989ef61bd26b0ef484f89373fb843))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.42.0 ([#1723](https://github.com/distr-sh/distr/issues/1723)) ([da7a28d](https://github.com/distr-sh/distr/commit/da7a28dede784c31afb0bd4ac98875b79e902130))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.19.1 ([#1731](https://github.com/distr-sh/distr/issues/1731)) ([555c5ac](https://github.com/distr-sh/distr/commit/555c5ac6736bfca261c3649ae1b8a0844ab63612))
* **deps:** update module github.com/onsi/gomega to v1.39.1 ([#1745](https://github.com/distr-sh/distr/issues/1745)) ([ddda70f](https://github.com/distr-sh/distr/commit/ddda70f320be5bab271222c9dff2f5d649abcadf))
* **deps:** update module github.com/stripe/stripe-go/v84 to v84.3.0 ([#1749](https://github.com/distr-sh/distr/issues/1749)) ([00d20ec](https://github.com/distr-sh/distr/commit/00d20ece596dbef429a0ba74ec4496065595edae))
* **deps:** update module go.opentelemetry.io/collector/consumer to v1.51.0 ([#1753](https://github.com/distr-sh/distr/issues/1753)) ([28f5f45](https://github.com/distr-sh/distr/commit/28f5f4552fa6c6741712663d36ec36e7ae500c80))
* **deps:** update module go.opentelemetry.io/collector/receiver to v1.51.0 ([#1756](https://github.com/distr-sh/distr/issues/1756)) ([922acf0](https://github.com/distr-sh/distr/commit/922acf0ffdaae7d2e471b0d73fc5a722850587bc))
* **deps:** update opentelemetry-go monorepo to v1.40.0 ([#1755](https://github.com/distr-sh/distr/issues/1755)) ([c4632fe](https://github.com/distr-sh/distr/commit/c4632fe0d9a9a2dbdf20bc50a58e9ba9f23878f8))
* **deps:** update opentelemetry-go-contrib monorepo to v0.65.0 ([#1757](https://github.com/distr-sh/distr/issues/1757)) ([beb715f](https://github.com/distr-sh/distr/commit/beb715f20d5378ed519e6e7c40e36b55e77778f1))


### Other

* add escaping newline characters in env secrets ([#1760](https://github.com/distr-sh/distr/issues/1760)) ([e8c0696](https://github.com/distr-sh/distr/commit/e8c06963410f0c5024770ae9e44d39f9505b2f4e))
* **deps:** update anchore/sbom-action action to v0.22.1 ([#1727](https://github.com/distr-sh/distr/issues/1727)) ([947ce71](https://github.com/distr-sh/distr/commit/947ce71c4687807ddf8471fab75438c5f966a0e0))
* **deps:** update angular monorepo to v21.1.2 ([#1732](https://github.com/distr-sh/distr/issues/1732)) ([2d08949](https://github.com/distr-sh/distr/commit/2d08949063bc4b5ef2d1ec77b2690ad6a1fdc162))
* **deps:** update angular-cli monorepo to v21.1.2 ([#1729](https://github.com/distr-sh/distr/issues/1729)) ([8902c0e](https://github.com/distr-sh/distr/commit/8902c0e63a920721dbad0ccd8b1681d30b0684a9))
* **deps:** update axllent/mailpit docker tag to v1.29.0 ([#1746](https://github.com/distr-sh/distr/issues/1746)) ([46a7d40](https://github.com/distr-sh/distr/commit/46a7d40ab90ad9153de10d23eb6edadf49fc6946))
* **deps:** update dependency @angular/cdk to v21.1.2 ([#1736](https://github.com/distr-sh/distr/issues/1736)) ([2df34af](https://github.com/distr-sh/distr/commit/2df34af3d5da2779ea84f92710c062b4a200b0ac))
* **deps:** update dependency @codemirror/view to v6.39.12 ([#1740](https://github.com/distr-sh/distr/issues/1740)) ([bbd6393](https://github.com/distr-sh/distr/commit/bbd6393307acb616fffa705db24edae49b9cbcfd))
* **deps:** update dependency autoprefixer to v10.4.24 ([#1744](https://github.com/distr-sh/distr/issues/1744)) ([652e029](https://github.com/distr-sh/distr/commit/652e029750cb15b9a766e6c9136141b9914f5363))
* **deps:** update dependency jsdom to v28 ([#1750](https://github.com/distr-sh/distr/issues/1750)) ([5aa88ac](https://github.com/distr-sh/distr/commit/5aa88acd5d66f0ab17714cc7f6c7617a70e042ff))
* **deps:** update dependency stripe to v1.35.0 ([#1747](https://github.com/distr-sh/distr/issues/1747)) ([0aaa75d](https://github.com/distr-sh/distr/commit/0aaa75dc60babf740dc0a5d283ecd6db601f8328))
* **deps:** update distr-sh/hello-distr to v0.2.4 ([#1737](https://github.com/distr-sh/distr/issues/1737)) ([7cdfcda](https://github.com/distr-sh/distr/commit/7cdfcdad6713aa79c269894e8ec8569e3ebbbe0c))
* **deps:** update distr-sh/hello-distr to v0.4.1 ([#1759](https://github.com/distr-sh/distr/issues/1759)) ([5daf8ab](https://github.com/distr-sh/distr/commit/5daf8ab82cb307bdb1dcd0fd815d987cbea74d38))
* **deps:** update docker/login-action action to v3.7.0 ([#1730](https://github.com/distr-sh/distr/issues/1730)) ([cd6f146](https://github.com/distr-sh/distr/commit/cd6f1465f5b79d06fa6def97dd7e4b9d45717325))
* **frontend:** strip ANSI color escape sequences from logs ([#1761](https://github.com/distr-sh/distr/issues/1761)) ([b64191f](https://github.com/distr-sh/distr/commit/b64191f54a9a91a51723f7cb29c8fb59d779fbbe))

## [2.7.1](https://github.com/distr-sh/distr/compare/2.7.0...2.7.1) (2026-01-27)


### Bug Fixes

* **backend:** add backwards-compatible parsing of DeploymentStatusType ([#1725](https://github.com/distr-sh/distr/issues/1725)) ([c3c93f7](https://github.com/distr-sh/distr/commit/c3c93f701bf42eef525705bc629fcf6c589d5da9))

## [2.7.0](https://github.com/distr-sh/distr/compare/2.6.0...2.7.0) (2026-01-27)


### Features

* add customer organization features ([#1691](https://github.com/distr-sh/distr/issues/1691)) ([970ed09](https://github.com/distr-sh/distr/commit/970ed0928e7f9c6fb96b46508e73e6a91cc0cfc5))
* add customer organization to artifact version pull ([#1690](https://github.com/distr-sh/distr/issues/1690)) ([c57f069](https://github.com/distr-sh/distr/commit/c57f06930c36bf695de7d0145e7fc96c38a30c31))
* add health checks for compose deployments and distinct status types for "running" and "healthy" ([#1703](https://github.com/distr-sh/distr/issues/1703)) ([bcb7248](https://github.com/distr-sh/distr/commit/bcb7248d31de4a8f5eba320f7e35f926ef426f0f))
* collect agent logs ([#1692](https://github.com/distr-sh/distr/issues/1692)) ([d892639](https://github.com/distr-sh/distr/commit/d892639574cf967aff15aecff64f55cef54efaee))


### Bug Fixes

* **backend:** improve subscription update error handling ([#1714](https://github.com/distr-sh/distr/issues/1714)) ([05f66f9](https://github.com/distr-sh/distr/commit/05f66f930d88413c0c90bb4cf7666c15cacd1d77))
* **backend:** use correct hello-distr version name in agents tutorial ([#1715](https://github.com/distr-sh/distr/issues/1715)) ([d5eb6a9](https://github.com/distr-sh/distr/commit/d5eb6a9a08e6d836e25e490b2cb08be62d2cb21e))
* **deps:** update module helm.sh/helm/v3 to v3.20.0 ([#1699](https://github.com/distr-sh/distr/issues/1699)) ([5993e57](https://github.com/distr-sh/distr/commit/5993e579cbd4d1932dc51eabc8fba704182f5415))
* **ui:** prevent deployment menu from being cut off ([#1713](https://github.com/distr-sh/distr/issues/1713)) ([443ab06](https://github.com/distr-sh/distr/commit/443ab06e942f3f3e259267784dc8c028e54808fd))


### Other

* allow deleting non-empty customer but show a warning ([#1702](https://github.com/distr-sh/distr/issues/1702)) ([b19665e](https://github.com/distr-sh/distr/commit/b19665e41f56a5d4fa1ac905577cd725752f8f27))
* clarify subscription renews on wording ([#1706](https://github.com/distr-sh/distr/issues/1706)) ([5390432](https://github.com/distr-sh/distr/commit/539043249b8038cb368a88ceb1f80fb90203d238))
* **deps:** bump tar from 7.5.3 to 7.5.6 in the npm_and_yarn group across 1 directory ([#1693](https://github.com/distr-sh/distr/issues/1693)) ([c62f549](https://github.com/distr-sh/distr/commit/c62f549856bbfdab98e9411af7a6bf8f686e7fe2))
* **deps:** update actions/checkout action to v6.0.2 ([#1704](https://github.com/distr-sh/distr/issues/1704)) ([7ae34be](https://github.com/distr-sh/distr/commit/7ae34bee04561d6c7a26fb5b1998889c47fae6a7))
* **deps:** update anchore/sbom-action action to v0.22.0 ([#1698](https://github.com/distr-sh/distr/issues/1698)) ([ff63999](https://github.com/distr-sh/distr/commit/ff63999a1901be94dc5abc63ea78e90604c40fcc))
* **deps:** update angular monorepo to v21.1.1 ([#1696](https://github.com/distr-sh/distr/issues/1696)) ([4f00ee6](https://github.com/distr-sh/distr/commit/4f00ee627408990bfffa4661c67af397ef406da0))
* **deps:** update angular-cli monorepo to v21.1.1 ([#1694](https://github.com/distr-sh/distr/issues/1694)) ([461bf50](https://github.com/distr-sh/distr/commit/461bf502e05f4cc5a7eae7efb1671ff1ed87a68b))
* **deps:** update axllent/mailpit docker tag to v1.28.4 ([#1708](https://github.com/distr-sh/distr/issues/1708)) ([167ffe9](https://github.com/distr-sh/distr/commit/167ffe9170102edb9ee3f1a1b58444e6b899388b))
* **deps:** update dependency @angular/cdk to v21.1.1 ([#1695](https://github.com/distr-sh/distr/issues/1695)) ([24b0668](https://github.com/distr-sh/distr/commit/24b0668f7e6323f029f152e97bd4351885afde2d))
* **deps:** update dependency @types/jasmine to v6 ([#1701](https://github.com/distr-sh/distr/issues/1701)) ([5037a14](https://github.com/distr-sh/distr/commit/5037a148e2bab9dba94c6adfbb4d97ed79cd7776))
* **deps:** update dependency pnpm to v10.28.2 ([#1718](https://github.com/distr-sh/distr/issues/1718)) ([e4c2359](https://github.com/distr-sh/distr/commit/e4c23596467bd74c503f227890fa284d5fdf2b6f))
* **deps:** update dependency posthog-js to v1.335.2 ([#1709](https://github.com/distr-sh/distr/issues/1709)) ([4cbdf61](https://github.com/distr-sh/distr/commit/4cbdf61a58e3258926cb0ec20d4c210b3217d661))
* **deps:** update dependency prettier to v3.8.1 ([#1697](https://github.com/distr-sh/distr/issues/1697)) ([6e5efe3](https://github.com/distr-sh/distr/commit/6e5efe3fe524d1356486f2cf90c02b1d20f67c88))
* **deps:** update dependency vitest to v4.0.18 ([#1705](https://github.com/distr-sh/distr/issues/1705)) ([6f20c87](https://github.com/distr-sh/distr/commit/6f20c87ac08bf2a81e261ebe9db2161262cce7ce))
* **deps:** update distr-sh/hello-distr to v0.2.3 ([#1719](https://github.com/distr-sh/distr/issues/1719)) ([0fe68c4](https://github.com/distr-sh/distr/commit/0fe68c43ffe35301214d3a2af60895d468f11231))
* **deps:** update docker docker tag to v29.2.0 ([#1722](https://github.com/distr-sh/distr/issues/1722)) ([f1ccd8f](https://github.com/distr-sh/distr/commit/f1ccd8ff048eefe6d28d74d704a1a997d3e7da19))
* **deps:** update sentry-javascript monorepo to v10.36.0 ([#1710](https://github.com/distr-sh/distr/issues/1710)) ([34ef8a4](https://github.com/distr-sh/distr/commit/34ef8a423fd38639720d1f186ff67c0623dce457))
* **frontend:** sort artifacts by last push ([#1716](https://github.com/distr-sh/distr/issues/1716)) ([b1bbac8](https://github.com/distr-sh/distr/commit/b1bbac882fe2d700af476d5a32d7a07466d02b22))
* **kubernetes-agent:** use parent resource name for collected logs ([#1712](https://github.com/distr-sh/distr/issues/1712)) ([8e99c99](https://github.com/distr-sh/distr/commit/8e99c99631b667a05ca33ae9d91bc10e9752b353))

## [2.6.0](https://github.com/distr-sh/distr/compare/2.5.2...2.6.0) (2026-01-20)


### Features

* add deployment target notes ([#1664](https://github.com/distr-sh/distr/issues/1664)) ([226d0c1](https://github.com/distr-sh/distr/commit/226d0c103523819309fd1d8767f67f5c49a737d3))
* add edit profile page ([#1682](https://github.com/distr-sh/distr/issues/1682)) ([3cf1c26](https://github.com/distr-sh/distr/commit/3cf1c26dbd0b00d43c3dc83efe7bd8d92cb6604c))
* **backend:** remember last logged in organization ([#1665](https://github.com/distr-sh/distr/issues/1665)) ([8ea14ea](https://github.com/distr-sh/distr/commit/8ea14ea2889d8b8301b2ad54eef027b1a429092a))
* implement status & ready endpoint ([#1673](https://github.com/distr-sh/distr/issues/1673)) ([d73703b](https://github.com/distr-sh/distr/commit/d73703b1fae594aa80c923923439154109c3a2b8))


### Bug Fixes

* **deps:** update module github.com/getsentry/sentry-go/otel to v0.41.0 ([#1660](https://github.com/distr-sh/distr/issues/1660)) ([7d9e462](https://github.com/distr-sh/distr/commit/7d9e462fca612b93100759114ce6c8cdabe086d6))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.144.0 ([#1689](https://github.com/distr-sh/distr/issues/1689)) ([82f5b12](https://github.com/distr-sh/distr/commit/82f5b121ce1b1beefea24d66c2319fb1de6eef49))
* **deps:** update module github.com/stripe/stripe-go/v84 to v84.2.0 ([#1667](https://github.com/distr-sh/distr/issues/1667)) ([52a41f5](https://github.com/distr-sh/distr/commit/52a41f5aad35acb3d1b6af1f1ea11028095eb7c3))
* **ui:** disable @angular/cdk overlay using popover API ([#1675](https://github.com/distr-sh/distr/issues/1675)) ([99e3c49](https://github.com/distr-sh/distr/commit/99e3c492d9570b475357e02dae9de386b0baf6d4))
* **ui:** prevent text overflow in toast component ([#1654](https://github.com/distr-sh/distr/issues/1654)) ([fc02207](https://github.com/distr-sh/distr/commit/fc02207942e79488d5fd91d7ebcf92b301210cbe))


### Other

* **deps:** bump tar from 7.5.2 to 7.5.3 in the npm_and_yarn group across 1 directory ([#1668](https://github.com/distr-sh/distr/issues/1668)) ([337d955](https://github.com/distr-sh/distr/commit/337d955e7dea84ca42ed9c36eafe1e682ce89c56))
* **deps:** update axllent/mailpit docker tag to v1.28.3 ([#1670](https://github.com/distr-sh/distr/issues/1670)) ([da764e0](https://github.com/distr-sh/distr/commit/da764e0bab2bd247c40f817c64ebdbcee5610f12))
* **deps:** update dependency @sentry/cli to v3.1.0 ([#1671](https://github.com/distr-sh/distr/issues/1671)) ([12ddbdb](https://github.com/distr-sh/distr/commit/12ddbdbbd8eef8648c0e92280ee2feba9959f11d))
* **deps:** update dependency go to v1.25.6 ([#1657](https://github.com/distr-sh/distr/issues/1657)) ([97da58d](https://github.com/distr-sh/distr/commit/97da58d4b208146685356e22f17c07c89926bafb))
* **deps:** update dependency jasmine-core to v6 ([#1669](https://github.com/distr-sh/distr/issues/1669)) ([ad0eb0f](https://github.com/distr-sh/distr/commit/ad0eb0fae9c6cb2c69bf5d1822ac06c8dd59a673))
* **deps:** update dependency jasmine-core to v6.0.1 ([#1680](https://github.com/distr-sh/distr/issues/1680)) ([4e2e79c](https://github.com/distr-sh/distr/commit/4e2e79ca9d3c7bea9fcef946b2b5b97f8f29ec6a))
* **deps:** update dependency pnpm to v10.28.1 ([#1677](https://github.com/distr-sh/distr/issues/1677)) ([2982714](https://github.com/distr-sh/distr/commit/29827144de70e2d992299fd4858faa2e5ba391f1))
* **deps:** update dependency posthog-js to v1.328.0 ([#1672](https://github.com/distr-sh/distr/issues/1672)) ([4f157a2](https://github.com/distr-sh/distr/commit/4f157a2bd111c9a6b3b765a8f39652004f4dbf24))
* **deps:** update distr-sh/hello-distr to v0.2.2 ([#1666](https://github.com/distr-sh/distr/issues/1666)) ([483caba](https://github.com/distr-sh/distr/commit/483caba59522c17759d7f03f84dc10f413dcccd9))
* **frontend:** remove update banner, replace with notification bubble per deployment ([#1674](https://github.com/distr-sh/distr/issues/1674)) ([0ba9dda](https://github.com/distr-sh/distr/commit/0ba9ddac90b77270314c463e3c829e20c289d614))
* remove DeploymentTarget created_by ([#1676](https://github.com/distr-sh/distr/issues/1676)) ([120812b](https://github.com/distr-sh/distr/commit/120812b876f66b45157b39c46928160a5bd12ce5))
* **ui:** use generic icon as default user avatar ([#1679](https://github.com/distr-sh/distr/issues/1679)) ([a31ec23](https://github.com/distr-sh/distr/commit/a31ec2313dbfa236d58f5819fc9f196660eae817))

## [2.5.2](https://github.com/distr-sh/distr/compare/2.5.1...2.5.2) (2026-01-15)


### Other

* change module name to github.com/distr-sh/distr ([#1651](https://github.com/distr-sh/distr/issues/1651)) ([4645d9a](https://github.com/distr-sh/distr/commit/4645d9aee5e451dee3a94fd4915ff0f25f18bb9f))
* rename ghcr.io/glasskube to ghcr.io/distr-sh ([#1653](https://github.com/distr-sh/distr/issues/1653)) ([b4fac20](https://github.com/distr-sh/distr/commit/b4fac204b49aa293606c155bb4499f97be9c7a18))

## [2.5.1](https://github.com/glasskube/distr/compare/2.5.0...2.5.1) (2026-01-15)


### Other

* **build:** fix build ([#1649](https://github.com/glasskube/distr/issues/1649)) ([b8e8362](https://github.com/glasskube/distr/commit/b8e8362ada14e161d9b7a0ce2000d384a9254bc7))

## [2.5.0](https://github.com/glasskube/distr/compare/2.4.0...2.5.0) (2026-01-15)


### Features

* add configuring resource requirements for Kubernetes agent deployment ([#1640](https://github.com/glasskube/distr/issues/1640)) ([017ca77](https://github.com/glasskube/distr/commit/017ca773d958fb13ded17a91887c2fbfee5a7448))
* add option to allow mutating artifact versions ([#1630](https://github.com/glasskube/distr/issues/1630)) ([79c786e](https://github.com/glasskube/distr/commit/79c786ebf2a0f1f21275b830a6a04c9d986ae4e0))
* add option to force the kubernetes agent to overwrite a newer helm release ([#1633](https://github.com/glasskube/distr/issues/1633)) ([012c11c](https://github.com/glasskube/distr/commit/012c11c585c3f588f3fc18e4dcb4bcf4f1da25cc))


### Bug Fixes

* **backend:** send 400 response to agent pushing logs when deployment doesn't exist ([#1637](https://github.com/glasskube/distr/issues/1637)) ([b09112b](https://github.com/glasskube/distr/commit/b09112b31a10fcb792dec12a937f3bbe9bf28970))
* **deps:** update module github.com/exaring/otelpgx to v0.10.0 ([#1623](https://github.com/glasskube/distr/issues/1623)) ([8dec87a](https://github.com/glasskube/distr/commit/8dec87a2bf05e8fe9a0dc77a5a7427d2eab150fd))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.4 ([#1642](https://github.com/glasskube/distr/issues/1642)) ([8cea56e](https://github.com/glasskube/distr/commit/8cea56e0c54eda66b903fc5fb851b1040cc5798d))
* **deps:** update module golang.org/x/crypto to v0.47.0 ([#1624](https://github.com/glasskube/distr/issues/1624)) ([05b35e1](https://github.com/glasskube/distr/commit/05b35e15bcd818afcb0c928a3f0477b80093a1bf))
* **deps:** update module helm.sh/helm/v3 to v3.19.5 ([#1643](https://github.com/glasskube/distr/issues/1643)) ([b8b9580](https://github.com/glasskube/distr/commit/b8b95805c4e1bc87e528d35fd80ff085e41ba501))
* handle delayed initial license use case  ([#1638](https://github.com/glasskube/distr/issues/1638)) ([7bfe05d](https://github.com/glasskube/distr/commit/7bfe05d47336c730c2cce4f25758db58123408b6))
* **ui:** fix no applications shown in deployment form ([#1636](https://github.com/glasskube/distr/issues/1636)) ([cd50e83](https://github.com/glasskube/distr/commit/cd50e839e511246a18b1d60b0235ed16ab540795))
* **ui:** prevent text overflow in deployment wizard ([#1628](https://github.com/glasskube/distr/issues/1628)) ([5f14b90](https://github.com/glasskube/distr/commit/5f14b9090902220c76deb7eba722b8ab11b75292))


### Other

* add dlv to mise for debugging ([#1639](https://github.com/glasskube/distr/issues/1639)) ([9a5bb22](https://github.com/glasskube/distr/commit/9a5bb225fb43edcc36d2d0f93559a087cea33b59))
* **deps:** bump qs from 6.13.0 to 6.14.1 in the npm_and_yarn group across 1 directory ([#1648](https://github.com/glasskube/distr/issues/1648)) ([b6f0724](https://github.com/glasskube/distr/commit/b6f0724f9d7614c2035162f91834a3fdc5523d89))
* **deps:** update actions/setup-go action to v6.2.0 ([#1627](https://github.com/glasskube/distr/issues/1627)) ([f7395e6](https://github.com/glasskube/distr/commit/f7395e6bffad7997d59a948c4a8e9ab389cacd69))
* **deps:** update actions/setup-node action to v6.2.0 ([#1645](https://github.com/glasskube/distr/issues/1645)) ([40c1259](https://github.com/glasskube/distr/commit/40c125987944b17ab9d1dbb031f53be59062dd5d))
* **deps:** update codemirror ([#1635](https://github.com/glasskube/distr/issues/1635)) ([b94898c](https://github.com/glasskube/distr/commit/b94898cf38cb252245abf98fa818a772ac56090d))
* **deps:** update dependency @codemirror/view to v6.39.10 ([#1629](https://github.com/glasskube/distr/issues/1629)) ([47bafc9](https://github.com/glasskube/distr/commit/47bafc9608abab14e46a1f4353ef8445163334a4))
* **deps:** update dependency @types/jasmine to v5.1.15 ([#1625](https://github.com/glasskube/distr/issues/1625)) ([b8d08fc](https://github.com/glasskube/distr/commit/b8d08fcf71d5e8e6d62ff76a4f80767f1f41860b))
* **deps:** update dependency prettier to v3.8.0 ([#1644](https://github.com/glasskube/distr/issues/1644)) ([0d82fcf](https://github.com/glasskube/distr/commit/0d82fcfd8ef80f488f7d172c5d1f7e4914cc4952))
* **deps:** update ghcr.io/glasskube/hello-distr/backend docker tag to v0.2.1 ([#1631](https://github.com/glasskube/distr/issues/1631)) ([56b8870](https://github.com/glasskube/distr/commit/56b8870b6ace508d594781814b49babde98ad518))
* **deps:** update ghcr.io/glasskube/hello-distr/frontend docker tag to v0.2.1 ([#1632](https://github.com/glasskube/distr/issues/1632)) ([391d85b](https://github.com/glasskube/distr/commit/391d85b0e6f7a65a9d24a631265a3f44de8bd64f))
* **deps:** update ghcr.io/glasskube/hello-distr/proxy docker tag to v0.2.1 ([#1634](https://github.com/glasskube/distr/issues/1634)) ([29d5144](https://github.com/glasskube/distr/commit/29d5144d4c640d9299a2a25e7b20bf371dbc44f7))
* **deps:** upgrade to Angular 21 ([#1647](https://github.com/glasskube/distr/issues/1647)) ([8374e08](https://github.com/glasskube/distr/commit/8374e08e6e275fef84916c8128d1650dc99f5e70))
* show contact founder dialog to community users ([#1646](https://github.com/glasskube/distr/issues/1646)) ([cd666ca](https://github.com/glasskube/distr/commit/cd666ca4d2c7d4b8fde03f770e3205d650f0b433))
* **ui:** hide tutorial navlink if org has subscription ([#1641](https://github.com/glasskube/distr/issues/1641)) ([ed12315](https://github.com/glasskube/distr/commit/ed123154a2fd5ed62750d073fe3572e4d0aab77f))

## [2.4.0](https://github.com/glasskube/distr/compare/2.3.0...2.4.0) (2026-01-12)


### Features

* improve deployment flow ([#1579](https://github.com/glasskube/distr/issues/1579)) ([251e6c6](https://github.com/glasskube/distr/commit/251e6c68a9e8d6310d8acb008a239319c98f3817))
* secret management ([#1572](https://github.com/glasskube/distr/issues/1572)) ([4e88059](https://github.com/glasskube/distr/commit/4e88059e13f4394c6af3a6b809235417ae787a48))


### Bug Fixes

* **deps:** update aws-sdk-go-v2 monorepo ([#1614](https://github.com/glasskube/distr/issues/1614)) ([a5a7832](https://github.com/glasskube/distr/commit/a5a78325031fa5f8edb3e6b2bef8b12cf9b7bd90))


### Other

* **deps:** update axllent/mailpit docker tag to v1.28.2 ([#1617](https://github.com/glasskube/distr/issues/1617)) ([de7f1cd](https://github.com/glasskube/distr/commit/de7f1cd149ba9e15a9947bcc33a4570c123276c0))
* **deps:** update dependency @sentry/cli to v3 ([#1622](https://github.com/glasskube/distr/issues/1622)) ([cbea931](https://github.com/glasskube/distr/commit/cbea93140df20c9679b4a209d82eac84ba9a021b))
* **deps:** update dependency @types/jasmine to v5.1.14 ([#1618](https://github.com/glasskube/distr/issues/1618)) ([b4bc1ca](https://github.com/glasskube/distr/commit/b4bc1cab4b5474629c06d745457af882a825da84))
* **deps:** update dependency pnpm to v10.28.0 ([#1616](https://github.com/glasskube/distr/issues/1616)) ([d489783](https://github.com/glasskube/distr/commit/d489783400cc53e0db44cb95a9142b3c4dc2e5e8))
* **deps:** update dependency posthog-js to v1.318.1 ([#1621](https://github.com/glasskube/distr/issues/1621)) ([b0ca9b0](https://github.com/glasskube/distr/commit/b0ca9b038d4ed5ef96b93e286955f065b1111826))
* **deps:** update dependency typedoc to v0.28.16 ([#1620](https://github.com/glasskube/distr/issues/1620)) ([31375c0](https://github.com/glasskube/distr/commit/31375c04ade2807cf787e61a60f3859eeea6d2c6))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to cba10d7 ([#1619](https://github.com/glasskube/distr/issues/1619)) ([7dffde2](https://github.com/glasskube/distr/commit/7dffde2739d58c9f1658a9c83364d14eaefff0e3))

## [2.3.0](https://github.com/glasskube/distr/compare/2.2.5...2.3.0) (2026-01-09)


### Features

* add organization deletion ([#1603](https://github.com/glasskube/distr/issues/1603)) ([342ef6f](https://github.com/glasskube/distr/commit/342ef6f5ebf3c5c540ae6ab2d5c58ccefb57be51))
* application link ([#1573](https://github.com/glasskube/distr/issues/1573)) ([e4cc953](https://github.com/glasskube/distr/commit/e4cc953d4cb4cfdd1801446049a4996db6d18796))


### Bug Fixes

* **deps:** update module github.com/onsi/gomega to v1.39.0 ([#1610](https://github.com/glasskube/distr/issues/1610)) ([69383c5](https://github.com/glasskube/distr/commit/69383c51760845aa3221de1009ca96627ac19507))


### Other

* enable promo codes for stripe checkout sessions ([#1613](https://github.com/glasskube/distr/issues/1613)) ([2252aa0](https://github.com/glasskube/distr/commit/2252aa0a78cb05e98157ae8ca98739a688674bc8))

## [2.2.5](https://github.com/glasskube/distr/compare/2.2.4...2.2.5) (2026-01-08)


### Other

* **deps:** update @lezer/common to 1.5.0 ([#1608](https://github.com/glasskube/distr/issues/1608)) ([9ddad9b](https://github.com/glasskube/distr/commit/9ddad9b872176209b73abc771def1d78f9c2cf98))

## [2.2.4](https://github.com/glasskube/distr/compare/2.2.3...2.2.4) (2026-01-08)


### Bug Fixes

* **deps:** update kubernetes packages to v0.35.0 ([#1578](https://github.com/glasskube/distr/issues/1578)) ([dfcbd64](https://github.com/glasskube/distr/commit/dfcbd6476070eaedaaeff9f3a492f52b33f6c2fa))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.95.0 ([#1594](https://github.com/glasskube/distr/issues/1594)) ([d8ce8e6](https://github.com/glasskube/distr/commit/d8ce8e6d694088861f0aaf6d74ec8e8774838d16))
* **deps:** update module github.com/jackc/pgx/v5 to v5.8.0 ([#1595](https://github.com/glasskube/distr/issues/1595)) ([fed6c89](https://github.com/glasskube/distr/commit/fed6c8975c26ff1bb569d39ff2450c5e4f0b4498))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.143.0 ([#1597](https://github.com/glasskube/distr/issues/1597)) ([68f036f](https://github.com/glasskube/distr/commit/68f036f383f974eaf5540290fc6b12287074564f))


### Other

* add option to run connect script as root ([#1596](https://github.com/glasskube/distr/issues/1596)) ([0b4f99d](https://github.com/glasskube/distr/commit/0b4f99d4d450a2073f7e5d6123aafebfb66fa0f5))
* **deps:** update anchore/sbom-action action to v0.21.1 ([#1607](https://github.com/glasskube/distr/issues/1607)) ([501dd1a](https://github.com/glasskube/distr/commit/501dd1ae2df70837e637f7bcdccd9c803828eca8))
* **deps:** update angular monorepo to v20.3.16 ([#1604](https://github.com/glasskube/distr/issues/1604)) ([c5d772c](https://github.com/glasskube/distr/commit/c5d772c5f56a5962ab225272527f421ab9082cbb))
* **deps:** update angular-cli monorepo to v20.3.14 ([#1605](https://github.com/glasskube/distr/issues/1605)) ([613b5db](https://github.com/glasskube/distr/commit/613b5db411ff8fdd7e0e252a493ab51535dda6ca))
* **deps:** update dependency golangci-lint to v2.8.0 ([#1606](https://github.com/glasskube/distr/issues/1606)) ([1bd389a](https://github.com/glasskube/distr/commit/1bd389aaa1ddb40eb3632252418f216a215eaebe))
* **frontend:** add read-only subscription view for non-admin users ([#1599](https://github.com/glasskube/distr/issues/1599)) ([dbe1501](https://github.com/glasskube/distr/commit/dbe15013acea7d6428dd29467f4da5c2b8d46266))

## [2.2.3](https://github.com/glasskube/distr/compare/2.2.2...2.2.3) (2026-01-07)


### Bug Fixes

* **backend:** fix PAT cannot be deleted ([#1593](https://github.com/glasskube/distr/issues/1593)) ([5c66a45](https://github.com/glasskube/distr/commit/5c66a451b6ed7eae1d8a21e5a2aa9bd92c116be8))
* **deps:** update module github.com/stripe/stripe-go/v84 to v84.1.0 ([#1575](https://github.com/glasskube/distr/issues/1575)) ([fca41cc](https://github.com/glasskube/distr/commit/fca41cc68664b6117910bee83c015ccc496cbe5f))


### Other

* **deps:** update anchore/sbom-action action to v0.21.0 ([#1592](https://github.com/glasskube/distr/issues/1592)) ([773f3c4](https://github.com/glasskube/distr/commit/773f3c47ca6e8de85737092abf311ee5bd5a9ea7))
* **deps:** update axllent/mailpit docker tag to v1.28.1 ([#1591](https://github.com/glasskube/distr/issues/1591)) ([2011222](https://github.com/glasskube/distr/commit/20112222dc0e3e6f96374855518e702183b1464b))
* **deps:** update codemirror ([#1587](https://github.com/glasskube/distr/issues/1587)) ([028f8be](https://github.com/glasskube/distr/commit/028f8be70f51bc6011835bb6b300060ebd88205d))
* **deps:** update dependency pnpm to v10.27.0 ([#1581](https://github.com/glasskube/distr/issues/1581)) ([378dccf](https://github.com/glasskube/distr/commit/378dccf2dcae6707f2fd28dba75ac85e19437f52))
* **deps:** update dependency posthog-js to v1.315.0 ([#1585](https://github.com/glasskube/distr/issues/1585)) ([707fe56](https://github.com/glasskube/distr/commit/707fe56c150afe00bbcdaeb59bf194e2dc1b0f6d))
* **deps:** update dependency stripe to v1.34.0 ([#1577](https://github.com/glasskube/distr/issues/1577)) ([8b0fba1](https://github.com/glasskube/distr/commit/8b0fba1de637419395c0ea01b01d753e36ab968c))
* **deps:** update sentry-javascript monorepo to v10.32.1 ([#1586](https://github.com/glasskube/distr/issues/1586)) ([d392ee8](https://github.com/glasskube/distr/commit/d392ee8ad9e5255680527495d6cd2aafaa8a57a0))

## [2.2.2](https://github.com/glasskube/distr/compare/2.2.1...2.2.2) (2025-12-30)


### Other

* **backend:** improve Stripe webhook error handling ([#1582](https://github.com/glasskube/distr/issues/1582)) ([07f9d6d](https://github.com/glasskube/distr/commit/07f9d6d58b9c34f8c5bd810d17ab75963de2d47d))
* **deps:** update docker/setup-buildx-action action to v3.12.0 ([#1583](https://github.com/glasskube/distr/issues/1583)) ([49dc6c1](https://github.com/glasskube/distr/commit/49dc6c1b745d20b6ca05fd166629aec22da4d079))
* increase deployments per customer for pro orgs ([#1589](https://github.com/glasskube/distr/issues/1589)) ([a8309f7](https://github.com/glasskube/distr/commit/a8309f7d4129d799445788333706f5599f57a708))

## [2.2.1](https://github.com/glasskube/distr/compare/2.2.0...2.2.1) (2025-12-18)


### Bug Fixes

* **deps:** update aws-sdk-go-v2 monorepo ([#1574](https://github.com/glasskube/distr/issues/1574)) ([46678b7](https://github.com/glasskube/distr/commit/46678b72d9b82ea13979bc691c9c60fc0151fd5b))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.94.0 ([#1565](https://github.com/glasskube/distr/issues/1565)) ([db9814f](https://github.com/glasskube/distr/commit/db9814f75c07ff63e27f59709052170df3561756))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.142.0 ([#1567](https://github.com/glasskube/distr/issues/1567)) ([b3d4078](https://github.com/glasskube/distr/commit/b3d4078cff37ae98a91494dcf856914b934070fb))
* **kubernetes-agent:** increase memory limit to 256Mi and add memory request ([#1580](https://github.com/glasskube/distr/issues/1580)) ([fde858e](https://github.com/glasskube/distr/commit/fde858ed1b81d4ce6f7a52b21af52e79bc71ccec))


### Other

* **deps:** update dependency @codemirror/commands to v6.10.1 ([#1576](https://github.com/glasskube/distr/issues/1576)) ([9e8fafa](https://github.com/glasskube/distr/commit/9e8fafacf1fdf16a9530747e22050322e3435767))
* **deps:** update dependency stripe to v1.33.2 ([#1564](https://github.com/glasskube/distr/issues/1564)) ([865b6d0](https://github.com/glasskube/distr/commit/865b6d0a426c095ecc9e55a5e235496b40286ed6))

## [2.2.0](https://github.com/glasskube/distr/compare/2.1.0...2.2.0) (2025-12-15)


### Features

* add OpenAPI specification and documentation ([#1563](https://github.com/glasskube/distr/issues/1563)) ([6c367cf](https://github.com/glasskube/distr/commit/6c367cfbc2a2d9e56400b6df0c08f30b58f4bb31))


### Bug Fixes

* also append secret name to pullSecrets in addition to imagePullSecrets ([#1561](https://github.com/glasskube/distr/issues/1561)) ([976f8ca](https://github.com/glasskube/distr/commit/976f8caabda0541a7310f886aea6a6d8bdf89407))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.19.0 ([#1551](https://github.com/glasskube/distr/issues/1551)) ([bfd10ca](https://github.com/glasskube/distr/commit/bfd10ca4ff9e9be7096a41006f270ac3c45780fb))
* **deps:** update module helm.sh/helm/v3 to v3.19.4 ([#1553](https://github.com/glasskube/distr/issues/1553)) ([39146e8](https://github.com/glasskube/distr/commit/39146e8221593ef78ddd2130927442fa7d1e8f06))


### Other

* **deps:** update dependency @codemirror/view to v6.39.4 ([#1549](https://github.com/glasskube/distr/issues/1549)) ([5073c1a](https://github.com/glasskube/distr/commit/5073c1a024fe65ff18b00ca9f5d75d68c4bfefab))
* **deps:** update dependency @sentry/cli to v2.58.4 ([#1555](https://github.com/glasskube/distr/issues/1555)) ([0851051](https://github.com/glasskube/distr/commit/0851051e1d203f5cac6834e6bf170d4480c6d822))
* **deps:** update dependency autoprefixer to v10.4.23 ([#1554](https://github.com/glasskube/distr/issues/1554)) ([9d5c950](https://github.com/glasskube/distr/commit/9d5c950634c443f1a2f6cefa945ff7837b1d6d48))
* **deps:** update dependency pnpm to v10.26.0 ([#1562](https://github.com/glasskube/distr/issues/1562)) ([9694437](https://github.com/glasskube/distr/commit/96944373515d9a7dd500db16a199c15101b313c5))
* **deps:** update dependency posthog-js to v1.306.1 ([#1556](https://github.com/glasskube/distr/issues/1556)) ([766ef89](https://github.com/glasskube/distr/commit/766ef89da850e18d5ac634a1ea3ddc3e41a92523))
* **deps:** update ghcr.io/glasskube/hello-distr/backend docker tag to v0.1.12 ([#1558](https://github.com/glasskube/distr/issues/1558)) ([9c25c3a](https://github.com/glasskube/distr/commit/9c25c3aa67885f7aeccacc38279921c7ae94ed14))
* **deps:** update ghcr.io/glasskube/hello-distr/proxy docker tag to v0.1.12 ([#1559](https://github.com/glasskube/distr/issues/1559)) ([81fd44c](https://github.com/glasskube/distr/commit/81fd44cb67f8a494ecdc2960b1d3270e83706a13))
* **deps:** update github artifact actions (major) ([#1552](https://github.com/glasskube/distr/issues/1552)) ([897e64a](https://github.com/glasskube/distr/commit/897e64abe2cda4ff0ebd20959b19813f122af530))
* remove deprecated env ([#1560](https://github.com/glasskube/distr/issues/1560)) ([9a49a06](https://github.com/glasskube/distr/commit/9a49a060cde4d4112d73acf52471ba9f47b5b642))
* **sdk/js:** remove deprecated deployment item in deployment target ([#1557](https://github.com/glasskube/distr/issues/1557)) ([0c3ca3c](https://github.com/glasskube/distr/commit/0c3ca3caeb6607e941ed8dfaa43681b01b1b9076))

## [2.1.0](https://github.com/glasskube/distr/compare/2.0.2...2.1.0) (2025-12-12)


### Features

* add custom pre/post connect scripts for docker agents ([#1543](https://github.com/glasskube/distr/issues/1543)) ([41d66ef](https://github.com/glasskube/distr/commit/41d66ef7391b7cfb796c25df5562ff2c68f99d0b))
* add deployment status and resource log export ([#1541](https://github.com/glasskube/distr/issues/1541)) ([c0c1d60](https://github.com/glasskube/distr/commit/c0c1d6038e58cb74bbcf29d40341a5cea21fbf5f))


### Bug Fixes

* **backend:** fix registration error ([#1542](https://github.com/glasskube/distr/issues/1542)) ([11e6524](https://github.com/glasskube/distr/commit/11e65245e1fd6df787aaf55b3284c28164d7a7ab))
* **deps:** update aws-sdk-go-v2 monorepo ([#1537](https://github.com/glasskube/distr/issues/1537)) ([3a20ff1](https://github.com/glasskube/distr/commit/3a20ff149411d33806b6a2e3b1c6d32f711528d6))
* **deps:** update kubernetes packages to v0.34.3 ([#1539](https://github.com/glasskube/distr/issues/1539)) ([16b3fa8](https://github.com/glasskube/distr/commit/16b3fa87012409f32c1a70ff40aaaaad41ccab80))
* **deps:** update module github.com/exaring/otelpgx to v0.9.4 ([#1540](https://github.com/glasskube/distr/issues/1540)) ([09ec3f7](https://github.com/glasskube/distr/commit/09ec3f7d0e8524a009cbffed619feacb53197271))
* **deps:** update module helm.sh/helm/v3 to v3.19.3 ([#1544](https://github.com/glasskube/distr/issues/1544)) ([f9b1100](https://github.com/glasskube/distr/commit/f9b110031edd7fedd1ea949b8e2b70e83f5c415b))
* **registry:** allow storing in-transit blobs on disk to save memory ([#1478](https://github.com/glasskube/distr/issues/1478)) ([caf6d92](https://github.com/glasskube/distr/commit/caf6d9270a110047b79cb6c0036ce84510ddbc6a))


### Other

* **deps:** update anchore/sbom-action action to v0.20.11 ([#1536](https://github.com/glasskube/distr/issues/1536)) ([ca43232](https://github.com/glasskube/distr/commit/ca43232c9c44c33b4712cc9a6a4b60ee24d1333e))
* **deps:** update dependency @codemirror/view to v6.39.3 ([#1545](https://github.com/glasskube/distr/issues/1545)) ([7ea57b0](https://github.com/glasskube/distr/commit/7ea57b0c612d0af8e480cbe23b63f366f82f4655))
* **deps:** update tailwindcss monorepo to v4.1.18 ([#1547](https://github.com/glasskube/distr/issues/1547)) ([60cc0f3](https://github.com/glasskube/distr/commit/60cc0f34b5a5caba0e8bc39cb8fd9792b1ac5bd9))
* indicate more clearly that this is only a docker compose preview ([#1548](https://github.com/glasskube/distr/issues/1548)) ([99dc09f](https://github.com/glasskube/distr/commit/99dc09fbf400de11af730cc9e9c04bbfda5d3a04))


### Docs

* add CLAUDE.md ([#1546](https://github.com/glasskube/distr/issues/1546)) ([f823c75](https://github.com/glasskube/distr/commit/f823c755a59262c197614e75ab88673345e976da))

## [2.0.2](https://github.com/glasskube/distr/compare/2.0.1...2.0.2) (2025-12-09)


### Bug Fixes

* **registry:** ensure that no artifact license allows read all ([#1534](https://github.com/glasskube/distr/issues/1534)) ([92f89aa](https://github.com/glasskube/distr/commit/92f89aa0c704c5847388027bf4e4a3e9e378bf74))

## [2.0.1](https://github.com/glasskube/distr/compare/2.0.0...2.0.1) (2025-12-09)


### Bug Fixes

* allow customer qty 0 ([#1532](https://github.com/glasskube/distr/issues/1532)) ([23082cf](https://github.com/glasskube/distr/commit/23082cf97098a344291b4c655a7eed1014b5f799))

## [2.0.0](https://github.com/glasskube/distr/compare/1.16.1...2.0.0) (2025-12-09)


### ⚠ BREAKING CHANGES

* add customer organizations ([#1388](https://github.com/glasskube/distr/issues/1388))

### Features

* add a subscription update confirmation modal ([#1494](https://github.com/glasskube/distr/issues/1494)) ([aba1e3c](https://github.com/glasskube/distr/commit/aba1e3c1055eda0216deb8fbcf369644ee48f547))
* add customer organizations ([#1388](https://github.com/glasskube/distr/issues/1388)) ([d7084eb](https://github.com/glasskube/distr/commit/d7084ebe0b189c73becf34f3f5af9b7e91a7fa67))
* only display subscription banner for vendors ([#1511](https://github.com/glasskube/distr/issues/1511)) ([74f1429](https://github.com/glasskube/distr/commit/74f142910de46ba1af142127a987f09964f3d94e))
* subscription management via Stripe ([#1426](https://github.com/glasskube/distr/issues/1426)) ([d496d14](https://github.com/glasskube/distr/commit/d496d1448c92212428960725a1d3fd30d05fd0a4))
* user roles ([#1448](https://github.com/glasskube/distr/issues/1448)) ([de7af21](https://github.com/glasskube/distr/commit/de7af21f8e4797bfe034e4cc6e6a7620e99f4c5f))


### Bug Fixes

* add support for SMTP with implicit TLS ([#1469](https://github.com/glasskube/distr/issues/1469)) ([e02d6a5](https://github.com/glasskube/distr/commit/e02d6a552dc65a7f374ff2e911c4162954ee1a8a))
* always run organization updates in transactions ([#1496](https://github.com/glasskube/distr/issues/1496)) ([156ed00](https://github.com/glasskube/distr/commit/156ed00544c2bb74c5c5b05ed932e499daaaa634))
* **backend:** fix visibility issues for deployments endpoints ([#1504](https://github.com/glasskube/distr/issues/1504)) ([5486382](https://github.com/glasskube/distr/commit/5486382649129c770d4f0ed2da9fd1600286f94f))
* **backend:** prevent error when two deployment target status with same timestamp ([#1435](https://github.com/glasskube/distr/issues/1435)) ([54f0436](https://github.com/glasskube/distr/commit/54f04364ff5ac44f751df5ef2fa8fd2d1c15dba7))
* **deps:** update aws-sdk-go-v2 monorepo ([#1430](https://github.com/glasskube/distr/issues/1430)) ([71c9b66](https://github.com/glasskube/distr/commit/71c9b66b9837898c3fb7b9d6a4cc4ae16df58a99))
* **deps:** update aws-sdk-go-v2 monorepo ([#1450](https://github.com/glasskube/distr/issues/1450)) ([9872064](https://github.com/glasskube/distr/commit/98720648c7c7c75b9f2a48e444d6935188e22fd2))
* **deps:** update aws-sdk-go-v2 monorepo ([#1485](https://github.com/glasskube/distr/issues/1485)) ([8806019](https://github.com/glasskube/distr/commit/8806019b2e2aa59071f64fe9cbe22f30c7f6732f))
* **deps:** update aws-sdk-go-v2 monorepo ([#1521](https://github.com/glasskube/distr/issues/1521)) ([89a85fc](https://github.com/glasskube/distr/commit/89a85fc751fde7071abad6b6321be7e7d9eb9ef8))
* **deps:** update module github.com/aws/smithy-go to v1.24.0 ([#1473](https://github.com/glasskube/distr/issues/1473)) ([63db29c](https://github.com/glasskube/distr/commit/63db29c028e85fb6f7ed8e902439b1763904e60d))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.39.0 ([#1442](https://github.com/glasskube/distr/issues/1442)) ([0a954cc](https://github.com/glasskube/distr/commit/0a954cc4b09b56e3ec1651e8294b612e167b85de))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.40.0 ([#1464](https://github.com/glasskube/distr/issues/1464)) ([05db0da](https://github.com/glasskube/distr/commit/05db0dab21114fd05e08a94113f8aff69b00cc6c))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.18.1 ([#1441](https://github.com/glasskube/distr/issues/1441)) ([934d2fa](https://github.com/glasskube/distr/commit/934d2fa3168f68dc587ff54852ffd85cf9676d9d))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.18.2 ([#1452](https://github.com/glasskube/distr/issues/1452)) ([081f05f](https://github.com/glasskube/distr/commit/081f05fdd559d9df126963aef9307a161709e0c6))
* **deps:** update module github.com/golang-migrate/migrate/v4 to v4.19.1 ([#1468](https://github.com/glasskube/distr/issues/1468)) ([b622e3b](https://github.com/glasskube/distr/commit/b622e3b6213431a2ac4c79564ca056545d8f52df))
* **deps:** update module github.com/mark3labs/mcp-go to v0.43.1 ([#1431](https://github.com/glasskube/distr/issues/1431)) ([b1b0b9b](https://github.com/glasskube/distr/commit/b1b0b9bb5492738d3707a0b5cd5b6c4596084753))
* **deps:** update module github.com/mark3labs/mcp-go to v0.43.2 ([#1503](https://github.com/glasskube/distr/issues/1503)) ([b965317](https://github.com/glasskube/distr/commit/b9653178e8946cec3b101f10b13adfa62212e07e))
* **deps:** update module github.com/onsi/gomega to v1.38.3 ([#1518](https://github.com/glasskube/distr/issues/1518)) ([c1ccadc](https://github.com/glasskube/distr/commit/c1ccadcccc06dee4ff3e08629b064117dd16c10c))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.141.0 ([#1474](https://github.com/glasskube/distr/issues/1474)) ([bb10280](https://github.com/glasskube/distr/commit/bb1028052f0f83ab3164bb42ca2a7de0998d093e))
* **deps:** update module github.com/spf13/cobra to v1.10.2 ([#1501](https://github.com/glasskube/distr/issues/1501)) ([5ea971a](https://github.com/glasskube/distr/commit/5ea971a2d6e99e5ea5dce0996d977a877ba7a9ae))
* **deps:** update module github.com/stripe/stripe-go/v83 to v84 ([#1480](https://github.com/glasskube/distr/issues/1480)) ([02707eb](https://github.com/glasskube/distr/commit/02707eb29d55ecd9f6ace2dfd918699f9ff9bd85))
* **deps:** update module golang.org/x/crypto to v0.46.0 ([#1522](https://github.com/glasskube/distr/issues/1522)) ([3cb5fc1](https://github.com/glasskube/distr/commit/3cb5fc13843d93ba706f5aafea715f76fcf4f506))
* **deps:** update module golang.org/x/oauth2 to v0.34.0 ([#1517](https://github.com/glasskube/distr/issues/1517)) ([d5c83a3](https://github.com/glasskube/distr/commit/d5c83a37d7aa7c153050d2e0ae5c1d2d2a9bf871))
* **deps:** update opentelemetry-go monorepo to v1.39.0 ([#1523](https://github.com/glasskube/distr/issues/1523)) ([67b081e](https://github.com/glasskube/distr/commit/67b081e2f9d7ada3307912e8cbb4e44483b37d43))
* **deps:** update opentelemetry-go-contrib monorepo to v0.64.0 ([#1526](https://github.com/glasskube/distr/issues/1526)) ([1ac9897](https://github.com/glasskube/distr/commit/1ac9897c49e5e2878862637ae63cd25636721671))
* fix deployment target visibility for customer organizations ([#1445](https://github.com/glasskube/distr/issues/1445)) ([2b1956f](https://github.com/glasskube/distr/commit/2b1956fa3babbcd2d874e867b6650a7a05601ae0))
* populate CustomerOrganizationID when switching auth context ([#1491](https://github.com/glasskube/distr/issues/1491)) ([c82f08f](https://github.com/glasskube/distr/commit/c82f08f41d12c7dbd7ae1f09841a1c17db41bed3))
* **registry:** fix access control for write actions ([#1506](https://github.com/glasskube/distr/issues/1506)) ([8c50d0a](https://github.com/glasskube/distr/commit/8c50d0ab28d1a563221140f4ce97fdab1027b6fe))
* **ui:** do not show install wizard for users with `read_only` role ([#1505](https://github.com/glasskube/distr/issues/1505)) ([d4d7d1e](https://github.com/glasskube/distr/commit/d4d7d1e7ddec48b6e61d51a084ca1cfafca095ce))
* **ui:** fix button hover style ([#1497](https://github.com/glasskube/distr/issues/1497)) ([3b6c640](https://github.com/glasskube/distr/commit/3b6c640e5a5ed85ca71b7b6010a37fd11f10a78f))
* **ui:** only display subscription upgrade notice to vendor admins ([#1489](https://github.com/glasskube/distr/issues/1489)) ([fa71ddb](https://github.com/glasskube/distr/commit/fa71ddbdaf1d9ff478d1b346123f5ac3de277b56))


### Other

* add enforcement of starter features ([#1525](https://github.com/glasskube/distr/issues/1525)) ([fff3c49](https://github.com/glasskube/distr/commit/fff3c498033c85038b0831126db5638fdd1a99de))
* add link to Distr Homepage ([#1454](https://github.com/glasskube/distr/issues/1454)) ([ac50a07](https://github.com/glasskube/distr/commit/ac50a078bc1eb11899f61daa72e2198228ca122e))
* add posthog groups ([#1460](https://github.com/glasskube/distr/issues/1460)) ([9807ac9](https://github.com/glasskube/distr/commit/9807ac9c8af6d16095abdbe10567826b299747ee))
* add subscription type "community" ([#1514](https://github.com/glasskube/distr/issues/1514)) ([c5c850c](https://github.com/glasskube/distr/commit/c5c850c4bbf6652b6f8c6e4ad43a584f02cbb89c))
* change image reference in compose, chart to CE ([#1531](https://github.com/glasskube/distr/issues/1531)) ([96e89b4](https://github.com/glasskube/distr/commit/96e89b40392d0c7b2cd685304b8c22d7e29dec72))
* **deps:** update actions/checkout action to v6.0.1 ([#1484](https://github.com/glasskube/distr/issues/1484)) ([bf39358](https://github.com/glasskube/distr/commit/bf39358860a5749e5caa1b2eccd60219d7dc88e9))
* **deps:** update actions/setup-node action to v6.1.0 ([#1486](https://github.com/glasskube/distr/issues/1486)) ([6986992](https://github.com/glasskube/distr/commit/69869928fe07861319106b27ef09f398de81cbc5))
* **deps:** update angular monorepo to v20.3.14 ([#1449](https://github.com/glasskube/distr/issues/1449)) ([4d1d6a0](https://github.com/glasskube/distr/commit/4d1d6a0dd5393b954147edfb699bdfa5b6230037))
* **deps:** update angular monorepo to v20.3.15 ([#1470](https://github.com/glasskube/distr/issues/1470)) ([ec8e533](https://github.com/glasskube/distr/commit/ec8e533ce8de9529a82a88a1647cd9bafa06e590))
* **deps:** update angular-cli monorepo to v20.3.12 ([#1447](https://github.com/glasskube/distr/issues/1447)) ([414cad7](https://github.com/glasskube/distr/commit/414cad70619c9f793816dc09db7c1f27ead3c94c))
* **deps:** update angular-cli monorepo to v20.3.13 ([#1499](https://github.com/glasskube/distr/issues/1499)) ([d60147d](https://github.com/glasskube/distr/commit/d60147dad2ce0678a16160f2ddccbf4b28348655))
* **deps:** update axllent/mailpit docker tag to v1.28.0 ([#1451](https://github.com/glasskube/distr/issues/1451)) ([b51ccca](https://github.com/glasskube/distr/commit/b51cccacc7c347d8582b77411e7e91e25a24f5a1))
* **deps:** update dependency @codemirror/view to v6.39.1 ([#1520](https://github.com/glasskube/distr/issues/1520)) ([57598bf](https://github.com/glasskube/distr/commit/57598bf003929ff96a79ac74ef37e9a38938507c))
* **deps:** update dependency @codemirror/view to v6.39.2 ([#1530](https://github.com/glasskube/distr/issues/1530)) ([4040cc1](https://github.com/glasskube/distr/commit/4040cc186b587d0c798de722a622eaf3e96da349))
* **deps:** update dependency go to v1.25.5 ([#1481](https://github.com/glasskube/distr/issues/1481)) ([3d39453](https://github.com/glasskube/distr/commit/3d39453909e639668f00036d109ac5de7d26fb56))
* **deps:** update dependency golangci-lint to v2.7.0 ([#1500](https://github.com/glasskube/distr/issues/1500)) ([847411d](https://github.com/glasskube/distr/commit/847411d37747b0eee78c99ffad9f9f0d7c41a63d))
* **deps:** update dependency golangci-lint to v2.7.1 ([#1507](https://github.com/glasskube/distr/issues/1507)) ([d5d084f](https://github.com/glasskube/distr/commit/d5d084f83cd61dc31e915647d6997495f7843bf3))
* **deps:** update dependency golangci-lint to v2.7.2 ([#1516](https://github.com/glasskube/distr/issues/1516)) ([b22662c](https://github.com/glasskube/distr/commit/b22662c491a38c18103dececa3d106800c8e00f4))
* **deps:** update dependency jasmine-core to ~5.13.0 ([#1476](https://github.com/glasskube/distr/issues/1476)) ([00d7fb9](https://github.com/glasskube/distr/commit/00d7fb9b7dd16c10a7ede70d08f8083148e3f3e6))
* **deps:** update dependency pnpm to v10.24.0 ([#1463](https://github.com/glasskube/distr/issues/1463)) ([5270b43](https://github.com/glasskube/distr/commit/5270b4330064eeb985e24899a783d56906957813))
* **deps:** update dependency pnpm to v10.25.0 ([#1519](https://github.com/glasskube/distr/issues/1519)) ([ae49b93](https://github.com/glasskube/distr/commit/ae49b93dc7bb145b20f3a709c20d0cb2f697c169))
* **deps:** update dependency posthog-js to v1.298.0 ([#1433](https://github.com/glasskube/distr/issues/1433)) ([744d2eb](https://github.com/glasskube/distr/commit/744d2eb97d6439ba57eda1068c4c94fd7f8fc71f))
* **deps:** update dependency prettier to v3.7.0 ([#1453](https://github.com/glasskube/distr/issues/1453)) ([b2ccd7b](https://github.com/glasskube/distr/commit/b2ccd7b2d6c7af0cd0ff53528b8eb38b8976d919))
* **deps:** update dependency prettier to v3.7.1 ([#1456](https://github.com/glasskube/distr/issues/1456)) ([edb723e](https://github.com/glasskube/distr/commit/edb723e50818775de67f550ce10bc4971739acc7))
* **deps:** update dependency prettier to v3.7.2 ([#1465](https://github.com/glasskube/distr/issues/1465)) ([ee7b9f4](https://github.com/glasskube/distr/commit/ee7b9f48d2d5f0e65357c0995ce9edfd89d55f49))
* **deps:** update dependency prettier to v3.7.3 ([#1467](https://github.com/glasskube/distr/issues/1467)) ([718dc32](https://github.com/glasskube/distr/commit/718dc32dad5ac2a363a28659b2164dc135e30665))
* **deps:** update dependency prettier to v3.7.4 ([#1488](https://github.com/glasskube/distr/issues/1488)) ([4288f58](https://github.com/glasskube/distr/commit/4288f58dafce00f1d5a365fc8b46604281f28073))
* **deps:** update dependency typedoc to v0.28.15 ([#1466](https://github.com/glasskube/distr/issues/1466)) ([98d09dc](https://github.com/glasskube/distr/commit/98d09dcb449e2235d321589521f2a8501f5d4efa))
* **deps:** update docker/metadata-action action to v5.10.0 ([#1459](https://github.com/glasskube/distr/issues/1459)) ([02b59d5](https://github.com/glasskube/distr/commit/02b59d5b01d0929de7640c1f2555a8909fc58558))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to 2b7c93f ([#1509](https://github.com/glasskube/distr/issues/1509)) ([c414221](https://github.com/glasskube/distr/commit/c41422158115556812d31eb79e3998a9e88e9956))
* **deps:** update ghcr.io/glasskube/distr docker tag to v1.16.2 ([#1438](https://github.com/glasskube/distr/issues/1438)) ([ece3264](https://github.com/glasskube/distr/commit/ece3264ed449a990baeb666d0ebb2b2e43bb341a))
* **deps:** update ghcr.io/glasskube/distr docker tag to v1.16.3 ([#1461](https://github.com/glasskube/distr/issues/1461)) ([9722ab3](https://github.com/glasskube/distr/commit/9722ab339bfe1672474157a3255287d290bb3366))
* **deps:** update golangci/golangci-lint-action action to v9.1.0 ([#1429](https://github.com/glasskube/distr/issues/1429)) ([2a3214b](https://github.com/glasskube/distr/commit/2a3214b0cd9986387872285227606077c78c09e6))
* **deps:** update golangci/golangci-lint-action action to v9.2.0 ([#1487](https://github.com/glasskube/distr/issues/1487)) ([437334f](https://github.com/glasskube/distr/commit/437334f26d9e75b8069bbb55c613701587368738))
* **deps:** update sentry-javascript monorepo to v10.26.0 ([#1434](https://github.com/glasskube/distr/issues/1434)) ([e53501d](https://github.com/glasskube/distr/commit/e53501daa0cb90013f82dd018895cea2758d9d5c))
* disable license checks if there are no licenses ([#1446](https://github.com/glasskube/distr/issues/1446)) ([70258b6](https://github.com/glasskube/distr/commit/70258b6102d8c58830dde6939b38f8b31fb57c1e))
* disable user roles in starter plan ([#1510](https://github.com/glasskube/distr/issues/1510)) ([a043d5b](https://github.com/glasskube/distr/commit/a043d5b309ec8d089b5cf22451a22a9c61113024))
* fix typo ([#1508](https://github.com/glasskube/distr/issues/1508)) ([215a391](https://github.com/glasskube/distr/commit/215a39148936317dfdf2f597e1d025d516debe5c))
* improve quota styling ([#1527](https://github.com/glasskube/distr/issues/1527)) ([b81f732](https://github.com/glasskube/distr/commit/b81f732ecfdfaa67e88db629a416e02285e1d6c2))
* **registry:** improve Error for violating immutable tags ([#1305](https://github.com/glasskube/distr/issues/1305)) ([fb05687](https://github.com/glasskube/distr/commit/fb05687035b8344d2e327b4462c17b0115aa49fa))
* remove debug logs ([#1492](https://github.com/glasskube/distr/issues/1492)) ([c37ebe5](https://github.com/glasskube/distr/commit/c37ebe5901d2a982260720dc3ab30bbb245b7d7f))
* set trial end date in checkout session ([#1515](https://github.com/glasskube/distr/issues/1515)) ([23d8c12](https://github.com/glasskube/distr/commit/23d8c129b7632f7d8cb7ee891e75c848c3de4b51))
* subscription page improvements ([#1524](https://github.com/glasskube/distr/issues/1524)) ([e2dc393](https://github.com/glasskube/distr/commit/e2dc39351a594eb5f582dc54cf2a0747bdb03a38))
* **ui:** rename customer organization to customer, customer user to user ([#1502](https://github.com/glasskube/distr/issues/1502)) ([5e7bf98](https://github.com/glasskube/distr/commit/5e7bf98c2b0262b0c834f75101eab5f2b425f59c))


### Docs

* add pre-release section ([#1529](https://github.com/glasskube/distr/issues/1529)) ([91c7068](https://github.com/glasskube/distr/commit/91c7068abd895cec3de61da6473beaa5a88a17ce))
* update sign-up url ([#1457](https://github.com/glasskube/distr/issues/1457)) ([f0b8e8f](https://github.com/glasskube/distr/commit/f0b8e8f09c92c257f0a99faa185ad141ad5a36ca))


### Performance

* **backend:** optimize DeploymentLogRecord cleanup routine ([#1455](https://github.com/glasskube/distr/issues/1455)) ([2caecc7](https://github.com/glasskube/distr/commit/2caecc7fc157f6de2d1f124c2a4ebee61d8a61b0))


### Refactoring

* always use -1 for unlimited subscription quantities ([#1490](https://github.com/glasskube/distr/issues/1490)) ([03f8576](https://github.com/glasskube/distr/commit/03f85764b8b86c8fd257cf8f0eb6dabd60815b02))
* don't wrap static boolean in signal ([#1513](https://github.com/glasskube/distr/issues/1513)) ([bf7b226](https://github.com/glasskube/distr/commit/bf7b226ab8a665e44537c4d6bcdcdeca48a14e61))

## [1.16.1](https://github.com/glasskube/distr/compare/1.16.0...1.16.1) (2025-11-21)


### Other

* force new release (no changes) ([#1423](https://github.com/glasskube/distr/issues/1423)) ([ec9b3f8](https://github.com/glasskube/distr/commit/ec9b3f83659f2cdddb79bf8192316303fbcd8c73))

## [1.16.0](https://github.com/glasskube/distr/compare/1.15.2...1.16.0) (2025-11-21)


### Features

* add markdown preview on organization branding page ([#1391](https://github.com/glasskube/distr/issues/1391)) ([eb8e291](https://github.com/glasskube/distr/commit/eb8e291b2a88cb92a6abd57c3a91ac52c6175bfc))
* generic OIDC provider ([#1208](https://github.com/glasskube/distr/issues/1208)) ([529ee18](https://github.com/glasskube/distr/commit/529ee184a2e2d2d68af948916878a62ecfd7914d))


### Bug Fixes

* **deps:** update angular monorepo ([#1407](https://github.com/glasskube/distr/issues/1407)) ([131ee3b](https://github.com/glasskube/distr/commit/131ee3b4b63aa7c9284e53d90bb928b24aa81128))
* **deps:** update aws-sdk-go-v2 monorepo ([#1410](https://github.com/glasskube/distr/issues/1410)) ([01e4109](https://github.com/glasskube/distr/commit/01e41098d04c179191fed4cc17396816408fd1f5))
* **deps:** update dependency @sentry/angular to v10 ([#1400](https://github.com/glasskube/distr/issues/1400)) ([56f4116](https://github.com/glasskube/distr/commit/56f4116a22402e550d34528596b9cccef8b26865))
* **deps:** update dependency marked to v17 ([#1403](https://github.com/glasskube/distr/issues/1403)) ([61d25ed](https://github.com/glasskube/distr/commit/61d25edaad71d0f7b4d7538fea98db2cb1453f49))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.92.0 ([#1420](https://github.com/glasskube/distr/issues/1420)) ([b8bc87f](https://github.com/glasskube/distr/commit/b8bc87f50e06cf49a2c62de7c92c52cf051933f7))
* **deps:** update module github.com/coreos/go-oidc/v3 to v3.17.0 ([#1421](https://github.com/glasskube/distr/issues/1421)) ([0bd5069](https://github.com/glasskube/distr/commit/0bd5069761b4e728b04ee98e1be746c9047995a7))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.38.0 ([#1379](https://github.com/glasskube/distr/issues/1379)) ([4401cd6](https://github.com/glasskube/distr/commit/4401cd6f45d6fb1fb57f26d74b103624ad4706d1))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.140.1 ([#1382](https://github.com/glasskube/distr/issues/1382)) ([79ce232](https://github.com/glasskube/distr/commit/79ce2329a03786f4e5ec6f579c50c4a5c98cd1e2))
* **deps:** update module go.uber.org/zap to v1.27.1 ([#1409](https://github.com/glasskube/distr/issues/1409)) ([a457f47](https://github.com/glasskube/distr/commit/a457f478b7378bedb59e3f3940539b34f23d98b9))
* **deps:** update module golang.org/x/crypto to v0.44.0 ([#1360](https://github.com/glasskube/distr/issues/1360)) ([1f25d02](https://github.com/glasskube/distr/commit/1f25d02d9501ae17e249a0beb5301a8da1ce98c7))
* **deps:** update module golang.org/x/crypto to v0.45.0 [security] ([#1411](https://github.com/glasskube/distr/issues/1411)) ([4dd31a4](https://github.com/glasskube/distr/commit/4dd31a4242eb901c0c5a86fc6710f2ea7fa0ebd1))


### Other

* **deps:** update actions/checkout action to v5.0.1 ([#1380](https://github.com/glasskube/distr/issues/1380)) ([22f9843](https://github.com/glasskube/distr/commit/22f9843b42c484549cc0d27acad824304e99db8c))
* **deps:** update actions/checkout action to v6 ([#1419](https://github.com/glasskube/distr/issues/1419)) ([4e1acfd](https://github.com/glasskube/distr/commit/4e1acfd0ed90d0fb8ec2c5512cf6922d27dcd828))
* **deps:** update actions/setup-go action to v6.1.0 ([#1412](https://github.com/glasskube/distr/issues/1412)) ([d805a94](https://github.com/glasskube/distr/commit/d805a945e2047ceb28a4b699a73f0ff8b67306b8))
* **deps:** update anchore/sbom-action action to v0.20.10 ([#1381](https://github.com/glasskube/distr/issues/1381)) ([3caa51d](https://github.com/glasskube/distr/commit/3caa51daa5c54d9843e6b05692551f0849c0d1fd))
* **deps:** update angular-cli monorepo to v20.3.11 ([#1404](https://github.com/glasskube/distr/issues/1404)) ([c2b409b](https://github.com/glasskube/distr/commit/c2b409b7a0ec7660ae9178a21d6490fb1cc740dd))
* **deps:** update dependency @angular/cdk to v20.2.14 ([#1406](https://github.com/glasskube/distr/issues/1406)) ([d6d9636](https://github.com/glasskube/distr/commit/d6d9636ac74061d8b0b1c8d6c2607ac62c1ee7b1))
* **deps:** update dependency @sentry/angular to v9.47.1 ([#1397](https://github.com/glasskube/distr/issues/1397)) ([5642db7](https://github.com/glasskube/distr/commit/5642db7e5e560334221823a6d689d58e0d9d2588))
* **deps:** update dependency @sentry/cli to v2.58.2 ([#1393](https://github.com/glasskube/distr/issues/1393)) ([17a3f60](https://github.com/glasskube/distr/commit/17a3f606334a47c2207ad07d262580aa1db950b7))
* **deps:** update dependency @types/jasmine to v5.1.13 ([#1373](https://github.com/glasskube/distr/issues/1373)) ([4f75704](https://github.com/glasskube/distr/commit/4f757045af2a0e3d8281342e11915629c6d70afc))
* **deps:** update dependency golangci-lint to v2.6.2 ([#1375](https://github.com/glasskube/distr/issues/1375)) ([98a8801](https://github.com/glasskube/distr/commit/98a88019c908f6a81d180a366c338410318a260b))
* **deps:** update dependency pnpm to v10.23.0 ([#1418](https://github.com/glasskube/distr/issues/1418)) ([7470080](https://github.com/glasskube/distr/commit/74700803b82f8e4166d1bbd0d34a7cedf1bd4cf4))
* **deps:** update dependency posthog-js to v1.296.0 ([#1398](https://github.com/glasskube/distr/issues/1398)) ([a96fc7e](https://github.com/glasskube/distr/commit/a96fc7e37b8ea80d27fa42d2b9ff35f379687536))
* **deps:** update dependency rimraf to v6.1.2 ([#1405](https://github.com/glasskube/distr/issues/1405)) ([a87a1ba](https://github.com/glasskube/distr/commit/a87a1ba25119c1ca6c197d29d46f8c3a7753ad52))
* **deps:** update ghcr.io/glasskube/hello-distr/backend docker tag to v0.1.11 ([#1394](https://github.com/glasskube/distr/issues/1394)) ([bd2aab5](https://github.com/glasskube/distr/commit/bd2aab5b6bf1a190f3528259703762800ad570c4))
* **deps:** update ghcr.io/glasskube/hello-distr/frontend docker tag to v0.1.11 ([#1395](https://github.com/glasskube/distr/issues/1395)) ([0db8a92](https://github.com/glasskube/distr/commit/0db8a928ea128a19fc859f5f56c32344200c54ae))
* **deps:** update ghcr.io/glasskube/hello-distr/proxy docker tag to v0.1.11 ([#1396](https://github.com/glasskube/distr/issues/1396)) ([2928543](https://github.com/glasskube/distr/commit/2928543b34fda64931ecd7210ca23cbf30520e25))
* **deps:** update npm dependencies ([#1384](https://github.com/glasskube/distr/issues/1384)) ([6ded2fd](https://github.com/glasskube/distr/commit/6ded2fd5d219376bd75dae8a79682f646a71bf69))
* move "delete artifact" button to artifact details page ([#1390](https://github.com/glasskube/distr/issues/1390)) ([4c34939](https://github.com/glasskube/distr/commit/4c349394ca7191cb9103736bcd04091986b5b1dd))

## [1.15.2](https://github.com/glasskube/distr/compare/1.15.1...1.15.2) (2025-11-17)


### Bug Fixes

* **deps:** update aws-sdk-go-v2 monorepo ([#1347](https://github.com/glasskube/distr/issues/1347)) ([3d0debd](https://github.com/glasskube/distr/commit/3d0debd554657ee99f62dc0609a6f30368ecb3d4))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.18.0 ([#1348](https://github.com/glasskube/distr/issues/1348)) ([a39076a](https://github.com/glasskube/distr/commit/a39076afbdda6e4230c4bb830da43af5590b4eec))
* **deps:** update module golang.org/x/oauth2 to v0.33.0 ([#1353](https://github.com/glasskube/distr/issues/1353)) ([9ec1626](https://github.com/glasskube/distr/commit/9ec1626caa3f82c41623d4d68edd950fc125abec))
* missing branding in email templates ([#1374](https://github.com/glasskube/distr/issues/1374)) ([097eb84](https://github.com/glasskube/distr/commit/097eb843e84ebc322ad8b92e036418174cb2de5f))


### Other

* **deps:** update angular monorepo to v20.3.12 ([#1369](https://github.com/glasskube/distr/issues/1369)) ([db84463](https://github.com/glasskube/distr/commit/db84463995b2e02612dc9ea7c049ab14ff75de5e))
* **deps:** update dependency @angular/cdk to v20.2.13 ([#1370](https://github.com/glasskube/distr/issues/1370)) ([74e3aba](https://github.com/glasskube/distr/commit/74e3aba5c53ff283c93849d9b6242e944955cc40))
* **deps:** update dependency @codemirror/view to v6.38.8 ([#1372](https://github.com/glasskube/distr/issues/1372)) ([75779bf](https://github.com/glasskube/distr/commit/75779bfa8878fa137369aca4565603fde3548bb2))
* **deps:** update dependency go to v1.25.4 ([#1345](https://github.com/glasskube/distr/issues/1345)) ([d48c2b7](https://github.com/glasskube/distr/commit/d48c2b79b34b849ab592e8869024ebf84ff3d824))
* **deps:** update dependency posthog-js to v1.293.0 ([#1356](https://github.com/glasskube/distr/issues/1356)) ([4bbbc88](https://github.com/glasskube/distr/commit/4bbbc880bcd8b74335e1107a2aa4ee43568eac4b))
* **deps:** update golangci/golangci-lint-action action to v9 ([#1352](https://github.com/glasskube/distr/issues/1352)) ([e2054d1](https://github.com/glasskube/distr/commit/e2054d169e6caf597e8e6b2b90b2ee649bd28288))

## [1.15.1](https://github.com/glasskube/distr/compare/1.15.0...1.15.1) (2025-11-13)


### Bug Fixes

* **deps:** update aws-sdk-go-v2 monorepo ([#1341](https://github.com/glasskube/distr/issues/1341)) ([d445ed9](https://github.com/glasskube/distr/commit/d445ed9d0e9d3d41e19ca4a1d88d8968b9f49845))
* **deps:** update kubernetes packages to v0.34.2 ([#1365](https://github.com/glasskube/distr/issues/1365)) ([e495120](https://github.com/glasskube/distr/commit/e495120c520de85e439e551d71b3df9c2f86d2d2))
* **deps:** update module github.com/docker/cli to v28.5.2+incompatible ([#1343](https://github.com/glasskube/distr/issues/1343)) ([aeb7e83](https://github.com/glasskube/distr/commit/aeb7e830a4fac65c5b7703507aa67f0990d9da45))
* **deps:** update module github.com/docker/docker to v28.5.2+incompatible ([#1346](https://github.com/glasskube/distr/issues/1346)) ([0ef6dd6](https://github.com/glasskube/distr/commit/0ef6dd6fcd092a00cbd3baae58ea63306e38fe12))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.139.0 ([#1333](https://github.com/glasskube/distr/issues/1333)) ([9de553d](https://github.com/glasskube/distr/commit/9de553d4c9cc61ec6014a52a3c38fb7c819bba57))
* **deps:** update module helm.sh/helm/v3 to v3.19.1 ([#1359](https://github.com/glasskube/distr/issues/1359)) ([c21b9aa](https://github.com/glasskube/distr/commit/c21b9aa1cc7f2e2091dd2b30c5c20cdfeeed7f02))
* **deps:** update module helm.sh/helm/v3 to v3.19.2 ([#1361](https://github.com/glasskube/distr/issues/1361)) ([0d123bc](https://github.com/glasskube/distr/commit/0d123bc0ec80e746fb62d28461fb40433b9b45cb))
* **ui:** align theme icon with it's parent box ([#1323](https://github.com/glasskube/distr/issues/1323)) ([6548244](https://github.com/glasskube/distr/commit/6548244d97a3bd10f26adb77e871068b1db2a8f8))


### Other

* add artifact deletion ([#1367](https://github.com/glasskube/distr/issues/1367)) ([3bc87d6](https://github.com/glasskube/distr/commit/3bc87d6ac106802d1f4342d42939643ed3240c63))
* **deps:** update angular monorepo to v20.3.10 ([#1344](https://github.com/glasskube/distr/issues/1344)) ([2a45f36](https://github.com/glasskube/distr/commit/2a45f362104e417dd3d59cf2964e73626da6887a))
* **deps:** update angular monorepo to v20.3.11 ([#1363](https://github.com/glasskube/distr/issues/1363)) ([abcc1cd](https://github.com/glasskube/distr/commit/abcc1cd5004734ce17f53dac0827a52a365f93ba))
* **deps:** update angular-cli monorepo to v20.3.10 ([#1364](https://github.com/glasskube/distr/issues/1364)) ([ef08177](https://github.com/glasskube/distr/commit/ef081775e4b58ae20b15359e4b67fc1414617934))
* **deps:** update angular-cli monorepo to v20.3.9 ([#1342](https://github.com/glasskube/distr/issues/1342)) ([450d45c](https://github.com/glasskube/distr/commit/450d45c2e7e8171efc11c9bb7ebb21c6d259824c))
* **deps:** update axllent/mailpit docker tag to v1.27.11 ([#1354](https://github.com/glasskube/distr/issues/1354)) ([892e09c](https://github.com/glasskube/distr/commit/892e09cd9bd12bd63ebdd099b4b313a24d1a8004))
* **deps:** update dependency @angular/cdk to v20.2.12 ([#1351](https://github.com/glasskube/distr/issues/1351)) ([38a264a](https://github.com/glasskube/distr/commit/38a264a0d933204f12be3323b1aa3eeb6b27068f))
* **deps:** update dependency @sentry/cli to v2.58.0 ([#1355](https://github.com/glasskube/distr/issues/1355)) ([73c3695](https://github.com/glasskube/distr/commit/73c36950d6c5abaf99f55a5e06bb8dbd526af4b0))
* **deps:** update dependency autoprefixer to v10.4.22 ([#1357](https://github.com/glasskube/distr/issues/1357)) ([6eb65ba](https://github.com/glasskube/distr/commit/6eb65ba6896d60d4cc9d2bb6c1e661fe0dc076ec))
* **deps:** update dependency golangci-lint to v2.6.1 ([#1338](https://github.com/glasskube/distr/issues/1338)) ([69a5040](https://github.com/glasskube/distr/commit/69a504032a59c5bcdf55b5aae1fe3cad35649aa8))
* **deps:** update docker/metadata-action action to v5.9.0 ([#1340](https://github.com/glasskube/distr/issues/1340)) ([908001e](https://github.com/glasskube/distr/commit/908001e5d4f9e33d260d079285c53fa800f83fbc))
* **deps:** update tailwindcss monorepo to v4.1.17 ([#1350](https://github.com/glasskube/distr/issues/1350)) ([42ff709](https://github.com/glasskube/distr/commit/42ff709e64e1d7e39d7540bd4b8330407b1d6a82))

## [1.15.0](https://github.com/glasskube/distr/compare/1.14.2...1.15.0) (2025-11-04)


### Features

* add resending invitation for non-verified users ([#1310](https://github.com/glasskube/distr/issues/1310)) ([7d56c6f](https://github.com/glasskube/distr/commit/7d56c6f60a90249954917b7ecd3f64b028c143c6))


### Bug Fixes

* **chart:** switch postgres to bitnamilegacy ([#1337](https://github.com/glasskube/distr/issues/1337)) ([09c80ad](https://github.com/glasskube/distr/commit/09c80ad2b8d3720f2184cfb5712b75ed5eafb57a))
* **deps:** update aws-sdk-go-v2 monorepo ([#1303](https://github.com/glasskube/distr/issues/1303)) ([d176c19](https://github.com/glasskube/distr/commit/d176c19bf86cac5fb4708995eff338b37a380204))
* **deps:** update module github.com/aws/smithy-go to v1.23.2 ([#1331](https://github.com/glasskube/distr/issues/1331)) ([ed3e074](https://github.com/glasskube/distr/commit/ed3e074706224eed467979e9808dbad1ef5bb80d))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.9.1 ([#1312](https://github.com/glasskube/distr/issues/1312)) ([6eb9b8d](https://github.com/glasskube/distr/commit/6eb9b8dead214447e87501e1e30d75cd0d9cff0e))
* **deps:** update module github.com/docker/compose/v2 to v2.40.2 [security] ([#1297](https://github.com/glasskube/distr/issues/1297)) ([400ae3a](https://github.com/glasskube/distr/commit/400ae3ac2ac5e01857478e010b74e0f6d3087faa))
* **deps:** update module github.com/docker/compose/v2 to v2.40.3 ([#1201](https://github.com/glasskube/distr/issues/1201)) ([5f27b95](https://github.com/glasskube/distr/commit/5f27b9514d5531f6a6d5f4ff68661ddb215aa7a6))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.36.2 ([#1300](https://github.com/glasskube/distr/issues/1300)) ([8a5ac0c](https://github.com/glasskube/distr/commit/8a5ac0c8990d29a4247c0f528c6824e8d59a73ac))
* **deps:** update module github.com/mark3labs/mcp-go to v0.42.0 ([#1291](https://github.com/glasskube/distr/issues/1291)) ([08d54fb](https://github.com/glasskube/distr/commit/08d54fb84afd49b64f3edd383f7b9067249227c4))
* **deps:** update module github.com/mark3labs/mcp-go to v0.43.0 ([#1327](https://github.com/glasskube/distr/issues/1327)) ([753d3e3](https://github.com/glasskube/distr/commit/753d3e39f5f6009c5ec8194129aee200172a338b))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.138.0 ([#1301](https://github.com/glasskube/distr/issues/1301)) ([01e0a05](https://github.com/glasskube/distr/commit/01e0a052d46f034f39fc9af541fb9355f36475e3))
* **deps:** update module golang.org/x/crypto to v0.43.0 ([#1314](https://github.com/glasskube/distr/issues/1314)) ([422222b](https://github.com/glasskube/distr/commit/422222bda7e5326320d62d9a298458d96f1c3a00))
* **deps:** update module golang.org/x/oauth2 to v0.32.0 ([#1316](https://github.com/glasskube/distr/issues/1316)) ([939e7ef](https://github.com/glasskube/distr/commit/939e7efab610c6ffd2ca3f6cadbb18e09ff9eec2))


### Other

* **deps:** update actions/setup-node action to v6 ([#1319](https://github.com/glasskube/distr/issues/1319)) ([22770cd](https://github.com/glasskube/distr/commit/22770cd4e0ed93dbd942737fda33b8ba03cad883))
* **deps:** update angular monorepo to v20.3.9 ([#1308](https://github.com/glasskube/distr/issues/1308)) ([be60758](https://github.com/glasskube/distr/commit/be60758dc1762ded7d4bd67794cb140d96f1b978))
* **deps:** update angular-cli monorepo to v20.3.8 ([#1311](https://github.com/glasskube/distr/issues/1311)) ([8cdaaaf](https://github.com/glasskube/distr/commit/8cdaaafcf5cd3ea43618d8ddd679f9582339f5e2))
* **deps:** update dependency @angular/cdk to v20.2.11 ([#1326](https://github.com/glasskube/distr/issues/1326)) ([cb20b26](https://github.com/glasskube/distr/commit/cb20b269c8bddfa28072593848beef16d2a9aa9a))
* **deps:** update dependency @sentry/cli to v2.57.0 ([#1329](https://github.com/glasskube/distr/issues/1329)) ([c6c3f67](https://github.com/glasskube/distr/commit/c6c3f679c84bacbd858f1802ed5f2abe10e9d7e6))
* **deps:** update dependency dayjs to v1.11.19 ([#1318](https://github.com/glasskube/distr/issues/1318)) ([8c63f80](https://github.com/glasskube/distr/commit/8c63f80d449c562f25ff8cdba4b7e615e83a229d))
* **deps:** update dependency golangci-lint to v2.6.0 ([#1313](https://github.com/glasskube/distr/issues/1313)) ([8e74c62](https://github.com/glasskube/distr/commit/8e74c62b240f7e87df32e81678fb3538371294c5))
* **deps:** update dependency jasmine-core to v5.12.1 ([#1309](https://github.com/glasskube/distr/issues/1309)) ([844525b](https://github.com/glasskube/distr/commit/844525b7879d91bb8c216478c74b4bef5e3fcaf7))
* **deps:** update dependency node to v24 ([#1320](https://github.com/glasskube/distr/issues/1320)) ([f1fb358](https://github.com/glasskube/distr/commit/f1fb3588479fdea0ebc2320fa2b94b762ee6611c))
* **deps:** update dependency rimraf to v6.1.0 ([#1317](https://github.com/glasskube/distr/issues/1317)) ([8621a5c](https://github.com/glasskube/distr/commit/8621a5ccfa1b2dc21d276685d642290fb609687d))
* **deps:** update dependency typedoc to v0.28.14 ([#1263](https://github.com/glasskube/distr/issues/1263)) ([c9eb1c2](https://github.com/glasskube/distr/commit/c9eb1c21d9009acca9f33b852bc778024b98efca))
* **deps:** update fontsource monorepo to v5.2.8 ([#1328](https://github.com/glasskube/distr/issues/1328)) ([f40e7eb](https://github.com/glasskube/distr/commit/f40e7eb897b9fa46a6fa31dcc160d1e128a6c1b7))
* **deps:** update github artifact actions (major) ([#1321](https://github.com/glasskube/distr/issues/1321)) ([db1a355](https://github.com/glasskube/distr/commit/db1a355bf53f6ba7bba4c7a975a9509a7cedecbc))
* **deps:** update sigstore/cosign-installer action to v3.10.1 ([#1330](https://github.com/glasskube/distr/issues/1330)) ([57939d9](https://github.com/glasskube/distr/commit/57939d917fa850fa99daea0bb0b06a22356a8c9d))
* **deps:** update sigstore/cosign-installer action to v4 ([#1324](https://github.com/glasskube/distr/issues/1324)) ([abf6f72](https://github.com/glasskube/distr/commit/abf6f7280342efe9a443fded27b4fe8c43828e4d))

## [1.14.2](https://github.com/glasskube/distr/compare/1.14.1...1.14.2) (2025-10-29)


### Bug Fixes

* **backend:** ensure db connections are cancelled on cleanup timeout ([#1306](https://github.com/glasskube/distr/issues/1306)) ([858b92f](https://github.com/glasskube/distr/commit/858b92f7c923ed81a83c215e9cf20fd1ce923ec6))
* **deps:** update aws-sdk-go-v2 monorepo ([#1289](https://github.com/glasskube/distr/issues/1289)) ([af6746c](https://github.com/glasskube/distr/commit/af6746c4a038615c426ddf050f61c83a397d83e5))
* **deps:** update aws-sdk-go-v2 monorepo ([#1294](https://github.com/glasskube/distr/issues/1294)) ([bb592e9](https://github.com/glasskube/distr/commit/bb592e9223d556c9cfac6f196321741d0c22c43c))
* **deps:** update module github.com/getsentry/sentry-go to v0.36.1 ([#1283](https://github.com/glasskube/distr/issues/1283)) ([33e00ed](https://github.com/glasskube/distr/commit/33e00edacf7a07f4aceab3c91e2dac0cb0d3a18e))
* **deps:** update module github.com/getsentry/sentry-go to v0.36.2 ([#1299](https://github.com/glasskube/distr/issues/1299)) ([f0d7f48](https://github.com/glasskube/distr/commit/f0d7f48800f40007de6ca955d56c0cbc8fde4054))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.36.1 ([#1285](https://github.com/glasskube/distr/issues/1285)) ([d598426](https://github.com/glasskube/distr/commit/d598426927b5e7ec8bd8e57ab0d190e971dc1cfb))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.17.0 ([#1287](https://github.com/glasskube/distr/issues/1287)) ([4c09faf](https://github.com/glasskube/distr/commit/4c09faf5f504420a9f0450abb48eb2ccde103a14))


### Other

* **deps:** update anchore/sbom-action action to v0.20.9 ([#1292](https://github.com/glasskube/distr/issues/1292)) ([0ca3f52](https://github.com/glasskube/distr/commit/0ca3f52774dd5f573c89f3504d66380d0838f2f9))
* **deps:** update angular monorepo to v20.3.7 ([#1288](https://github.com/glasskube/distr/issues/1288)) ([409d810](https://github.com/glasskube/distr/commit/409d810615bdadd137779c1e762c1b152f9a3077))
* **deps:** update angular-cli monorepo to v20.3.7 ([#1290](https://github.com/glasskube/distr/issues/1290)) ([766027b](https://github.com/glasskube/distr/commit/766027baadbdff371a9967d4a71efcd7fbad735b))
* **deps:** update dependency @angular/cdk to v20.2.10 ([#1286](https://github.com/glasskube/distr/issues/1286)) ([edf7105](https://github.com/glasskube/distr/commit/edf7105a85c6d4fdba9e3272f9d684eef575f8d2))
* **deps:** update dependency @codemirror/commands to v6.10.0 ([#1295](https://github.com/glasskube/distr/issues/1295)) ([58d6887](https://github.com/glasskube/distr/commit/58d688721a7693c7f88fd1b58592311761bdb08f))
* **deps:** update dependency @types/jasmine to v5.1.12 ([#1277](https://github.com/glasskube/distr/issues/1277)) ([a1a76f5](https://github.com/glasskube/distr/commit/a1a76f5844e1bb0a5f36d6a9ae633156149169cc))
* **deps:** update docker/login-action action to v3.6.0 ([#1284](https://github.com/glasskube/distr/issues/1284)) ([8c5d285](https://github.com/glasskube/distr/commit/8c5d285243d3d0d3ce611abe9b6eb82ddf3f7472))
* **deps:** update googleapis/release-please-action action to v4.4.0 ([#1296](https://github.com/glasskube/distr/issues/1296)) ([5a05c99](https://github.com/glasskube/distr/commit/5a05c9979f7000a5b455ef09716adabc7f7617a4))
* **deps:** update tailwindcss monorepo to v4.1.15 ([#1282](https://github.com/glasskube/distr/issues/1282)) ([ab5c458](https://github.com/glasskube/distr/commit/ab5c458e38651f64fdac4867bcd6a1b67c680f4e))
* **deps:** update tailwindcss monorepo to v4.1.16 ([#1293](https://github.com/glasskube/distr/issues/1293)) ([d5aa69d](https://github.com/glasskube/distr/commit/d5aa69d0fdc30cc857ee3657a55c8609e88aa25e))


### Docs

* add hosted distr mcp to README.md ([#1279](https://github.com/glasskube/distr/issues/1279)) ([9c287a9](https://github.com/glasskube/distr/commit/9c287a924d60fc0e777719feff40ae4abfda5484))

## [1.14.1](https://github.com/glasskube/distr/compare/1.14.0...1.14.1) (2025-10-21)


### Bug Fixes

* **agent:** disable JWT validation ([#1275](https://github.com/glasskube/distr/issues/1275)) ([dc70e6a](https://github.com/glasskube/distr/commit/dc70e6a93db91e4d645324c5a8d67e3dad10a949))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.9.0 ([#1270](https://github.com/glasskube/distr/issues/1270)) ([3621bbf](https://github.com/glasskube/distr/commit/3621bbf818bf3e672c895596d41ca290124746e1))
* **deps:** update module github.com/coreos/go-oidc/v3 to v3.16.0 ([#1271](https://github.com/glasskube/distr/issues/1271)) ([c2098b5](https://github.com/glasskube/distr/commit/c2098b5d678fb70f589b29800a05ca50b21c0ecf))
* **deps:** update module github.com/docker/docker to v28.5.1+incompatible ([#1272](https://github.com/glasskube/distr/issues/1272)) ([248eea8](https://github.com/glasskube/distr/commit/248eea87b4a5b1c487acb8f1526f11ae43de2ddc))
* **deps:** update module github.com/getsentry/sentry-go to v0.36.0 ([#1273](https://github.com/glasskube/distr/issues/1273)) ([1f610f9](https://github.com/glasskube/distr/commit/1f610f94c69bff075c361f028c48af9c3da101bf))


### Other

* **deps:** update codemirror ([#1234](https://github.com/glasskube/distr/issues/1234)) ([3103eb8](https://github.com/glasskube/distr/commit/3103eb85ec38af53dcf6b4094175794b843cc269))
* **deps:** update dependency go to v1.25.3 ([#1261](https://github.com/glasskube/distr/issues/1261)) ([15691bb](https://github.com/glasskube/distr/commit/15691bb5bf66a70d2b86fd56a484e8fb93dafcfd))
* **deps:** update font awesome to v7.1.0 ([#1266](https://github.com/glasskube/distr/issues/1266)) ([6ce2dbc](https://github.com/glasskube/distr/commit/6ce2dbcc7d9eff3a97054e9ef04e58c87ebf533c))

## [1.14.0](https://github.com/glasskube/distr/compare/1.13.0...1.14.0) (2025-10-20)


### Features

* **mcp:** add streamable http transport to mcp server ([#1262](https://github.com/glasskube/distr/issues/1262)) ([b88235e](https://github.com/glasskube/distr/commit/b88235e5f6d4ea97d0fdee5bcd7ca86686ad7f03))


### Bug Fixes

* **chart:** fix invalid timeout parameter for cleanup jobs ([#1243](https://github.com/glasskube/distr/issues/1243)) ([37bc9f5](https://github.com/glasskube/distr/commit/37bc9f53bf35fcf633247d6c7037a92156663318))
* **deps:** update angular monorepo to v20.3.0 ([#1187](https://github.com/glasskube/distr/issues/1187)) ([4e9d528](https://github.com/glasskube/distr/commit/4e9d5284b2d90e273c6d32c6fe22dbaf350c5ebd))
* **deps:** update angular monorepo to v20.3.1 ([#1226](https://github.com/glasskube/distr/issues/1226)) ([732f064](https://github.com/glasskube/distr/commit/732f06469fb61989ce916fd6f9a2240c8cc86141))
* **deps:** update aws-sdk-go-v2 monorepo ([#1235](https://github.com/glasskube/distr/issues/1235)) ([a4f14da](https://github.com/glasskube/distr/commit/a4f14da57e79598e43317cfba1c374add6b9cea3))
* **deps:** update aws-sdk-go-v2 monorepo ([#1269](https://github.com/glasskube/distr/issues/1269)) ([4dadccc](https://github.com/glasskube/distr/commit/4dadccc130c283349fae71aee8248088b333d7a6))
* **deps:** update dependency @angular/cdk to v20.2.3 ([#1178](https://github.com/glasskube/distr/issues/1178)) ([adc0e9e](https://github.com/glasskube/distr/commit/adc0e9e348456dcd82f91e3284f14951cda44a60))
* **deps:** update dependency @angular/cdk to v20.2.4 ([#1227](https://github.com/glasskube/distr/issues/1227)) ([3457df2](https://github.com/glasskube/distr/commit/3457df2ab5cef8bd0d944a41d1129db523eae28d))
* **deps:** update dependency @fontsource/inter to v5.2.7 ([#1206](https://github.com/glasskube/distr/issues/1206)) ([d307d5c](https://github.com/glasskube/distr/commit/d307d5cb2b20d249917e8398a6d0fa7f560cb9ec))
* **deps:** update dependency ngx-markdown to v20.1.0 ([#1194](https://github.com/glasskube/distr/issues/1194)) ([d4de5f6](https://github.com/glasskube/distr/commit/d4de5f63247b73cde861e98d75700e99cd51cd97))
* **deps:** update dependency ngx-toastr to v19.1.0 ([#1212](https://github.com/glasskube/distr/issues/1212)) ([987b1b6](https://github.com/glasskube/distr/commit/987b1b6b64b82f590040cf95a3c097970258b7d7))
* **deps:** update font awesome (major) ([#1229](https://github.com/glasskube/distr/issues/1229)) ([05a1c46](https://github.com/glasskube/distr/commit/05a1c462c3de7f6eaa2f4e6b06514d9194a698f6))
* **deps:** update kubernetes packages to v0.34.1 ([#1195](https://github.com/glasskube/distr/issues/1195)) ([e2a17dd](https://github.com/glasskube/distr/commit/e2a17dde9e31c7d7dde7d279f2bc83fb38c1fc9f))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.8.2 ([#1196](https://github.com/glasskube/distr/issues/1196)) ([30e0a2c](https://github.com/glasskube/distr/commit/30e0a2c0aed5eb1e60730b59118bae2fcd022050))
* **deps:** update module github.com/containers/image/v5 to v5.36.2 ([#1197](https://github.com/glasskube/distr/issues/1197)) ([7c26d4c](https://github.com/glasskube/distr/commit/7c26d4cde915485a36a603d3923d3e3d8eb14d59))
* **deps:** update module github.com/coreos/go-oidc/v3 to v3.15.0 ([#1198](https://github.com/glasskube/distr/issues/1198)) ([93d4870](https://github.com/glasskube/distr/commit/93d4870758dc0ce42f9f9bac59c98140df00144a))
* **deps:** update module github.com/docker/docker to v28.4.0+incompatible ([#1202](https://github.com/glasskube/distr/issues/1202)) ([94fe72e](https://github.com/glasskube/distr/commit/94fe72e0ee6f0ffb6b74876fe1ee2c4ae3e3500a))
* **deps:** update module github.com/getsentry/sentry-go to v0.35.3 ([#1210](https://github.com/glasskube/distr/issues/1210)) ([a3192ed](https://github.com/glasskube/distr/commit/a3192edd797ab990b0feae4567bcb800e2366352))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.35.3 ([#1211](https://github.com/glasskube/distr/issues/1211)) ([83f9619](https://github.com/glasskube/distr/commit/83f9619efbeb8a790d6b6171f59489b960b8885d))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.3 ([#1172](https://github.com/glasskube/distr/issues/1172)) ([00e2c91](https://github.com/glasskube/distr/commit/00e2c9142faf7357176dd76af00dbd25ddcd6229))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.16.5 ([#1173](https://github.com/glasskube/distr/issues/1173)) ([5c122a7](https://github.com/glasskube/distr/commit/5c122a7fff3293b094cc3609564701d1a42f40ac))
* **deps:** update module github.com/go-co-op/gocron/v2 to v2.16.6 ([#1254](https://github.com/glasskube/distr/issues/1254)) ([d6bb23b](https://github.com/glasskube/distr/commit/d6bb23bb5d32e36965071c3508d78304e3c48055))
* **deps:** update module github.com/golang-migrate/migrate/v4 to v4.19.0 ([#1203](https://github.com/glasskube/distr/issues/1203)) ([0e37037](https://github.com/glasskube/distr/commit/0e370370d308999f0511850f7cccd43e8872aaea))
* **deps:** update module github.com/jackc/pgx/v5 to v5.7.6 ([#1176](https://github.com/glasskube/distr/issues/1176)) ([7add766](https://github.com/glasskube/distr/commit/7add766d16dd391b88c8cfda293fb67a3d0d48f7))
* **deps:** update module github.com/mark3labs/mcp-go to v0.41.1 ([#1231](https://github.com/glasskube/distr/issues/1231)) ([e39d252](https://github.com/glasskube/distr/commit/e39d2529ff7dd064c49d4fbea1d28f50f221a60e))
* **deps:** update module github.com/onsi/gomega to v1.38.2 ([#1213](https://github.com/glasskube/distr/issues/1213)) ([f72250f](https://github.com/glasskube/distr/commit/f72250fd33bf13c8eadc9f9df7392023eb146150))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.135.0 ([#1214](https://github.com/glasskube/distr/issues/1214)) ([0affe45](https://github.com/glasskube/distr/commit/0affe45083dc77abb04e024c6b1cf51c7edc4985))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.137.0 ([#1236](https://github.com/glasskube/distr/issues/1236)) ([aa9bd55](https://github.com/glasskube/distr/commit/aa9bd557f395708dd4d7cb461a12b8b8831110fa))
* **deps:** update module github.com/spf13/cobra to v1.10.1 ([#1215](https://github.com/glasskube/distr/issues/1215)) ([c1133a6](https://github.com/glasskube/distr/commit/c1133a6cec0a1341166a49fde5f7c5d61944e4b6))
* **deps:** update module github.com/wneessen/go-mail to v0.7.0 ([#1216](https://github.com/glasskube/distr/issues/1216)) ([8b0996c](https://github.com/glasskube/distr/commit/8b0996c4aa3ccdff950b17f9809fe6c75eaaf856))
* **deps:** update module github.com/wneessen/go-mail to v0.7.1 [security] ([#1239](https://github.com/glasskube/distr/issues/1239)) ([44c2dd6](https://github.com/glasskube/distr/commit/44c2dd6fab2bec5e8da0b98a585f01d85d843751))
* **deps:** update module github.com/wneessen/go-mail to v0.7.2 ([#1255](https://github.com/glasskube/distr/issues/1255)) ([2389b02](https://github.com/glasskube/distr/commit/2389b0217dab03b22bb7af9092d61b3a7b90ba4f))
* **deps:** update module go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc to v1.38.0 ([#1222](https://github.com/glasskube/distr/issues/1222)) ([42e29e0](https://github.com/glasskube/distr/commit/42e29e0899291d6554c9e2b220a98eb59f62bc89))
* **deps:** update module golang.org/x/oauth2 to v0.31.0 ([#1223](https://github.com/glasskube/distr/issues/1223)) ([3ced494](https://github.com/glasskube/distr/commit/3ced494adc3ac0f608ce01afc580db7303356f9e))
* **deps:** update module helm.sh/helm/v3 to v3.18.6 ([#1177](https://github.com/glasskube/distr/issues/1177)) ([2d1d2fd](https://github.com/glasskube/distr/commit/2d1d2fdf8e7f2415e8be1a151b1d0ba47679b587))
* **deps:** update module helm.sh/helm/v3 to v3.19.0 ([#1225](https://github.com/glasskube/distr/issues/1225)) ([b821ddb](https://github.com/glasskube/distr/commit/b821ddb6ad242ab185f725fc35cbc52ed396df3e))
* **deps:** update opentelemetry-go-contrib monorepo to v0.63.0 ([#1228](https://github.com/glasskube/distr/issues/1228)) ([12abe1f](https://github.com/glasskube/distr/commit/12abe1f1260a0eaef0a2ab5d26715e636d0a89d1))


### Other

* **deps:** update actions/checkout action to v5 ([#1188](https://github.com/glasskube/distr/issues/1188)) ([a9150fa](https://github.com/glasskube/distr/commit/a9150faebf5d48955b045bc683338ca0210531ce))
* **deps:** update actions/download-artifact action to v5 ([#1189](https://github.com/glasskube/distr/issues/1189)) ([7066589](https://github.com/glasskube/distr/commit/7066589c1323baef3231f170cc534c8bbd07dd53))
* **deps:** update actions/setup-go action to v6 ([#1190](https://github.com/glasskube/distr/issues/1190)) ([bdcd765](https://github.com/glasskube/distr/commit/bdcd76565fe2c77f11200d1d456113a2ed07a4e6))
* **deps:** update actions/setup-node action to v5 ([#1191](https://github.com/glasskube/distr/issues/1191)) ([1c198d6](https://github.com/glasskube/distr/commit/1c198d6bad664e5c3f642795ef031fd7c228e5ae))
* **deps:** update amannn/action-semantic-pull-request action to v6 ([#1192](https://github.com/glasskube/distr/issues/1192)) ([6587b10](https://github.com/glasskube/distr/commit/6587b10699df0ff0129a6d03b771de197bc134b4))
* **deps:** update anchore/sbom-action action to v0.20.6 ([#1209](https://github.com/glasskube/distr/issues/1209)) ([b31164f](https://github.com/glasskube/distr/commit/b31164ff1de68ec744294b57e85908c6df08dab9))
* **deps:** update anchore/sbom-action action to v0.20.7 ([#1258](https://github.com/glasskube/distr/issues/1258)) ([5f87006](https://github.com/glasskube/distr/commit/5f87006f016bb993f83c1f248b427e5c9bbc2071))
* **deps:** update anchore/sbom-action action to v0.20.8 ([#1264](https://github.com/glasskube/distr/issues/1264)) ([d82fe14](https://github.com/glasskube/distr/commit/d82fe147d744e35bd8ab1bcc5ac51f696f0eab55))
* **deps:** update angular monorepo to v20.3.5 ([#1237](https://github.com/glasskube/distr/issues/1237)) ([1fd632b](https://github.com/glasskube/distr/commit/1fd632b948b31484b2d1764bc9dc7bb8dd6353da))
* **deps:** update angular monorepo to v20.3.6 ([#1267](https://github.com/glasskube/distr/issues/1267)) ([a57c9fa](https://github.com/glasskube/distr/commit/a57c9fae633d6e679afa3da31a3160c9f727aa00))
* **deps:** update angular-cli monorepo to v20.3.1 ([#1193](https://github.com/glasskube/distr/issues/1193)) ([31fc454](https://github.com/glasskube/distr/commit/31fc4540169d4c8fe4dd5979495cd55ea4ee3fb9))
* **deps:** update angular-cli monorepo to v20.3.2 ([#1224](https://github.com/glasskube/distr/issues/1224)) ([8de47ab](https://github.com/glasskube/distr/commit/8de47ab605004ea38cd6d65666f624a005f5d3ed))
* **deps:** update angular-cli monorepo to v20.3.6 ([#1259](https://github.com/glasskube/distr/issues/1259)) ([6eb2037](https://github.com/glasskube/distr/commit/6eb2037cc56e58543ea9c255e35254d36359989a))
* **deps:** update axllent/mailpit docker tag to v1.27.10 ([#1251](https://github.com/glasskube/distr/issues/1251)) ([a62176a](https://github.com/glasskube/distr/commit/a62176ac30f0d09962efb815083e144d83537bee))
* **deps:** update axllent/mailpit docker tag to v1.27.8 ([#1204](https://github.com/glasskube/distr/issues/1204)) ([64457b8](https://github.com/glasskube/distr/commit/64457b80117dee9115c45c18ec1996b0a50cc4de))
* **deps:** update axllent/mailpit docker tag to v1.27.9 ([#1246](https://github.com/glasskube/distr/issues/1246)) ([7fa8973](https://github.com/glasskube/distr/commit/7fa8973cacdd55b0b280bd4f55f962b7fe8d889a))
* **deps:** update dependency @angular/cdk to v20.2.9 ([#1247](https://github.com/glasskube/distr/issues/1247)) ([075bb1d](https://github.com/glasskube/distr/commit/075bb1d9f1d8377b1910ee02e6107661187b116e))
* **deps:** update dependency @sentry/cli to v2.53.0 ([#1207](https://github.com/glasskube/distr/issues/1207)) ([fa8c9d9](https://github.com/glasskube/distr/commit/fa8c9d9016db96387d09d4efed00f33cfaca7884))
* **deps:** update dependency @types/jasmine to v5.1.11 ([#1260](https://github.com/glasskube/distr/issues/1260)) ([e403661](https://github.com/glasskube/distr/commit/e4036618002e6a505c3614d4b21677990ccc21c8))
* **deps:** update dependency go to v1.25.2 ([#1248](https://github.com/glasskube/distr/issues/1248)) ([2d402c3](https://github.com/glasskube/distr/commit/2d402c3b30f49a60db927e38af6d7da2df33f4f6))
* **deps:** update dependency jasmine-core to ~5.10.0 ([#1180](https://github.com/glasskube/distr/issues/1180)) ([a788d99](https://github.com/glasskube/distr/commit/a788d99f61a6b0d70552e675afed7bdbbbd4dae3))
* **deps:** update dependency jasmine-core to ~5.12.0 ([#1256](https://github.com/glasskube/distr/issues/1256)) ([ec46e73](https://github.com/glasskube/distr/commit/ec46e73ba88be10c15ae0a3e4bf9941454a99a99))
* **deps:** update dependency semver to v7.7.3 ([#1249](https://github.com/glasskube/distr/issues/1249)) ([1c5d9f4](https://github.com/glasskube/distr/commit/1c5d9f4e0465cf57056ac6091b740927a42820d2))
* **deps:** update dependency typedoc to v0.28.13 ([#1205](https://github.com/glasskube/distr/issues/1205)) ([0ac1823](https://github.com/glasskube/distr/commit/0ac18236bd59c8cef99d477643f5ab17b45009ab))
* **deps:** update dependency typedoc-plugin-markdown to v4.8.1 ([#1181](https://github.com/glasskube/distr/issues/1181)) ([664a496](https://github.com/glasskube/distr/commit/664a4965eb30d6d1611ba6c754385744eda9071f))
* **deps:** update dependency typedoc-plugin-markdown to v4.9.0 ([#1232](https://github.com/glasskube/distr/issues/1232)) ([f902bee](https://github.com/glasskube/distr/commit/f902beee1c26d5f4414fba56401cbe7372de4512))
* **deps:** update dependency typescript to ~5.9.0 ([#1182](https://github.com/glasskube/distr/issues/1182)) ([67c1899](https://github.com/glasskube/distr/commit/67c1899f4d0df309e2e926be3724922d6192c164))
* **deps:** update dependency typescript to v5.9.2 ([#1183](https://github.com/glasskube/distr/issues/1183)) ([139e24d](https://github.com/glasskube/distr/commit/139e24d6d395834563625037dcde4cd25351d73b))
* **deps:** update dependency typescript to v5.9.3 ([#1252](https://github.com/glasskube/distr/issues/1252)) ([8135e18](https://github.com/glasskube/distr/commit/8135e18923a837cfe6291dfa72d77282c31947d5))
* **deps:** update docker/login-action action to v3.5.0 ([#1184](https://github.com/glasskube/distr/issues/1184)) ([9b51c2e](https://github.com/glasskube/distr/commit/9b51c2e20857ce7e4639f6103c29b5e93353dfcd))
* **deps:** update docker/login-action action to v3.6.0 ([#1257](https://github.com/glasskube/distr/issues/1257)) ([fd4fae8](https://github.com/glasskube/distr/commit/fd4fae884fad25ec25832902a40c9c7e408812a2))
* **deps:** update docker/metadata-action action to v5.8.0 ([#1185](https://github.com/glasskube/distr/issues/1185)) ([3ba4108](https://github.com/glasskube/distr/commit/3ba4108aa2e87b6e571f2517d0d6f7796f49e2ed))
* **deps:** update googleapis/release-please-action action to v4.3.0 ([#1186](https://github.com/glasskube/distr/issues/1186)) ([ed9c89f](https://github.com/glasskube/distr/commit/ed9c89f2f159d9bb0804e0387e55b1404eb07e79))
* **deps:** update sigstore/cosign-installer action to v3.10.0 ([#1199](https://github.com/glasskube/distr/issues/1199)) ([c314816](https://github.com/glasskube/distr/commit/c3148168f16b8c5929297fde3536b3031f377bfd))
* **deps:** update sigstore/cosign-installer action to v3.10.1 ([#1268](https://github.com/glasskube/distr/issues/1268)) ([c046ddf](https://github.com/glasskube/distr/commit/c046ddf54de0f34f68404da846793639a9e8fe20))
* **deps:** update tailwindcss monorepo to v4.1.14 ([#1253](https://github.com/glasskube/distr/issues/1253)) ([63134ad](https://github.com/glasskube/distr/commit/63134adcf01ba15c9c3e98d5fe8b4a6a01b188de))

## [1.13.0](https://github.com/glasskube/distr/compare/1.12.5...1.13.0) (2025-09-11)


### Features

* **registry:** add GCS compatibility for registry storage ([#1158](https://github.com/glasskube/distr/issues/1158)) ([4588ef0](https://github.com/glasskube/distr/commit/4588ef01b277d52e7133e6ebbe25830e70b893df))


### Bug Fixes

* **deps:** update angular monorepo to v20.2.4 ([#1144](https://github.com/glasskube/distr/issues/1144)) ([e6636f6](https://github.com/glasskube/distr/commit/e6636f61aa1810e19056bb2055d680cff4ac3430))
* **deps:** update aws-sdk-go-v2 monorepo ([#1170](https://github.com/glasskube/distr/issues/1170)) ([d3d269e](https://github.com/glasskube/distr/commit/d3d269e9b2390a65567a4645c7a6b93e35cea09b))
* **deps:** update codemirror ([#1169](https://github.com/glasskube/distr/issues/1169)) ([397ebac](https://github.com/glasskube/distr/commit/397ebacb7cd9aca62606d7caf9684e8fb13985b1))
* **deps:** update dependency @angular/cdk to v20.2.2 ([#1145](https://github.com/glasskube/distr/issues/1145)) ([066dd52](https://github.com/glasskube/distr/commit/066dd52f8b6f1bd1f57f104504c171bd5051eb77))
* **deps:** update dependency @sentry/angular to v9.46.0 ([#1134](https://github.com/glasskube/distr/issues/1134)) ([90037a2](https://github.com/glasskube/distr/commit/90037a2612bb4e49d89967cf59fd3fc9d09395ee))
* **deps:** update dependency dayjs to v1.11.18 ([#1171](https://github.com/glasskube/distr/issues/1171)) ([4403648](https://github.com/glasskube/distr/commit/440364847a867a972e5b04408bf4df4b4ebfa09a))
* **deps:** update dependency posthog-js to v1.262.0 ([#1135](https://github.com/glasskube/distr/issues/1135)) ([8c3bdb0](https://github.com/glasskube/distr/commit/8c3bdb0d7cea345bc9358dbd3b98f3d05a44e966))
* **deps:** update github.com/jackc/pgerrcode digest to afb5586 ([#1160](https://github.com/glasskube/distr/issues/1160)) ([6e12edb](https://github.com/glasskube/distr/commit/6e12edbadeec23987ae90284bdf25d3fe0522ae9))
* **deps:** update module github.com/docker/cli to v28.3.2+incompatible ([#1146](https://github.com/glasskube/distr/issues/1146)) ([2dc8d10](https://github.com/glasskube/distr/commit/2dc8d10547456d4efdc043b6a72eae9e6b2483e9))
* **deps:** update module github.com/docker/compose/v2 to v2.38.2 ([#1138](https://github.com/glasskube/distr/issues/1138)) ([a6e18e9](https://github.com/glasskube/distr/commit/a6e18e96cadaca49b0d7463577df146cc610d6e5))
* **deps:** update module github.com/docker/docker to v28.3.2+incompatible ([#1147](https://github.com/glasskube/distr/issues/1147)) ([3cf1f60](https://github.com/glasskube/distr/commit/3cf1f60abab292eb0294b4c699e6b6775d80d3af))
* **deps:** update module github.com/docker/docker to v28.3.3+incompatible [security] ([#1151](https://github.com/glasskube/distr/issues/1151)) ([eb29fd7](https://github.com/glasskube/distr/commit/eb29fd758e2caee215f50f64d85a90403af3e373))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.35.2 ([#1137](https://github.com/glasskube/distr/issues/1137)) ([150e7e3](https://github.com/glasskube/distr/commit/150e7e3f0d173e68ebaba983096a071cf7a64ce6))
* **deps:** update module github.com/mark3labs/mcp-go to v0.39.1 ([#1139](https://github.com/glasskube/distr/issues/1139)) ([13311e5](https://github.com/glasskube/distr/commit/13311e5266a233be0b099eb3385769450aca7efb))
* **deps:** update module golang.org/x/crypto to v0.42.0 ([#1148](https://github.com/glasskube/distr/issues/1148)) ([1fa961d](https://github.com/glasskube/distr/commit/1fa961df1537357d2ca5c584caeb62fa02e1b466))
* **deps:** update module helm.sh/helm/v3 to v3.18.4 [security] ([#1143](https://github.com/glasskube/distr/issues/1143)) ([039deee](https://github.com/glasskube/distr/commit/039deeeb422843fa281a663301596238e7c0e519))
* **deps:** update module helm.sh/helm/v3 to v3.18.5 [security] ([#1156](https://github.com/glasskube/distr/issues/1156)) ([10e214c](https://github.com/glasskube/distr/commit/10e214cd540302bab115102ffc5575dfc07ee6dd))


### Other

* **deps:** update amannn/action-semantic-pull-request digest to e32d7e6 ([#1159](https://github.com/glasskube/distr/issues/1159)) ([5981ebf](https://github.com/glasskube/distr/commit/5981ebf64787c0273c482f5aefdd42dfa72b5b4e))
* **deps:** update anchore/sbom-action action to v0.20.5 ([#1161](https://github.com/glasskube/distr/issues/1161)) ([4275901](https://github.com/glasskube/distr/commit/42759019051e2dc6fed643f8f7844385c6c62ddd))
* **deps:** update axllent/mailpit docker tag to v1.27.1 ([#1132](https://github.com/glasskube/distr/issues/1132)) ([3fb8a29](https://github.com/glasskube/distr/commit/3fb8a2958124a396fde088608650005d60f2dfac))
* **deps:** update axllent/mailpit docker tag to v1.27.7 ([#1162](https://github.com/glasskube/distr/issues/1162)) ([1a537db](https://github.com/glasskube/distr/commit/1a537dbd24cdd301e5bb61c1d77883867e775424))
* **deps:** update azure/setup-helm action to v4.3.1 ([#1163](https://github.com/glasskube/distr/issues/1163)) ([8dde0d7](https://github.com/glasskube/distr/commit/8dde0d70228dfcae9c3e24b896353a19f0967c16))
* **deps:** update dependency @types/jasmine to v5.1.9 ([#1164](https://github.com/glasskube/distr/issues/1164)) ([47fdfd1](https://github.com/glasskube/distr/commit/47fdfd1f2aff9118a77595e257288ac91debae81))
* **deps:** update dependency @types/semver to v7.7.1 ([#1165](https://github.com/glasskube/distr/issues/1165)) ([f3ee925](https://github.com/glasskube/distr/commit/f3ee925c8477d4579b172d0807961f48e7abb372))
* **deps:** update dependency go to v1.25.1 ([#1140](https://github.com/glasskube/distr/issues/1140)) ([5edd3f3](https://github.com/glasskube/distr/commit/5edd3f38df44338aa99904db07ab4d407d0d819b))
* **deps:** update dependency golangci-lint to v2.4.0 ([#1149](https://github.com/glasskube/distr/issues/1149)) ([ac820e2](https://github.com/glasskube/distr/commit/ac820e28a17d7607a45b083653945ff930d5bc08))
* **deps:** update dependency typedoc to v0.28.12 ([#1166](https://github.com/glasskube/distr/issues/1166)) ([56750e5](https://github.com/glasskube/distr/commit/56750e5d526ff1dd825cc1e235160c78b31cf502))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to e8a4044 ([#1152](https://github.com/glasskube/distr/issues/1152)) ([df9e1f5](https://github.com/glasskube/distr/commit/df9e1f5969e6bba94ccacf814950d3d6deeacf52))
* **deps:** update sigstore/cosign-installer action to v3.9.2 ([#1167](https://github.com/glasskube/distr/issues/1167)) ([54e585e](https://github.com/glasskube/distr/commit/54e585ea4c057cd7176c47b66c9ec902885b710b))
* **deps:** update tailwindcss monorepo to v4.1.13 ([#1168](https://github.com/glasskube/distr/issues/1168)) ([3fa86d8](https://github.com/glasskube/distr/commit/3fa86d822c7dd4e5eb9fb43790870d25e5c02537))

## [1.12.5](https://github.com/glasskube/distr/compare/1.12.4...1.12.5) (2025-07-04)


### Bug Fixes

* **mcp:** update deployment mcp tool ([#1130](https://github.com/glasskube/distr/issues/1130)) ([bbf90fe](https://github.com/glasskube/distr/commit/bbf90fe17f9ea9ebf025395de7a9896578d406b8))


### Other

* **ui:** show a progress dialog during undeploy operations ([#1108](https://github.com/glasskube/distr/issues/1108)) ([b14bda6](https://github.com/glasskube/distr/commit/b14bda6bae84cb758d28fee74ad4aae5a37deba4))

## [1.12.4](https://github.com/glasskube/distr/compare/1.12.3...1.12.4) (2025-07-03)


### Docs

* update deployment screenshot in README.md ([#1061](https://github.com/glasskube/distr/issues/1061)) ([8e9fb77](https://github.com/glasskube/distr/commit/8e9fb77e4e82a91176f534e9ceb6e003db9fdf92))

## [1.12.3](https://github.com/glasskube/distr/compare/1.12.2...1.12.3) (2025-07-03)


### Bug Fixes

* **deps:** update angular monorepo to v20.0.6 ([#1121](https://github.com/glasskube/distr/issues/1121)) ([4f7e2d6](https://github.com/glasskube/distr/commit/4f7e2d69d5f10f2b73ad573c59564e1e984ae03b))
* **deps:** update dependency @angular/cdk to v20.0.5 ([#1124](https://github.com/glasskube/distr/issues/1124)) ([723afee](https://github.com/glasskube/distr/commit/723afeef60847b603fccc86137d868c624508a4f))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.83.0 ([#1125](https://github.com/glasskube/distr/issues/1125)) ([237d5bf](https://github.com/glasskube/distr/commit/237d5bfbcc86ffef45c9bf7e88a49642dcaa02a0))
* **deps:** update module github.com/docker/cli to v28.3.1+incompatible ([#1126](https://github.com/glasskube/distr/issues/1126)) ([c5744ab](https://github.com/glasskube/distr/commit/c5744ab64ba1487f1bf7099deb9d8250c6da0585))
* **deps:** update module github.com/docker/docker to v28.3.1+incompatible ([#1127](https://github.com/glasskube/distr/issues/1127)) ([892eea8](https://github.com/glasskube/distr/commit/892eea832b14f51deca9fdd3fb72ed63ac0000d9))


### Other

* add ability to set the sentry environment ([#1104](https://github.com/glasskube/distr/issues/1104)) ([b6ae803](https://github.com/glasskube/distr/commit/b6ae80397c24a5445a1ba1c923f714a6d640a46c))
* **deps:** update anchore/sbom-action action to v0.20.2 ([#1123](https://github.com/glasskube/distr/issues/1123)) ([e07aca6](https://github.com/glasskube/distr/commit/e07aca60956ce00e5f86f76713115d912dbf97ba))
* **deps:** update angular-cli monorepo to v20.0.5 ([#1122](https://github.com/glasskube/distr/issues/1122)) ([a23b806](https://github.com/glasskube/distr/commit/a23b8064306bdf1850b92ad746ff5fe705f6b310))

## [1.12.2](https://github.com/glasskube/distr/compare/1.12.1...1.12.2) (2025-07-01)


### Bug Fixes

* **deps:** update angular monorepo to v20.0.4 ([#1077](https://github.com/glasskube/distr/issues/1077)) ([a5cd1df](https://github.com/glasskube/distr/commit/a5cd1dfec1c016e273597e66113947fbc329cc19))
* **deps:** update angular monorepo to v20.0.5 ([#1098](https://github.com/glasskube/distr/issues/1098)) ([4332cf4](https://github.com/glasskube/distr/commit/4332cf40dab65534d02ab9a3ff9b6300cb53e16f))
* **deps:** update aws-sdk-go-v2 monorepo ([#1102](https://github.com/glasskube/distr/issues/1102)) ([458114d](https://github.com/glasskube/distr/commit/458114dfa38bb170c344d7a535d4af222ec9e3fb))
* **deps:** update codemirror ([#1099](https://github.com/glasskube/distr/issues/1099)) ([09a6322](https://github.com/glasskube/distr/commit/09a6322be634694cf67f7cabd7c3fb46575c80bf))
* **deps:** update dependency @angular/cdk to v20.0.4 ([#1101](https://github.com/glasskube/distr/issues/1101)) ([230099d](https://github.com/glasskube/distr/commit/230099d27b445da89e48cacde248e2371320303a))
* **deps:** update kubernetes packages to v0.33.2 ([#1080](https://github.com/glasskube/distr/issues/1080)) ([bc799ba](https://github.com/glasskube/distr/commit/bc799baccc82e06dffcaf9c00a116db62844e832))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.81.0 ([#1078](https://github.com/glasskube/distr/issues/1078)) ([1f3cb28](https://github.com/glasskube/distr/commit/1f3cb28f0e41d405263fda7901fc9e2cc3da6902))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.6.5 ([#1085](https://github.com/glasskube/distr/issues/1085)) ([6e41d3a](https://github.com/glasskube/distr/commit/6e41d3aaae4cf7d0854ceedc84a8df95a14b25c0))
* **deps:** update module github.com/docker/cli to v28.3.0+incompatible ([#1092](https://github.com/glasskube/distr/issues/1092)) ([c055d1b](https://github.com/glasskube/distr/commit/c055d1b36a282f7ae12ac1efb7e99e8fa2ca1b84))
* **deps:** update module github.com/docker/compose/v2 to v2.37.2 ([#1084](https://github.com/glasskube/distr/issues/1084)) ([ffe34f4](https://github.com/glasskube/distr/commit/ffe34f45ab514e782d467e1367641a57be067cf5))
* **deps:** update module github.com/docker/compose/v2 to v2.37.3 ([#1091](https://github.com/glasskube/distr/issues/1091)) ([40b40e4](https://github.com/glasskube/distr/commit/40b40e4234db0498aa476821f4e35bcd961fd043))
* **deps:** update module github.com/docker/compose/v2 to v2.38.1 ([#1116](https://github.com/glasskube/distr/issues/1116)) ([72c0e9c](https://github.com/glasskube/distr/commit/72c0e9c462f618e5a45631a934fc06e4ed800171))
* **deps:** update module github.com/docker/docker to v28.3.0+incompatible ([#1093](https://github.com/glasskube/distr/issues/1093)) ([c3ddc07](https://github.com/glasskube/distr/commit/c3ddc07747e6b0e3b1dcb2ee47b7bb92c297c8f3))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.34.0 ([#1089](https://github.com/glasskube/distr/issues/1089)) ([5c1aaab](https://github.com/glasskube/distr/commit/5c1aaab5d57f44df43326bf3cd1989cd104f06ab))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.2 [security] ([#1083](https://github.com/glasskube/distr/issues/1083)) ([90ed87d](https://github.com/glasskube/distr/commit/90ed87d7dbdc9297413a851f02e72d205d728806))
* **deps:** update module github.com/masterminds/semver/v3 to v3.4.0 ([#1103](https://github.com/glasskube/distr/issues/1103)) ([f7b9096](https://github.com/glasskube/distr/commit/f7b9096243e42149fa4cddc4482dcfed9b8f2040))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.129.0 ([#1115](https://github.com/glasskube/distr/issues/1115)) ([077a063](https://github.com/glasskube/distr/commit/077a063b30b1ac3e6ffd7ab3922860c383958392))
* **deps:** update opentelemetry-go monorepo to v1.37.0 ([#1094](https://github.com/glasskube/distr/issues/1094)) ([19aa86d](https://github.com/glasskube/distr/commit/19aa86d9ac480ab1c1874a10511b7a790b543425))
* **deps:** update opentelemetry-go-contrib monorepo to v0.62.0 ([#1107](https://github.com/glasskube/distr/issues/1107)) ([915d7b1](https://github.com/glasskube/distr/commit/915d7b17109bc0bb4f0a497105533b63c4c6cf7e))
* **registry:** prevent occasional S3 NoSuchKey error ([#1118](https://github.com/glasskube/distr/issues/1118)) ([997e5e9](https://github.com/glasskube/distr/commit/997e5e9718e925c355ccbf529dd9a18b42b27ca9))
* **security:** add workflow read permission to validate migrations job ([#1090](https://github.com/glasskube/distr/issues/1090)) ([9c278a1](https://github.com/glasskube/distr/commit/9c278a106ac7732d744df75e28dc90c6a4b4c651))


### Other

* **deps:** update angular-cli monorepo to v20.0.4 ([#1100](https://github.com/glasskube/distr/issues/1100)) ([0ef9ad1](https://github.com/glasskube/distr/commit/0ef9ad1ea8d61ec62d3d76488a060d9f190f24dd))
* **deps:** update axllent/mailpit docker tag to v1.26.2 ([#1086](https://github.com/glasskube/distr/issues/1086)) ([9c50879](https://github.com/glasskube/distr/commit/9c508790403e7564d1f440e6d41f1bdaa37fdbe2))
* **deps:** update axllent/mailpit docker tag to v1.27.0 ([#1105](https://github.com/glasskube/distr/issues/1105)) ([2e74ae6](https://github.com/glasskube/distr/commit/2e74ae682a7682d9296dd6ebf4f25b3eadab9cdb))
* **deps:** update dependency golangci-lint to v2.2.1 ([#1106](https://github.com/glasskube/distr/issues/1106)) ([e017579](https://github.com/glasskube/distr/commit/e0175790a44295c0db74973af8e24eb2a2c2dc5e))
* **deps:** update dependency prettier to v3.6.2 ([#1087](https://github.com/glasskube/distr/issues/1087)) ([95ea1e2](https://github.com/glasskube/distr/commit/95ea1e2cffab282f268dd09fda9413650dca64d9))
* **deps:** update dependency typedoc to v0.28.7 ([#1096](https://github.com/glasskube/distr/issues/1096)) ([57d7200](https://github.com/glasskube/distr/commit/57d72008fda65cd659e2dae57cbbae20ee2a69dc))
* **deps:** update dependency typedoc-plugin-markdown to v4.7.0 ([#1081](https://github.com/glasskube/distr/issues/1081)) ([fbbed23](https://github.com/glasskube/distr/commit/fbbed230dc467d8d457490dc6a960f38db1b3d9a))
* **deps:** update tailwindcss monorepo to v4.1.11 ([#1097](https://github.com/glasskube/distr/issues/1097)) ([42f27a3](https://github.com/glasskube/distr/commit/42f27a3eba78b6535c5f175f9a509681130549f1))

## [1.12.1](https://github.com/glasskube/distr/compare/1.12.0...1.12.1) (2025-06-18)


### Bug Fixes

* **backend:** change https detection for OIDC redirect URI ([#1072](https://github.com/glasskube/distr/issues/1072)) ([5567d32](https://github.com/glasskube/distr/commit/5567d32d6df893172d0697188118c4e3f8099e8c))


### Other

* **deps:** update angular-cli monorepo to v20.0.3 ([#1071](https://github.com/glasskube/distr/issues/1071)) ([70019df](https://github.com/glasskube/distr/commit/70019dff4b0cbc2b6c87a25aeaf3ea68703049a9))
* **deps:** update docker/setup-buildx-action action to v3.11.1 ([#1075](https://github.com/glasskube/distr/issues/1075)) ([b5b5a11](https://github.com/glasskube/distr/commit/b5b5a11442a1ad1722632cb8ae1283fa70298c2e))
* **ui:** make application details accessible in modal ([#1073](https://github.com/glasskube/distr/issues/1073)) ([976396c](https://github.com/glasskube/distr/commit/976396c9b1648f83b71d37ddabf3a1a8dc44514d))

## [1.12.0](https://github.com/glasskube/distr/compare/1.11.2...1.12.0) (2025-06-18)


### Features

* add force restarting workload for a deployment ([#1065](https://github.com/glasskube/distr/issues/1065)) ([2668751](https://github.com/glasskube/distr/commit/2668751b85547d906ff737c3c1bf7674ee07b7e2))
* **mcp:** add initial version of the Distr MCP server ([#1016](https://github.com/glasskube/distr/issues/1016)) ([a1b61ad](https://github.com/glasskube/distr/commit/a1b61ad44cbcc93d91c6c806f36a9a79d84b54de))
* store artifact manifest in database and show more specific usage instructions ([#1006](https://github.com/glasskube/distr/issues/1006)) ([e9e7412](https://github.com/glasskube/distr/commit/e9e7412bbe44d71cb5b5b5fcf89ac41c0484d0ff))
* support login with GitHub, Google and Microsoft via OIDC ([#1039](https://github.com/glasskube/distr/issues/1039)) ([8bc5f81](https://github.com/glasskube/distr/commit/8bc5f81fa8c0e92fabd2c539a6770d06eb9c7a62))


### Bug Fixes

* **agent:** get logs with namespace to allow logs collection with namespace-scoped agent ([#1044](https://github.com/glasskube/distr/issues/1044)) ([536cef8](https://github.com/glasskube/distr/commit/536cef8733579143b6e4dde17f11771d91ed6980))
* **backend:** add add context cancellation on SIGTERM and timeout flag for cleanup command ([#1060](https://github.com/glasskube/distr/issues/1060)) ([c941b88](https://github.com/glasskube/distr/commit/c941b88127645c35b1f07a7acf5e85746ea979db))
* **deps:** update angular monorepo to v20.0.1 ([#1012](https://github.com/glasskube/distr/issues/1012)) ([5f11add](https://github.com/glasskube/distr/commit/5f11addc6ca8deafcce96d80018d418a1d658e16))
* **deps:** update angular monorepo to v20.0.2 ([#1018](https://github.com/glasskube/distr/issues/1018)) ([c466b35](https://github.com/glasskube/distr/commit/c466b35d34d38af741b3448a54610e5958a2bb20))
* **deps:** update angular monorepo to v20.0.3 ([#1037](https://github.com/glasskube/distr/issues/1037)) ([d956f6f](https://github.com/glasskube/distr/commit/d956f6f408b1234108ea95f87b9210ebc3ed49b0))
* **deps:** update aws-sdk-go-v2 monorepo ([#1019](https://github.com/glasskube/distr/issues/1019)) ([cdb8d14](https://github.com/glasskube/distr/commit/cdb8d14886699050f2cbc88cfe79d8719b2aca27))
* **deps:** update aws-sdk-go-v2 monorepo ([#1035](https://github.com/glasskube/distr/issues/1035)) ([5bb799c](https://github.com/glasskube/distr/commit/5bb799cb64433102a8fc725a1733076cd1e777ef))
* **deps:** update aws-sdk-go-v2 monorepo ([#1070](https://github.com/glasskube/distr/issues/1070)) ([1d27a5c](https://github.com/glasskube/distr/commit/1d27a5c95f2c7d258d270002849446e44e2a0337))
* **deps:** update dependency @angular/cdk to v20.0.2 ([#1013](https://github.com/glasskube/distr/issues/1013)) ([add1c9f](https://github.com/glasskube/distr/commit/add1c9fa36187b8c9d46d3db69b545bb88cf2172))
* **deps:** update dependency @angular/cdk to v20.0.3 ([#1036](https://github.com/glasskube/distr/issues/1036)) ([4467ee7](https://github.com/glasskube/distr/commit/4467ee775155251838327872a7b5bd3e8ea9d5c3))
* **deps:** update dependency @codemirror/view to v6.37.2 ([#1050](https://github.com/glasskube/distr/issues/1050)) ([65e512a](https://github.com/glasskube/distr/commit/65e512a1a15d26d3d1a6c9fdae4dae139fca31c7))
* **deps:** update dependency @fontsource/inter to v5.2.6 ([#1024](https://github.com/glasskube/distr/issues/1024)) ([e828721](https://github.com/glasskube/distr/commit/e8287217cdb4f05293b9716b95a15c175d0c9859))
* **deps:** update dependency @sentry/angular to v9.29.0 ([#1058](https://github.com/glasskube/distr/issues/1058)) ([65a0e80](https://github.com/glasskube/distr/commit/65a0e803c87ccb758e1d134953b912a10f09e8fc))
* **deps:** update dependency posthog-js to v1.249.5 ([#1025](https://github.com/glasskube/distr/issues/1025)) ([96607e9](https://github.com/glasskube/distr/commit/96607e92eb08bed0166807e6874a1422c7822002))
* **deps:** update dependency posthog-js to v1.252.1 ([#1059](https://github.com/glasskube/distr/issues/1059)) ([45263e4](https://github.com/glasskube/distr/commit/45263e48d8c99752fe45e3d178808a0bd4ebbaf0))
* **deps:** update module github.com/containers/image/v5 to v5.35.0 ([#1032](https://github.com/glasskube/distr/issues/1032)) ([4242e72](https://github.com/glasskube/distr/commit/4242e728c378fbf6170d83122434be2872ccd1b3))
* **deps:** update module github.com/docker/compose/v2 to v2.37.0 ([#1022](https://github.com/glasskube/distr/issues/1022)) ([dcf04bf](https://github.com/glasskube/distr/commit/dcf04bf9721da81ce9fe0e44c79879ab47b99a63))
* **deps:** update module github.com/docker/compose/v2 to v2.37.1 ([#1043](https://github.com/glasskube/distr/issues/1043)) ([857aaa2](https://github.com/glasskube/distr/commit/857aaa2ff95756f7d4104389e5ca3499ce9504fe))
* **deps:** update module github.com/mark3labs/mcp-go to v0.32.0 ([#1053](https://github.com/glasskube/distr/issues/1053)) ([087f21a](https://github.com/glasskube/distr/commit/087f21a3fdf91308ba1d4e71e59c00c7e9675d68))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.128.0 ([#1026](https://github.com/glasskube/distr/issues/1026)) ([59dea18](https://github.com/glasskube/distr/commit/59dea182dcd637508f0be0fb9be6bb4e46214fa7))
* **deps:** update module golang.org/x/crypto to v0.39.0 ([#1023](https://github.com/glasskube/distr/issues/1023)) ([3ece41f](https://github.com/glasskube/distr/commit/3ece41f566d088308584f40701340eb012e5ae79))
* **deps:** update module helm.sh/helm/v3 to v3.18.3 ([#1066](https://github.com/glasskube/distr/issues/1066)) ([49dad05](https://github.com/glasskube/distr/commit/49dad051614dd658929d7b7e611f9a63a7ee2d76))
* **ui:** navbar falls back to showing user info from token ([#1062](https://github.com/glasskube/distr/issues/1062)) ([e6910e7](https://github.com/glasskube/distr/commit/e6910e76d3f37200a0efc8dc9adb9461615aaa80))


### Other

* add enabling logs on deployment creation ([#1008](https://github.com/glasskube/distr/issues/1008)) ([b3d16cc](https://github.com/glasskube/distr/commit/b3d16cc67eaaba58be7d470e3ffb3f611c1fad39))
* **backend:** add separate sampler config for agent API, registry ([#1038](https://github.com/glasskube/distr/issues/1038)) ([f14516e](https://github.com/glasskube/distr/commit/f14516e7543ddb9501d595b17b66ab475d7e3af9))
* **backend:** add tracing for commands, jobs ([#1000](https://github.com/glasskube/distr/issues/1000)) ([26d09f5](https://github.com/glasskube/distr/commit/26d09f5775f5f3e3ac36cc06c2389ad282ffdb64))
* **backend:** fix typo in previously added `ValidateDeploymentLogRecords` ([#1055](https://github.com/glasskube/distr/issues/1055)) ([03a69b6](https://github.com/glasskube/distr/commit/03a69b6828046a72c1d0b3418c48e8cc1ad28958))
* **backend:** improve OTEL trace naming ([#1069](https://github.com/glasskube/distr/issues/1069)) ([34e74d9](https://github.com/glasskube/distr/commit/34e74d962f1e4f9610f38be630f3c2c5636f724c))
* **deps:** update angular-cli monorepo to v20.0.1 ([#1015](https://github.com/glasskube/distr/issues/1015)) ([a96235a](https://github.com/glasskube/distr/commit/a96235a861ac83c4282053e2c1c7f14db9b20d1d))
* **deps:** update angular-cli monorepo to v20.0.2 ([#1040](https://github.com/glasskube/distr/issues/1040)) ([355b24c](https://github.com/glasskube/distr/commit/355b24ca92987cc60ee8cbe9816e3ab32cba3387))
* **deps:** update axllent/mailpit docker tag to v1.26.0 ([#1020](https://github.com/glasskube/distr/issues/1020)) ([f1bb779](https://github.com/glasskube/distr/commit/f1bb779e0ced10df7dbecc23c19ad75117017af2))
* **deps:** update axllent/mailpit docker tag to v1.26.1 ([#1056](https://github.com/glasskube/distr/issues/1056)) ([4ad1dff](https://github.com/glasskube/distr/commit/4ad1dffb15bae706e12402ee28854872f50c9a06))
* **deps:** update dependency go to v1.24.4 ([#1017](https://github.com/glasskube/distr/issues/1017)) ([94ed156](https://github.com/glasskube/distr/commit/94ed156e3078e46c58b94b9b47a885446dc2ff1f))
* **deps:** update dependency jasmine-core to ~5.8.0 ([#1021](https://github.com/glasskube/distr/issues/1021)) ([ee3e12c](https://github.com/glasskube/distr/commit/ee3e12cd13549b6a1d3444143612d16746334e8c))
* **deps:** update dependency postcss to v8.5.5 ([#1041](https://github.com/glasskube/distr/issues/1041)) ([8407082](https://github.com/glasskube/distr/commit/8407082b256c23ebb2b27629722d6362c9d0aecd))
* **deps:** update dependency postcss to v8.5.6 ([#1064](https://github.com/glasskube/distr/issues/1064)) ([af85b11](https://github.com/glasskube/distr/commit/af85b1191e87fe4efc6e231dbc64e39f6c1e82a0))
* **deps:** update docker/setup-buildx-action action to v3.11.0 ([#1063](https://github.com/glasskube/distr/issues/1063)) ([17ba5eb](https://github.com/glasskube/distr/commit/17ba5eb044c9d00a90c8d8281ec5b09422666a12))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to 627d6c5 ([#1057](https://github.com/glasskube/distr/issues/1057)) ([116d6c3](https://github.com/glasskube/distr/commit/116d6c35a7941d321fb0dcb3a107a9a1e0e84ff9))
* **deps:** update ghcr.io/glasskube/distr docker tag to v1.11.3 ([#1047](https://github.com/glasskube/distr/issues/1047)) ([35ddf50](https://github.com/glasskube/distr/commit/35ddf5006512bb8f9c83e9939ccbd1fdc1a315e9))
* **deps:** update ghcr.io/glasskube/distr docker tag to v1.11.5 ([#1049](https://github.com/glasskube/distr/issues/1049)) ([4236fbd](https://github.com/glasskube/distr/commit/4236fbd63b19b90ffe99bb3dfa834b2a4a80d1ff))
* **deps:** update tailwindcss monorepo to v4.1.10 ([#1042](https://github.com/glasskube/distr/issues/1042)) ([20b799b](https://github.com/glasskube/distr/commit/20b799bccae1212abc7cc82d3af8c836f58ccd3d))
* **mcp:** add mcp build target ([#1068](https://github.com/glasskube/distr/issues/1068)) ([3fddf62](https://github.com/glasskube/distr/commit/3fddf62991dd1eabc55284f74a577a5c03139ef3))
* **mcp:** add missing tools ([#1067](https://github.com/glasskube/distr/issues/1067)) ([287f120](https://github.com/glasskube/distr/commit/287f120a12f679eae6e84b2ff638e24f81050f8d))
* **ui:** add text masking ([#1033](https://github.com/glasskube/distr/issues/1033)) ([6b5dcaa](https://github.com/glasskube/distr/commit/6b5dcaa2d8cba574906aba58bf18343f81505302))


### Docs

* add contributing guidelines ([#1011](https://github.com/glasskube/distr/issues/1011)) ([601ffc9](https://github.com/glasskube/distr/commit/601ffc9bfcf0a3e197dd46c3771567eb989e76f8))


### Performance

* **backend:** add index on `DeploymentLogRecord(resource)` ([#1046](https://github.com/glasskube/distr/issues/1046)) ([0c9466f](https://github.com/glasskube/distr/commit/0c9466ff614f24c972ffd75834cbeb7faea2a225))
* **backend:** optimize `GetDeploymentsForDeploymentTarget` db query with lateral join ([#1051](https://github.com/glasskube/distr/issues/1051)) ([2d1f2d4](https://github.com/glasskube/distr/commit/2d1f2d438ff126285f2f688b9765db2b98722c4e))
* **backend:** validate deployment log record creation with bespoke db query ([#1054](https://github.com/glasskube/distr/issues/1054)) ([a8d3970](https://github.com/glasskube/distr/commit/a8d39700181ddb1f1cb6695d7a0b45d4519c13f3))

## [1.11.2](https://github.com/glasskube/distr/compare/1.11.1...1.11.2) (2025-06-04)


### Bug Fixes

* **backend:** disable smtp noop check ([#1001](https://github.com/glasskube/distr/issues/1001)) ([48056b6](https://github.com/glasskube/distr/commit/48056b60f6fd996014850cebeb1ffd14dc97cab8))
* **backend:** send 400 response code for deployment release name conflict ([#1005](https://github.com/glasskube/distr/issues/1005)) ([8e512e9](https://github.com/glasskube/distr/commit/8e512e954770b1cae74f091d6ff9941ceee70ebb))
* **deps:** update angular monorepo to v19.2.14 ([#980](https://github.com/glasskube/distr/issues/980)) ([c52e75f](https://github.com/glasskube/distr/commit/c52e75f512b0d9a74605966861dd0f1b957f8666))
* **deps:** update dependency @angular/cdk to v19.2.18 ([#981](https://github.com/glasskube/distr/issues/981)) ([825117a](https://github.com/glasskube/distr/commit/825117ac88e3e44cbd11e8e5da1383823618c487))
* **deps:** update dependency @codemirror/language to v6.11.1 ([#1002](https://github.com/glasskube/distr/issues/1002)) ([43adea5](https://github.com/glasskube/distr/commit/43adea5e2ce458db8cf95e9b388e544b743baa34))
* **deps:** update dependency @codemirror/view to v6.37.1 ([#985](https://github.com/glasskube/distr/issues/985)) ([a613245](https://github.com/glasskube/distr/commit/a6132450d2b365e328dfe6fd57cb56f2c68f148b))
* **deps:** update dependency @fortawesome/angular-fontawesome to v2.0.1 ([#1003](https://github.com/glasskube/distr/issues/1003)) ([8f4aec4](https://github.com/glasskube/distr/commit/8f4aec4de3076d6194cc68a73123847cee2c07c7))
* **deps:** update dependency @sentry/angular to v9.24.0 ([#996](https://github.com/glasskube/distr/issues/996)) ([7ad5383](https://github.com/glasskube/distr/commit/7ad5383e75e074513848f8830211277fad22391f))
* **deps:** update dependency posthog-js to v1.249.0 ([#997](https://github.com/glasskube/distr/issues/997)) ([4c81796](https://github.com/glasskube/distr/commit/4c81796d0c40d4cf4cb06282cf0ddaec11ab6f5b))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.80.0 ([#987](https://github.com/glasskube/distr/issues/987)) ([1bb2266](https://github.com/glasskube/distr/commit/1bb226648518555ea9a9954946d97d632c08d1f8))
* **deps:** update module github.com/docker/cli to v28.2.2+incompatible ([#988](https://github.com/glasskube/distr/issues/988)) ([cc07ac1](https://github.com/glasskube/distr/commit/cc07ac1956b8d84077ff1a74b4172b368b02a39f))
* **deps:** update module helm.sh/helm/v3 to v3.18.1 ([#984](https://github.com/glasskube/distr/issues/984)) ([1aed99a](https://github.com/glasskube/distr/commit/1aed99a0facdbd691f154148b52b21da01489ab0))
* **deps:** update module helm.sh/helm/v3 to v3.18.2 ([#999](https://github.com/glasskube/distr/issues/999)) ([9874194](https://github.com/glasskube/distr/commit/9874194b1f8a51a656210567a3ad5ef56879c931))


### Other

* **backend:** improve deployment target endpoint visibility constraint ([#982](https://github.com/glasskube/distr/issues/982)) ([753386f](https://github.com/glasskube/distr/commit/753386f7737dc39f53f7ca309eed271327dfe618))
* **deps:** update angular-cli monorepo to v19.2.14 ([#986](https://github.com/glasskube/distr/issues/986)) ([522058a](https://github.com/glasskube/distr/commit/522058afd4a9d417a5d013ab9fa0a5eb0c1e7a3e))
* **deps:** update dependency typedoc-plugin-markdown to v4.6.4 ([#994](https://github.com/glasskube/distr/issues/994)) ([10557a7](https://github.com/glasskube/distr/commit/10557a72e6c498c50f01a492df33bb7b202b4b38))
* **deps:** upgrade Angular to v20 ([#998](https://github.com/glasskube/distr/issues/998)) ([8108fca](https://github.com/glasskube/distr/commit/8108fca162c5db35c139199cccd5dfd2f376a24a))
* remove deployment target geolocation ([#979](https://github.com/glasskube/distr/issues/979)) ([45062ec](https://github.com/glasskube/distr/commit/45062ecb19a24ff3ba6eb03b4d8b9db3e4107d66))
* **ui:** show an example OCI URL in application version form ([#1004](https://github.com/glasskube/distr/issues/1004)) ([4906c60](https://github.com/glasskube/distr/commit/4906c607c31eb55892e5cf2211cee1b4608088b9))


### Performance

* **backend:** optimize GetVersionsForArtifact database query ([#1009](https://github.com/glasskube/distr/issues/1009)) ([1f5e5ea](https://github.com/glasskube/distr/commit/1f5e5eaa8646101fec2de75fe3bcbf9cb28270f8))

## [1.11.1](https://github.com/glasskube/distr/compare/1.11.0...1.11.1) (2025-05-30)


### Other

* **deploy:** add cleanup jobs config in deploy/docker and helm chart ([#974](https://github.com/glasskube/distr/issues/974)) ([f6ed64d](https://github.com/glasskube/distr/commit/f6ed64d021344a5a5adb8cbb5e6e5dfae6d8186e))
* **deps:** update dependency postcss to v8.5.4 ([#976](https://github.com/glasskube/distr/issues/976)) ([bb8deb2](https://github.com/glasskube/distr/commit/bb8deb21fa15d1000650f98b6a8f95d7d9fc5bc2))
* **deps:** update tailwindcss monorepo to v4.1.8 ([#977](https://github.com/glasskube/distr/issues/977)) ([0c7a593](https://github.com/glasskube/distr/commit/0c7a593604f56740a6a0b42a91a3631ec40439a4))

## [1.11.0](https://github.com/glasskube/distr/compare/1.10.0...1.11.0) (2025-05-28)


### Features

* users can be part of multiple organizations ([#959](https://github.com/glasskube/distr/issues/959)) ([afea5fe](https://github.com/glasskube/distr/commit/afea5fef4eee1c574d2401fd235801e75843bdb7))


### Bug Fixes

* **deps:** update dependency posthog-js to v1.248.0 ([#972](https://github.com/glasskube/distr/issues/972)) ([5bec987](https://github.com/glasskube/distr/commit/5bec9876eb91238236654607ef01e276960d095c))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.127.0 ([#963](https://github.com/glasskube/distr/issues/963)) ([58c1de8](https://github.com/glasskube/distr/commit/58c1de87dae7000ba01c9e5bd82eabe6b13b6c53))


### Other

* **deps:** update dependency @sentry/cli to v2.46.0 ([#971](https://github.com/glasskube/distr/issues/971)) ([3ac6cf1](https://github.com/glasskube/distr/commit/3ac6cf19a53df45c02763e5ef04a54e7ee97c269))
* **deps:** update dependency typedoc to v0.28.5 ([#960](https://github.com/glasskube/distr/issues/960)) ([a03473f](https://github.com/glasskube/distr/commit/a03473f1bb94a4901958a51089178b1ada2fec66))
* **deps:** update docker/build-push-action action to v6.18.0 ([#970](https://github.com/glasskube/distr/issues/970)) ([6aece3c](https://github.com/glasskube/distr/commit/6aece3c1098f0ab51eef75f62cfe1ad14cd4ac3d))
* **ui:** agent update pending indicator for connected agents only ([#968](https://github.com/glasskube/distr/issues/968)) ([490288e](https://github.com/glasskube/distr/commit/490288e13d148f8623b7a13fab22e088afc22432))
* **ui:** clarify base/template value labels and add descriptions ([#958](https://github.com/glasskube/distr/issues/958)) ([ac90bd2](https://github.com/glasskube/distr/commit/ac90bd26e6e4be8f3079765a00eff0eecee3e338))
* **ui:** show "+ Deployment" and "Update" buttons unconditionally ([#949](https://github.com/glasskube/distr/issues/949)) ([e5fdd05](https://github.com/glasskube/distr/commit/e5fdd05afeca3b8fcfec60869dff01f30655a809))

## [1.10.0](https://github.com/glasskube/distr/compare/1.9.1...1.10.0) (2025-05-26)


### Features

* add job scheduling ([#935](https://github.com/glasskube/distr/issues/935)) ([f6ae73b](https://github.com/glasskube/distr/commit/f6ae73b1e5d7701cb4d82c0d59515be644cbe024))
* add support for container logs ([#881](https://github.com/glasskube/distr/issues/881)) ([a5a184c](https://github.com/glasskube/distr/commit/a5a184c062c966597fbec394aac2d8cba9e5e5ac))
* **ui:** add bulk actions for application versions ([#924](https://github.com/glasskube/distr/issues/924)) ([075ba08](https://github.com/glasskube/distr/commit/075ba0861c66f72d733839227268d848bf0949ea))


### Bug Fixes

* **agent:** fix kubernetes agent sending progressing updates during status check ([#948](https://github.com/glasskube/distr/issues/948)) ([8e153e2](https://github.com/glasskube/distr/commit/8e153e2c694d00d127839d818fb4d21e3f43c8ca))
* **backend:** lazily initialize jwt auth ([#941](https://github.com/glasskube/distr/issues/941)) ([923b3cf](https://github.com/glasskube/distr/commit/923b3cfe5b2d411eeddfba09198fe5c66ff3e3ae))
* **deps:** update angular monorepo to v19.2.12 ([#936](https://github.com/glasskube/distr/issues/936)) ([f49e406](https://github.com/glasskube/distr/commit/f49e406d1bfe21ea86ddfee09d917af44fbff1d0))
* **deps:** update angular monorepo to v19.2.13 ([#951](https://github.com/glasskube/distr/issues/951)) ([cdc34a7](https://github.com/glasskube/distr/commit/cdc34a7b4728a3e7b16bcb0f245f599d2721eb17))
* **deps:** update dependency @angular/cdk to v19.2.17 ([#937](https://github.com/glasskube/distr/issues/937)) ([a3e3510](https://github.com/glasskube/distr/commit/a3e3510b47cd86eda74a1bf5abaa49900b5e3778))
* **deps:** update dependency @sentry/angular to v9.22.0 ([#956](https://github.com/glasskube/distr/issues/956)) ([a75458d](https://github.com/glasskube/distr/commit/a75458d581995929775e9645c98ff3163d4ca851))
* **deps:** update dependency posthog-js to v1.246.0 ([#957](https://github.com/glasskube/distr/issues/957)) ([45cf0af](https://github.com/glasskube/distr/commit/45cf0af65a2138ac9031d32d0089d38a1c6540fa))
* **deps:** update dependency zone.js to v0.15.1 ([#946](https://github.com/glasskube/distr/issues/946)) ([48508c3](https://github.com/glasskube/distr/commit/48508c314b9558330d46190729de97f1f6523a45))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.79.4 ([#944](https://github.com/glasskube/distr/issues/944)) ([49f3fbe](https://github.com/glasskube/distr/commit/49f3fbea856c8d36e33505c1265296ef66c99295))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.6.4 ([#950](https://github.com/glasskube/distr/issues/950)) ([d7901e8](https://github.com/glasskube/distr/commit/d7901e8e94b4bafa05fb934e6fbde84fb746056c))
* **deps:** update module github.com/docker/compose/v2 to v2.36.1 ([#940](https://github.com/glasskube/distr/issues/940)) ([b13a977](https://github.com/glasskube/distr/commit/b13a9770f52f8a183998eee842d9695dfab9e730))
* **deps:** update module github.com/docker/compose/v2 to v2.36.2 ([#952](https://github.com/glasskube/distr/issues/952)) ([617e530](https://github.com/glasskube/distr/commit/617e53028659e32dd07a02ff3eaa03b8e6dedbb2))
* **deps:** update module github.com/google/go-containerregistry to v0.20.4 ([#932](https://github.com/glasskube/distr/issues/932)) ([df3a83b](https://github.com/glasskube/distr/commit/df3a83b7f78fe65f1f5da5e8aa3a290341341043))
* **deps:** update module github.com/google/go-containerregistry to v0.20.5 ([#945](https://github.com/glasskube/distr/issues/945)) ([b566b60](https://github.com/glasskube/distr/commit/b566b606be4954132bad4db6be5cae8ed7907fda))
* **deps:** update module helm.sh/helm/v3 to v3.18.0, oras-go to v2 ([#926](https://github.com/glasskube/distr/issues/926)) ([824c996](https://github.com/glasskube/distr/commit/824c996e8e3e24800f3a2b9662622edd1476598b))
* **deps:** update module k8s.io/kubectl to v0.33.1 ([#939](https://github.com/glasskube/distr/issues/939)) ([ae2023c](https://github.com/glasskube/distr/commit/ae2023cc58b7e2942557a632857e84ba3b34c3ba))
* **deps:** update opentelemetry-go monorepo to v1.36.0 ([#933](https://github.com/glasskube/distr/issues/933)) ([1ad2e70](https://github.com/glasskube/distr/commit/1ad2e70bdbe79c1be703983b6f474478b91d13c9))


### Other

* **backend:** turn the hub into a cobra app ([#931](https://github.com/glasskube/distr/issues/931)) ([cfe3f6f](https://github.com/glasskube/distr/commit/cfe3f6f56251884306397dcc0e4894303d9c5ef7))
* **deps:** bump github.com/containerd/containerd/v2 from 2.0.4 to 2.0.5 in the go_modules group across 1 directory ([#938](https://github.com/glasskube/distr/issues/938)) ([aa23d49](https://github.com/glasskube/distr/commit/aa23d4961298ded91d62c30d8b58940d1e262df2))
* **deps:** update angular-cli monorepo to v19.2.13 ([#934](https://github.com/glasskube/distr/issues/934)) ([84b978c](https://github.com/glasskube/distr/commit/84b978c8ccbfcdc8c4566f48fa9c3ccab2998719))
* **deps:** update axllent/mailpit docker tag to v1.25.1 ([#955](https://github.com/glasskube/distr/issues/955)) ([dad03e2](https://github.com/glasskube/distr/commit/dad03e2e34302ab4b48797aa087da84d1cbd4d24))
* **ui:** add robots.txt file to only index the login page ([#947](https://github.com/glasskube/distr/issues/947)) ([2de09e9](https://github.com/glasskube/distr/commit/2de09e9e8900bd8cb80fa3c2728919eba3a6c7fb))

## [1.9.1](https://github.com/glasskube/distr/compare/1.9.0...1.9.1) (2025-05-20)


### Bug Fixes

* **deps:** update dependency @sentry/angular to v9.19.0 ([#922](https://github.com/glasskube/distr/issues/922)) ([86d253a](https://github.com/glasskube/distr/commit/86d253a07801954a07d71790a8cf7e016be37ce9))
* **deps:** update dependency posthog-js to v1.242.3 ([#923](https://github.com/glasskube/distr/issues/923)) ([7c9db6f](https://github.com/glasskube/distr/commit/7c9db6f0432683f2907bf507020d50c525734482))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.6.3 ([#925](https://github.com/glasskube/distr/issues/925)) ([0e7695b](https://github.com/glasskube/distr/commit/0e7695b61dd2544813466cf611620568b0855a05))
* **deps:** update module github.com/jackc/pgx/v5 to v5.7.5 ([#918](https://github.com/glasskube/distr/issues/918)) ([6ebe47a](https://github.com/glasskube/distr/commit/6ebe47a5edeacb673a5b8f08b5a992dcc3ddce38))
* **ui:** deployments page handles archived version scenarios better ([#927](https://github.com/glasskube/distr/issues/927)) ([c966b22](https://github.com/glasskube/distr/commit/c966b22c756b0acb2ebf36228d104e7d7213f936))


### Other

* **backend:** use registry plain http for development ([#928](https://github.com/glasskube/distr/issues/928)) ([5eb1129](https://github.com/glasskube/distr/commit/5eb112961ba218b8f3c47de7234f25189b811ebf))
* **deps:** update axllent/mailpit docker tag to v1.25.0 ([#919](https://github.com/glasskube/distr/issues/919)) ([09748bc](https://github.com/glasskube/distr/commit/09748bcf322a2a261e317709b0b1693555bc964b))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to 188ddfb ([#921](https://github.com/glasskube/distr/issues/921)) ([78421a8](https://github.com/glasskube/distr/commit/78421a801ca13591481eaabf4dbb67507c300d43))
* **ui:** hint that metrics reporting needs cluster scope ([#915](https://github.com/glasskube/distr/issues/915)) ([99efdae](https://github.com/glasskube/distr/commit/99efdae91c73abe148707e0f14569496bfd25f5d))

## [1.9.0](https://github.com/glasskube/distr/compare/1.8.1...1.9.0) (2025-05-16)


### Features

* add host system metrics collection for deployment targets ([#899](https://github.com/glasskube/distr/issues/899)) ([1d03c3a](https://github.com/glasskube/distr/commit/1d03c3ab25c4b62de126909127bb5442025c8c3d))


### Bug Fixes

* **deps:** update angular monorepo to v19.2.11 ([#909](https://github.com/glasskube/distr/issues/909)) ([b587766](https://github.com/glasskube/distr/commit/b5877663ed0ef6c2cac4596ea3551e1900357c8b))
* **deps:** update dependency @angular/cdk to v19.2.16 ([#903](https://github.com/glasskube/distr/issues/903)) ([7243289](https://github.com/glasskube/distr/commit/7243289293bc85f5e9cb1b4cca3686bbf59242fb))
* **deps:** update dependency @codemirror/view to v6.36.8 ([#895](https://github.com/glasskube/distr/issues/895)) ([88b2a84](https://github.com/glasskube/distr/commit/88b2a84feed918f0166c46f0af28a7323ef9c233))
* **deps:** update dependency semver to v7.7.2 ([#897](https://github.com/glasskube/distr/issues/897)) ([68814a5](https://github.com/glasskube/distr/commit/68814a5e03b8b1a5a4c794d8d2890d3d9a697e7a))
* **deps:** update kubernetes packages to v0.33.1 ([#912](https://github.com/glasskube/distr/issues/912)) ([e241943](https://github.com/glasskube/distr/commit/e2419434154b2f7d023ff9c8516cec75cc36d24a))
* **deps:** update module github.com/exaring/otelpgx to v0.9.3 ([#898](https://github.com/glasskube/distr/issues/898)) ([4f47e0d](https://github.com/glasskube/distr/commit/4f47e0d1ecaf520a9d90c726e5d0753f324e0fd0))
* **deps:** update module github.com/getsentry/sentry-go to v0.33.0 ([#910](https://github.com/glasskube/distr/issues/910)) ([f389821](https://github.com/glasskube/distr/commit/f38982160165990460c4fc066c043e80da45d3eb))
* **deps:** update module github.com/getsentry/sentry-go/otel to v0.33.0 ([#911](https://github.com/glasskube/distr/issues/911)) ([4ef490b](https://github.com/glasskube/distr/commit/4ef490bfad5278a41fa19e953cad102dc7bc7740))
* **deps:** update module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver to v0.126.0 ([#904](https://github.com/glasskube/distr/issues/904)) ([55288c4](https://github.com/glasskube/distr/commit/55288c4050f03fd69d24a01db20375a7e2535552))
* **deps:** update module k8s.io/metrics to v0.33.0 ([#906](https://github.com/glasskube/distr/issues/906)) ([ef8b17d](https://github.com/glasskube/distr/commit/ef8b17d97004952c5b2b2283445168677f195ecd))
* set tutorial deployment docker type ([#916](https://github.com/glasskube/distr/issues/916)) ([46aeebe](https://github.com/glasskube/distr/commit/46aeebea79a1320416222ddf4faed9ab2ef3be7c))
* **ui:** break long texts to avoid layout disruption ([#914](https://github.com/glasskube/distr/issues/914)) ([5fd2d08](https://github.com/glasskube/distr/commit/5fd2d081c72fc6c8cef5dfd9e6a43d4d8688ac8b))
* **ui:** optimize deployments view for smaller devices ([#901](https://github.com/glasskube/distr/issues/901)) ([47794b1](https://github.com/glasskube/distr/commit/47794b1af5c90eabf91a5d83d084cd4dbbf0b9ef))


### Other

* **agent:** rename cluster role binding ([#913](https://github.com/glasskube/distr/issues/913)) ([39ecbdf](https://github.com/glasskube/distr/commit/39ecbdf7938152a5c70921ea61395b6dc4e6abf0))
* **deps:** update angular-cli monorepo to v19.2.12 ([#902](https://github.com/glasskube/distr/issues/902)) ([27ef621](https://github.com/glasskube/distr/commit/27ef62128a03826dd225bff65d2e56f62461ef94))
* **deps:** update docker/build-push-action action to v6.17.0 ([#908](https://github.com/glasskube/distr/issues/908)) ([804be8b](https://github.com/glasskube/distr/commit/804be8bc9eba096aefa685852ae851d57a117d14))
* **deps:** update tailwindcss monorepo to v4.1.7 ([#907](https://github.com/glasskube/distr/issues/907)) ([8c0580d](https://github.com/glasskube/distr/commit/8c0580df60d90afd88405199c41baaeae39e36d6))
* **deps:** update webpack due to vuln ([#900](https://github.com/glasskube/distr/issues/900)) ([21ae03d](https://github.com/glasskube/distr/commit/21ae03de80bfe60d081d5dc0ec850834f043be13))

## [1.8.1](https://github.com/glasskube/distr/compare/1.8.0...1.8.1) (2025-05-12)


### Bug Fixes

* **agent:** prevent implicit import of `env` package ([#893](https://github.com/glasskube/distr/issues/893)) ([6eb1be6](https://github.com/glasskube/distr/commit/6eb1be67c7f7ff932f31d1b3ebeaece77be9e515))

## [1.8.0](https://github.com/glasskube/distr/compare/1.7.1...1.8.0) (2025-05-12)


### Features

* add support for custom domains and email from address ([#882](https://github.com/glasskube/distr/issues/882)) ([b7974e6](https://github.com/glasskube/distr/commit/b7974e6e84796727aac3bc4673b4fb18d1f3e25a))
* docker swarm support ([#645](https://github.com/glasskube/distr/issues/645)) ([d842dda](https://github.com/glasskube/distr/commit/d842dda44fd6b575d2586a5689ad2d2a83371d4e))


### Bug Fixes

* **deps:** update angular monorepo to v19.2.10 ([#884](https://github.com/glasskube/distr/issues/884)) ([841865b](https://github.com/glasskube/distr/commit/841865bdf49ca935116d722d4580f62a10c35a74))
* **deps:** update dependency @angular/cdk to v19.2.15 ([#883](https://github.com/glasskube/distr/issues/883)) ([5fec38a](https://github.com/glasskube/distr/commit/5fec38a4e75abcfe3aaa3a376606a77b0020638b))
* **deps:** update dependency @sentry/angular to v9.17.0 ([#890](https://github.com/glasskube/distr/issues/890)) ([081c384](https://github.com/glasskube/distr/commit/081c3848fa1974351ab2ae1759bab7b2633050b0))
* **deps:** update dependency posthog-js to v1.240.6 ([#891](https://github.com/glasskube/distr/issues/891)) ([1062edc](https://github.com/glasskube/distr/commit/1062edc9f58588f6c634043cab259310465d6e03))
* **deps:** update module github.com/compose-spec/compose-go/v2 to v2.6.2 ([#880](https://github.com/glasskube/distr/issues/880)) ([9eb2e41](https://github.com/glasskube/distr/commit/9eb2e41f5b36325febbbcaccbdc5004d6897b640))
* **deps:** update module github.com/exaring/otelpgx to v0.9.2 ([#888](https://github.com/glasskube/distr/issues/888)) ([24f119a](https://github.com/glasskube/distr/commit/24f119a25862c2ca389da85ff30b3afe0d5f6e63))


### Other

* **deps:** update actions/setup-go action to v5.5.0 ([#885](https://github.com/glasskube/distr/issues/885)) ([33f72d9](https://github.com/glasskube/distr/commit/33f72d96b4f31a97b17ad5e0ed419f59c9f30187))
* **deps:** update angular-cli monorepo to v19.2.11 ([#879](https://github.com/glasskube/distr/issues/879)) ([fe470f7](https://github.com/glasskube/distr/commit/fe470f76340fc6b2db764115158f59b693f76cf2))
* **deps:** update dependency @sentry/cli to v2.45.0 ([#889](https://github.com/glasskube/distr/issues/889)) ([b7c9bd8](https://github.com/glasskube/distr/commit/b7c9bd8dd4707895150d938e028c2d978bb896dd))
* **deps:** update postgres docker tag to v17.5 ([#886](https://github.com/glasskube/distr/issues/886)) ([c327a19](https://github.com/glasskube/distr/commit/c327a199e95c6c7fe83f7906d040d5166ed180c8))
* **deps:** update tailwindcss monorepo to v4.1.6 ([#887](https://github.com/glasskube/distr/issues/887)) ([630a659](https://github.com/glasskube/distr/commit/630a6591d1e438404252ccf612514888ab7f3e1b))
* redirect new users to tutorials ([#892](https://github.com/glasskube/distr/issues/892)) ([e69f3f9](https://github.com/glasskube/distr/commit/e69f3f91acd77e21d5365b0da935efd6173cf2d2))
* **ui:** make tutorial backlink more prominent ([#873](https://github.com/glasskube/distr/issues/873)) ([dfa3453](https://github.com/glasskube/distr/commit/dfa3453ce1a11cfcaba48f9a8f7ff9718b0cf0d9))

## [1.7.1](https://github.com/glasskube/distr/compare/1.7.0...1.7.1) (2025-05-07)


### Bug Fixes

* **agent:** kubernetes agent GetLatestHelmRelease supports more than 10 revisions ([#876](https://github.com/glasskube/distr/issues/876)) ([7558784](https://github.com/glasskube/distr/commit/75587844f59c770e45238e7292b29c1bd22da876))
* **deps:** update angular monorepo to v19.2.9 ([#846](https://github.com/glasskube/distr/issues/846)) ([2528080](https://github.com/glasskube/distr/commit/252808084f29df27e8efce0dbbc86376bcae661a))
* **deps:** update dependency @angular/cdk to v19.2.14 ([#848](https://github.com/glasskube/distr/issues/848)) ([245ddbb](https://github.com/glasskube/distr/commit/245ddbb22dee168f5e5d0d4472136737db8dc3be))
* **deps:** update dependency @codemirror/view to v6.36.7 ([#851](https://github.com/glasskube/distr/issues/851)) ([73c396a](https://github.com/glasskube/distr/commit/73c396a15c0f0599427736af6a41bb2e2ff33c9a))
* **deps:** update dependency @fontsource/poppins to v5.2.6 ([#860](https://github.com/glasskube/distr/issues/860)) ([c16818d](https://github.com/glasskube/distr/commit/c16818d64e6b36942e1ff4a10f700f5ced71f82d))
* **deps:** update dependency @sentry/angular to v9.15.0 ([#863](https://github.com/glasskube/distr/issues/863)) ([0b6b206](https://github.com/glasskube/distr/commit/0b6b206f137f103a346e32d9802d88fc727ee6f6))
* **deps:** update dependency posthog-js to v1.239.1 ([#864](https://github.com/glasskube/distr/issues/864)) ([78ced57](https://github.com/glasskube/distr/commit/78ced579fb319725aecd9e9919e8ffba3f60a919))
* **deps:** update module github.com/exaring/otelpgx to v0.9.1 ([#853](https://github.com/glasskube/distr/issues/853)) ([2ae5ac6](https://github.com/glasskube/distr/commit/2ae5ac6a91cf05ca44b4d9df79f25db2734cc217))
* **deps:** update module golang.org/x/crypto to v0.38.0 ([#871](https://github.com/glasskube/distr/issues/871)) ([728f7f1](https://github.com/glasskube/distr/commit/728f7f1272d6f71a1b3c0e3f958d0d77682956a8))
* recreate tutorial resources if deleted ([#869](https://github.com/glasskube/distr/issues/869)) ([25047a1](https://github.com/glasskube/distr/commit/25047a123700eba88003b2c384981d6f04253b25))
* show customer image in navbar ([#866](https://github.com/glasskube/distr/issues/866)) ([258e807](https://github.com/glasskube/distr/commit/258e8070cdfc815c70ef767bcff4e08222b8f86a))
* **ui:** add missing icon when licensing disabled ([#858](https://github.com/glasskube/distr/issues/858)) ([72bbf0f](https://github.com/glasskube/distr/commit/72bbf0f9daafa3e937b2d45042047b09c90e9550))
* **ui:** improve markdown rendering on customer "home" page ([#867](https://github.com/glasskube/distr/issues/867)) ([cb23725](https://github.com/glasskube/distr/commit/cb2372570d92ab069bce35f8ef81056a5ab27b5a))
* **ui:** show hint for mac users ([#861](https://github.com/glasskube/distr/issues/861)) ([e696033](https://github.com/glasskube/distr/commit/e696033fe1f3f709f007592a810c149297757287))


### Other

* **backend:** add `DATABASE_MAX_CONNS` override parameter ([#868](https://github.com/glasskube/distr/issues/868)) ([4e8fb82](https://github.com/glasskube/distr/commit/4e8fb82546b8e304986434fb2b24c63840781428))
* **deps:** update angular-cli monorepo to v19.2.10 ([#849](https://github.com/glasskube/distr/issues/849)) ([70b0bc8](https://github.com/glasskube/distr/commit/70b0bc83ced6be374e51a3e82863dc90393ddd57))
* **deps:** update axllent/mailpit docker tag to v1.24.2 ([#852](https://github.com/glasskube/distr/issues/852)) ([c675495](https://github.com/glasskube/distr/commit/c675495939c16b2a71daf763d4e7015b2c9b51f9))
* **deps:** update dependency @sentry/cli to v2.44.0 ([#859](https://github.com/glasskube/distr/issues/859)) ([8c25ab4](https://github.com/glasskube/distr/commit/8c25ab41db1f5bf03f38c79aca66bcf0263e0fce))
* **deps:** update dependency @types/jasmine to v5.1.8 ([#870](https://github.com/glasskube/distr/issues/870)) ([a644901](https://github.com/glasskube/distr/commit/a644901641ed17589ab7540c1c1fa54bc15b9a5c))
* **deps:** update dependency go to v1.24.3 ([#877](https://github.com/glasskube/distr/issues/877)) ([21f90d1](https://github.com/glasskube/distr/commit/21f90d14e1807c3850fd406927672c776e787c4e))
* **deps:** update dependency golangci-lint to v2.1.6 ([#855](https://github.com/glasskube/distr/issues/855)) ([94abaf2](https://github.com/glasskube/distr/commit/94abaf233bd764faf98e023cf50f15d0465d5040))
* **deps:** update dependency jasmine-core to v5.7.1 ([#850](https://github.com/glasskube/distr/issues/850)) ([971b79a](https://github.com/glasskube/distr/commit/971b79a2c4371f3f0a3e79d002d53a7b82abb8a1))
* **deps:** update dependency typedoc to v0.28.4 ([#854](https://github.com/glasskube/distr/issues/854)) ([9bab574](https://github.com/glasskube/distr/commit/9bab574105e8197c2190ca816acf614e44501bfb))
* **deps:** update golangci/golangci-lint-action action to v7.0.1 ([#856](https://github.com/glasskube/distr/issues/856)) ([351a857](https://github.com/glasskube/distr/commit/351a857be7dc69375e29e18b194793c0c8881213))
* **deps:** update golangci/golangci-lint-action action to v8 ([#857](https://github.com/glasskube/distr/issues/857)) ([1c5d369](https://github.com/glasskube/distr/commit/1c5d369b5329ab2e6e6174b2b4fe33d1d7b40e18))
* **deps:** update tailwindcss monorepo to v4.1.5 ([#845](https://github.com/glasskube/distr/issues/845)) ([fb960ae](https://github.com/glasskube/distr/commit/fb960ae940015d2c623d210a5742d7827e01b51a))
* **ui:** hide application version form by default ([#875](https://github.com/glasskube/distr/issues/875)) ([6eec16b](https://github.com/glasskube/distr/commit/6eec16b1b04b2b163454ed94017b980de9075779))
* **ui:** hide artifact download information in customer portal ([#862](https://github.com/glasskube/distr/issues/862)) ([2aef7d4](https://github.com/glasskube/distr/commit/2aef7d45e30ce3b604170770df16df67352c9772))
* **ui:** increase font sizes in tutorials ([#874](https://github.com/glasskube/distr/issues/874)) ([c6fad6a](https://github.com/glasskube/distr/commit/c6fad6ab2a07bc15d5c08881e61c7c68d702e36c))
* **ui:** use explicit monospace font stack ([#865](https://github.com/glasskube/distr/issues/865)) ([2097d31](https://github.com/glasskube/distr/commit/2097d318bbc710bb5becd66fd402bf1d5fd23acc))

## [1.7.0](https://github.com/glasskube/distr/compare/1.6.1...1.7.0) (2025-04-30)


### Features

* add tutorials, adapt feature flag semantics and sidebar ([#814](https://github.com/glasskube/distr/issues/814)) ([f86f513](https://github.com/glasskube/distr/commit/f86f513bfdc29d1834f777a28b33334d261c1b6f))


### Bug Fixes

* **backend:** return image ID in GetArtifactByName ([#843](https://github.com/glasskube/distr/issues/843)) ([445e562](https://github.com/glasskube/distr/commit/445e5620d1569b73e4d0ad209eff3983c7b4c8f4))


### Other

* increase request limit ([#844](https://github.com/glasskube/distr/issues/844)) ([03ef0e9](https://github.com/glasskube/distr/commit/03ef0e999ed27fb1e02abe5976b85db4f0920853))
* new design for vendor dashboard ([#840](https://github.com/glasskube/distr/issues/840)) ([a04615d](https://github.com/glasskube/distr/commit/a04615d80a276898ef57a03e388917e33fbc0dd4))
* **ui:** make agent status indicator bigger ([#839](https://github.com/glasskube/distr/issues/839)) ([54ae61f](https://github.com/glasskube/distr/commit/54ae61f4635df34c89ee9a0941b842f8172ccdfd))
* **ui:** update helm release name validation ([#842](https://github.com/glasskube/distr/issues/842)) ([39d0619](https://github.com/glasskube/distr/commit/39d061920082d86df7b15fe9b8b9391985b55f2d))

## [1.6.1](https://github.com/glasskube/distr/compare/1.6.0...1.6.1) (2025-04-29)


### Bug Fixes

* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.79.3 ([#832](https://github.com/glasskube/distr/issues/832)) ([415c2f9](https://github.com/glasskube/distr/commit/415c2f980c1f1c6a13e8b957ac9384e637fd0037))
* **deps:** update module github.com/lestrrat-go/jwx/v2 to v2.1.6 ([#835](https://github.com/glasskube/distr/issues/835)) ([5863f7f](https://github.com/glasskube/distr/commit/5863f7f897d5bb168a112ebda383605c014d43f2))
* **ui:** make deployment modal scrollable ([#837](https://github.com/glasskube/distr/issues/837)) ([9fe543d](https://github.com/glasskube/distr/commit/9fe543d65313afa23a58df6583689a37c085dc59))


### Other

* **ui:** improve deploy button wording on deployment page ([#836](https://github.com/glasskube/distr/issues/836)) ([1b525d9](https://github.com/glasskube/distr/commit/1b525d9fdc091cd05951145f08361f97999dc87d))
* **ui:** improve external registry wording ([#838](https://github.com/glasskube/distr/issues/838)) ([630afe2](https://github.com/glasskube/distr/commit/630afe22d4d32b272c4d42dcd3bd05556977ff50))

## [1.6.0](https://github.com/glasskube/distr/compare/1.5.2...1.6.0) (2025-04-28)


### Features

* add images for users, artifacts and applications ([#764](https://github.com/glasskube/distr/issues/764)) ([a414a44](https://github.com/glasskube/distr/commit/a414a4480ea204f1f90ab3191f210cf7d180ef8b))
* add support for multiple deployments per deployment target ([#809](https://github.com/glasskube/distr/issues/809)) ([4c58021](https://github.com/glasskube/distr/commit/4c580218bbfd24bce5c8fdbb654996a40bac4e20))


### Bug Fixes

* **agent:** ensure token after 401 response ([#807](https://github.com/glasskube/distr/issues/807)) ([592b7a8](https://github.com/glasskube/distr/commit/592b7a8818d92997ace5bd611f71fc9a10fcf4f8))
* **backend:** correct artifact version download numbers ([#817](https://github.com/glasskube/distr/issues/817)) ([7695c91](https://github.com/glasskube/distr/commit/7695c918e4dc4beb5356d317e5ed583212a328f6))
* **deps:** update angular monorepo to v19.2.8 ([#818](https://github.com/glasskube/distr/issues/818)) ([a2f3588](https://github.com/glasskube/distr/commit/a2f3588907fd3432e915c199a2d3846772829b8f))
* **deps:** update dependency @angular/cdk to v19.2.11 ([#819](https://github.com/glasskube/distr/issues/819)) ([6241f1f](https://github.com/glasskube/distr/commit/6241f1f86fa33adb0a31ae4dc017b1bfc4e89377))
* **deps:** update dependency @codemirror/view to v6.36.6 ([#821](https://github.com/glasskube/distr/issues/821)) ([33d3407](https://github.com/glasskube/distr/commit/33d3407d074bc66f0520f6137833907dbacb3d4d))
* **deps:** update dependency @sentry/angular to v9.13.0 ([#804](https://github.com/glasskube/distr/issues/804)) ([61a6c20](https://github.com/glasskube/distr/commit/61a6c20d216d13acdac67c6322145b2f70fdd07d))
* **deps:** update dependency @sentry/angular to v9.14.0 ([#830](https://github.com/glasskube/distr/issues/830)) ([da75b30](https://github.com/glasskube/distr/commit/da75b308f9654b621d34f49a792d936024d2a4f6))
* **deps:** update dependency apexcharts to v4.6.0 ([#806](https://github.com/glasskube/distr/issues/806)) ([40a4d4e](https://github.com/glasskube/distr/commit/40a4d4e7c28aa37241f636c0a3fffa04e83b3896))
* **deps:** update dependency apexcharts to v4.7.0 ([#823](https://github.com/glasskube/distr/issues/823)) ([089841e](https://github.com/glasskube/distr/commit/089841e0e6672e6ed71de25cd02a019c69622539))
* **deps:** update dependency flowbite to v3.1.2 ([#808](https://github.com/glasskube/distr/issues/808)) ([f4ade80](https://github.com/glasskube/distr/commit/f4ade80a5874c397771264a6d47d792004c8a8a9))
* **deps:** update dependency posthog-js to v1.236.4 ([#805](https://github.com/glasskube/distr/issues/805)) ([857a48e](https://github.com/glasskube/distr/commit/857a48e26249fec801865b1fd76781802f0a6599))
* **deps:** update dependency posthog-js to v1.236.7 ([#829](https://github.com/glasskube/distr/issues/829)) ([895b5ab](https://github.com/glasskube/distr/commit/895b5abfd170eb5bd1259ca43655be8ee68d0b03))
* **deps:** update kubernetes packages to v0.33.0 ([#820](https://github.com/glasskube/distr/issues/820)) ([639dcea](https://github.com/glasskube/distr/commit/639dceafd111c9dfde44aa35ce9d9e01bb628bf8))
* **deps:** update module github.com/docker/cli to v28.1.0+incompatible ([#797](https://github.com/glasskube/distr/issues/797)) ([5f1c4bf](https://github.com/glasskube/distr/commit/5f1c4bf9773554d8f2825f43c5ba73c43e7b28f2))
* **deps:** update module github.com/docker/cli to v28.1.1+incompatible ([#802](https://github.com/glasskube/distr/issues/802)) ([f175122](https://github.com/glasskube/distr/commit/f175122d52c955e20b056b8daf1cf83b0da06a81))
* **deps:** update module github.com/golang-migrate/migrate/v4 to v4.18.3 ([#822](https://github.com/glasskube/distr/issues/822)) ([baae889](https://github.com/glasskube/distr/commit/baae8892063b3cf8ff24711e1db04cd259ba06bf))
* **deps:** update module github.com/masterminds/semver/v3 to v3.3.1 ([#831](https://github.com/glasskube/distr/issues/831)) ([ae11c3a](https://github.com/glasskube/distr/commit/ae11c3a1263ee11ba945a948f47435ec311b5e8a))
* **deps:** update module k8s.io/cli-runtime to v0.32.4 ([#816](https://github.com/glasskube/distr/issues/816)) ([474d6b2](https://github.com/glasskube/distr/commit/474d6b2f66b1163903e81e90fedac6ea98c6e90a))
* **deps:** update module k8s.io/client-go to v0.32.4 ([#813](https://github.com/glasskube/distr/issues/813)) ([1f07da9](https://github.com/glasskube/distr/commit/1f07da9a0f3dc1ee7d868ee268e3ba019de4b172))


### Other

* add OpenTelemetry tracing ([#801](https://github.com/glasskube/distr/issues/801)) ([67ce94b](https://github.com/glasskube/distr/commit/67ce94bda6c674a5edc16250953dfe1c94c09b2e))
* **deps:** update angular-cli monorepo to v19.2.9 ([#811](https://github.com/glasskube/distr/issues/811)) ([f7c0757](https://github.com/glasskube/distr/commit/f7c0757b26c591527e9ee83ef701a233bb8bc554))
* **deps:** update dependency golangci-lint to v2.1.5 ([#825](https://github.com/glasskube/distr/issues/825)) ([1ff2176](https://github.com/glasskube/distr/commit/1ff2176a975be459053aa83afa941ae8a665d720))
* **deps:** update dependency jasmine-core to ~5.7.0 ([#828](https://github.com/glasskube/distr/issues/828)) ([730529e](https://github.com/glasskube/distr/commit/730529ee42aab882a2491ed75aa213f77adbc45e))
* **deps:** update dependency typedoc to v0.28.3 ([#803](https://github.com/glasskube/distr/issues/803)) ([9ae5f9a](https://github.com/glasskube/distr/commit/9ae5f9a143fd551dac7bddea9a3bd1755ea2fd1f))
* **deps:** update dependency typedoc-plugin-markdown to v4.6.3 ([#815](https://github.com/glasskube/distr/issues/815)) ([8d6ceee](https://github.com/glasskube/distr/commit/8d6ceee815c3cf6ce6473c0c2b8b1f780defab4e))
* **deps:** update docker/build-push-action action to v6.16.0 ([#824](https://github.com/glasskube/distr/issues/824)) ([1374af9](https://github.com/glasskube/distr/commit/1374af9531f24baaccd831dc3d5c36b02d59d203))
* **deps:** upgrade tailwindcss to v4 and flowbite to v3 ([#799](https://github.com/glasskube/distr/issues/799)) ([4306903](https://github.com/glasskube/distr/commit/4306903a0e29babd90c9037682f0cb4ff2e54fb6))
* **registry:** do not accept non-compliant OCI manifest ([#827](https://github.com/glasskube/distr/issues/827)) ([72ea6da](https://github.com/glasskube/distr/commit/72ea6da763ca7955db82a1837dd55aa06ee1c14c))
* **ui:** fix button cursor after tailwind upgrade ([#812](https://github.com/glasskube/distr/issues/812)) ([5231b24](https://github.com/glasskube/distr/commit/5231b246d48cf7f16bafb584eab52f6ff9e808a4))
* **ui:** fix spacing issues after tailwind upgrade ([#826](https://github.com/glasskube/distr/issues/826)) ([c00f20a](https://github.com/glasskube/distr/commit/c00f20a6140cfd9727743b27c32ced5bcd97908b))

## [1.5.2](https://github.com/glasskube/distr/compare/1.5.1...1.5.2) (2025-04-17)


### Bug Fixes

* **deps:** update angular monorepo to v19.2.7 ([#794](https://github.com/glasskube/distr/issues/794)) ([dbffea2](https://github.com/glasskube/distr/commit/dbffea2a693d28f104ecbbb7d107e14e97d03932))
* **deps:** update dependency @angular/cdk to v19.2.10 ([#791](https://github.com/glasskube/distr/issues/791)) ([4ccf8be](https://github.com/glasskube/distr/commit/4ccf8be6cff8c52b4b785158a94347ea8180fafe))


### Other

* **deps:** update angular-cli monorepo to v19.2.8 ([#790](https://github.com/glasskube/distr/issues/790)) ([c243855](https://github.com/glasskube/distr/commit/c243855a4589c1d9fc9c3c6203b0dccb77003e01))
* enable Distr registry for new organizations ([#796](https://github.com/glasskube/distr/issues/796)) ([29a27a3](https://github.com/glasskube/distr/commit/29a27a34ce0d223fa2d4585afb770e692840e231))
* **ui:** disable archived versions in license edit form ([#782](https://github.com/glasskube/distr/issues/782)) ([4622fc9](https://github.com/glasskube/distr/commit/4622fc904a4e80e6e1f46f784c0d43089138a0d9))

## [1.5.1](https://github.com/glasskube/distr/compare/1.5.0...1.5.1) (2025-04-16)


### Bug Fixes

* **deps:** update module github.com/lestrrat-go/jwx/v2 to v2.1.5 ([#789](https://github.com/glasskube/distr/issues/789)) ([2033559](https://github.com/glasskube/distr/commit/203355914f1ba09f3266d13becc1fac28343f4dd))


### Other

* **agent:** kubernetes agent client config reload ([#775](https://github.com/glasskube/distr/issues/775)) ([00b59b6](https://github.com/glasskube/distr/commit/00b59b66d6bff01542eaee93d5dfedad7b11e4c7))
* **deps:** update dependency golangci-lint to v2.1.2 ([#787](https://github.com/glasskube/distr/issues/787)) ([215227a](https://github.com/glasskube/distr/commit/215227a9dba3ab42f5c0b033fec9e34ff72de868))
* **ui:** don't display push instructions for non vendors ([#788](https://github.com/glasskube/distr/issues/788)) ([2aa786b](https://github.com/glasskube/distr/commit/2aa786bd954184a9ec25ca007e755325ea310bf7))


### Docs

* add Discord support link ([#785](https://github.com/glasskube/distr/issues/785)) ([13c8729](https://github.com/glasskube/distr/commit/13c87298fc621d73a4468de5b95c38ccaa7e30c0))
* update readme ([#777](https://github.com/glasskube/distr/issues/777)) ([a00c8e2](https://github.com/glasskube/distr/commit/a00c8e273bcbd2c809acd6c2316c9f2455fb5cb0))

## [1.5.0](https://github.com/glasskube/distr/compare/1.4.7...1.5.0) (2025-04-14)


### Features

* agent registry auth ([#747](https://github.com/glasskube/distr/issues/747)) ([e55005a](https://github.com/glasskube/distr/commit/e55005af477c4e0f95802fb6eaf2f88678b7b170))


### Bug Fixes

* **agent:** kubernetes agent sends wrong status type for "progressing" ([ff0a337](https://github.com/glasskube/distr/commit/ff0a337db48aa073948f214cd36526aa592ef5de))
* **deps:** update angular monorepo to v19.2.6 ([#767](https://github.com/glasskube/distr/issues/767)) ([295bce3](https://github.com/glasskube/distr/commit/295bce3913898374cbd0429edf1e2287f29d4e28))
* **deps:** update aws-sdk-go-v2 monorepo ([#776](https://github.com/glasskube/distr/issues/776)) ([9377f94](https://github.com/glasskube/distr/commit/9377f94fb8e86ce0a838d61988f5b0eb59214d67))
* **deps:** update dependency @angular/cdk to v19.2.9 ([#770](https://github.com/glasskube/distr/issues/770)) ([51e3d82](https://github.com/glasskube/distr/commit/51e3d82f91def38f6676b6499ee332cfc66822bb))
* **deps:** update dependency @sentry/angular to v9.12.0 ([#781](https://github.com/glasskube/distr/issues/781)) ([8adc88a](https://github.com/glasskube/distr/commit/8adc88abea66ceeb81149bb6923f7ded4861abae))
* **deps:** update dependency posthog-js to v1.235.6 ([#783](https://github.com/glasskube/distr/issues/783)) ([a78c127](https://github.com/glasskube/distr/commit/a78c127cf804774afbdece6a08b3a40db4e07160))
* **deps:** update module github.com/getsentry/sentry-go to v0.32.0 ([#773](https://github.com/glasskube/distr/issues/773)) ([01c5b58](https://github.com/glasskube/distr/commit/01c5b585bc3ab832a58196008be3807dd5dcf723))
* **deps:** update module helm.sh/helm/v3 to v3.17.3 ([#772](https://github.com/glasskube/distr/issues/772)) ([6f611a3](https://github.com/glasskube/distr/commit/6f611a37aa085ae0a866ce03863b00e12db21655))


### Other

* **deps:** update actions/setup-node action to v4.4.0 ([#780](https://github.com/glasskube/distr/issues/780)) ([8dc422c](https://github.com/glasskube/distr/commit/8dc422c1d12c59e6dcff974d60e795ee2fc0261a))
* **deps:** update angular-cli monorepo to v19.2.7 ([#769](https://github.com/glasskube/distr/issues/769)) ([ba499e2](https://github.com/glasskube/distr/commit/ba499e21af6c9075c03a5688839903214ec06eae))
* **deps:** update axllent/mailpit docker tag to v1.24.1 ([#778](https://github.com/glasskube/distr/issues/778)) ([12cb796](https://github.com/glasskube/distr/commit/12cb7964ff274db09836cae4c1b14725284cc618))
* **deps:** update dependency golangci-lint to v2.1.1 ([#779](https://github.com/glasskube/distr/issues/779)) ([6e26b4b](https://github.com/glasskube/distr/commit/6e26b4b2b1ca9f10614f81cf14d63ac5ed3ded25))
* **deps:** update dependency typedoc-plugin-markdown to v4.6.2 ([#766](https://github.com/glasskube/distr/issues/766)) ([f675c1b](https://github.com/glasskube/distr/commit/f675c1b92ec85d246de5e44235ddf0880dbc1397))
* **registry:** propagate invalid name errors and return 400 ([#765](https://github.com/glasskube/distr/issues/765)) ([110ff0d](https://github.com/glasskube/distr/commit/110ff0d929ac38ccc13ee53e4743cdd149bb7455))
* **ui:** change stale message for empty deployment targets ([#771](https://github.com/glasskube/distr/issues/771)) ([f324d0b](https://github.com/glasskube/distr/commit/f324d0b2ca86f31d69e07c8b46e9e782c4c46167))
* **ui:** reconnect in dropdown, hide status if target empty ([#768](https://github.com/glasskube/distr/issues/768)) ([7f9e6ac](https://github.com/glasskube/distr/commit/7f9e6ac993691cb18efc873a364dbf1247e135a6))


### Docs

* add registry to README.md ([#761](https://github.com/glasskube/distr/issues/761)) ([7d1a657](https://github.com/glasskube/distr/commit/7d1a65795c061f0301b1097c1782eadaa9c55ed9))

## [1.4.7](https://github.com/glasskube/distr/compare/1.4.6...1.4.7) (2025-04-08)


### Bug Fixes

* **ui:** deployed version text overflow ([#759](https://github.com/glasskube/distr/issues/759)) ([ee45edf](https://github.com/glasskube/distr/commit/ee45edf597dfa6748471d589277e64f4b0492664))


### Other

* **ui:** add Helm release name validation ([#757](https://github.com/glasskube/distr/issues/757)) ([a8adff4](https://github.com/glasskube/distr/commit/a8adff40dd68654b7ec661d5fb77b3d7fe9ae1ad))
* **ui:** increase stale timeout to 60 secs ([#758](https://github.com/glasskube/distr/issues/758)) ([117e288](https://github.com/glasskube/distr/commit/117e28829e376499b8f56cff0e33b6cde6f54f73))

## [1.4.6](https://github.com/glasskube/distr/compare/1.4.5...1.4.6) (2025-04-08)


### Other

* **ui:** add loading spinner on deployment status modal ([#754](https://github.com/glasskube/distr/issues/754)) ([7fad29c](https://github.com/glasskube/distr/commit/7fad29c3709db5044fe4fa79d7a7ae3f97edda1d))


### Performance

* **backend:** optimize DB query to get DeploymentRevisionStatus ([#753](https://github.com/glasskube/distr/issues/753)) ([46be9e9](https://github.com/glasskube/distr/commit/46be9e98a3141b3f8b82eaf9cd51882e18b41b40))

## [1.4.5](https://github.com/glasskube/distr/compare/1.4.4...1.4.5) (2025-04-07)


### Bug Fixes

* **backend:** correct artifact version downloads for customer ([#742](https://github.com/glasskube/distr/issues/742)) ([ecfb521](https://github.com/glasskube/distr/commit/ecfb521b7acafb733c1edc0fcc36e2b04da426a6))
* **backend:** correctly handle artifact size when there are no parts ([#719](https://github.com/glasskube/distr/issues/719)) ([22c0133](https://github.com/glasskube/distr/commit/22c0133bfcba1cc9d3686902c7bc2215a8f0cba0))
* **backend:** update organization without slug leads to panic ([#727](https://github.com/glasskube/distr/issues/727)) ([c926fed](https://github.com/glasskube/distr/commit/c926fed5acba08f43e80070e1519b6b611c129e9))
* **deps:** update angular monorepo to v19.2.5 ([#739](https://github.com/glasskube/distr/issues/739)) ([8c52183](https://github.com/glasskube/distr/commit/8c52183ae8d5bc97a67c480b15c525172c1201af))
* **deps:** update aws-sdk-go-v2 monorepo ([#744](https://github.com/glasskube/distr/issues/744)) ([c392d30](https://github.com/glasskube/distr/commit/c392d3070b5130a95b1fe90f0fc83569321ff5df))
* **deps:** update dependency @angular/cdk to v19.2.8 ([#740](https://github.com/glasskube/distr/issues/740)) ([511a602](https://github.com/glasskube/distr/commit/511a6021384c3cf38fde80c0db4dfea665ff44ef))
* **deps:** update dependency @codemirror/commands to v6.8.1 ([#725](https://github.com/glasskube/distr/issues/725)) ([b3e5e12](https://github.com/glasskube/distr/commit/b3e5e129438f68319ad3a53da81b661a66cf8498))
* **deps:** update dependency @codemirror/view to v6.36.5 ([#721](https://github.com/glasskube/distr/issues/721)) ([78e3bec](https://github.com/glasskube/distr/commit/78e3bec5c95f96fc04a56b0d7efc1bcd2007f33e))
* **deps:** update dependency @sentry/angular to v9.10.1 ([#729](https://github.com/glasskube/distr/issues/729)) ([70b9d26](https://github.com/glasskube/distr/commit/70b9d2610c177d202ec3503cb2d84d88beeefd01))
* **deps:** update dependency @sentry/angular to v9.11.0 ([#752](https://github.com/glasskube/distr/issues/752)) ([d0f3f9c](https://github.com/glasskube/distr/commit/d0f3f9ca6cd862173e3a469e62fb25119bfa4a68))
* **deps:** update dependency globe.gl to v2.41.4 ([#745](https://github.com/glasskube/distr/issues/745)) ([709dedd](https://github.com/glasskube/distr/commit/709deddf02869e4677647ff4fffedb1cce1a8aab))
* **deps:** update dependency posthog-js to v1.234.6 ([#730](https://github.com/glasskube/distr/issues/730)) ([783d57f](https://github.com/glasskube/distr/commit/783d57f35501f2ad76e268fa6e22b20269080fbc))
* **deps:** update dependency posthog-js to v1.234.9 ([#751](https://github.com/glasskube/distr/issues/751)) ([1909b0f](https://github.com/glasskube/distr/commit/1909b0fc7902568d79eafa72a73258ebaec774d9))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.79.0 ([#731](https://github.com/glasskube/distr/issues/731)) ([30273a0](https://github.com/glasskube/distr/commit/30273a08a416ceb0c4123f313e158a60ad432444))
* **deps:** update module github.com/go-chi/httprate to v0.15.0 ([#724](https://github.com/glasskube/distr/issues/724)) ([91eb949](https://github.com/glasskube/distr/commit/91eb9490bd0d65ebde35355c99935c45ad6f2179))
* **deps:** update module github.com/onsi/gomega to v1.37.0 ([#738](https://github.com/glasskube/distr/issues/738)) ([23b64d8](https://github.com/glasskube/distr/commit/23b64d8cbbb2221ba4666228bc08a898adbcff8e))
* **deps:** update module golang.org/x/crypto to v0.37.0 ([#749](https://github.com/glasskube/distr/issues/749)) ([9c7d9c6](https://github.com/glasskube/distr/commit/9c7d9c6f2f8b52d19da9808922eb555a5df7a24b))
* **ui:** avoid invite link redirect loop and guide user ([#736](https://github.com/glasskube/distr/issues/736)) ([e1ee6bf](https://github.com/glasskube/distr/commit/e1ee6bf2328e13c31d6fec93075bfb008c5efa82))


### Other

* add artifact download history page ([#743](https://github.com/glasskube/distr/issues/743)) ([959cb10](https://github.com/glasskube/distr/commit/959cb1097bec2c8d880e6c36d67324336582cebd))
* add deployment revision status "progressing" ([#746](https://github.com/glasskube/distr/issues/746)) ([5e1ede9](https://github.com/glasskube/distr/commit/5e1ede9d36f16440039064f5515339508c60cb77))
* **backend:** enhanced token verification and improved visibility constraints ([#733](https://github.com/glasskube/distr/issues/733)) ([fd0ebb3](https://github.com/glasskube/distr/commit/fd0ebb31b37ca5124d05cd01e57efd730e0209b9))
* **backend:** save UserAccount last_logged_in_at on successful login ([#741](https://github.com/glasskube/distr/issues/741)) ([1216584](https://github.com/glasskube/distr/commit/12165840d216207de4da7ede088fccbb6f9f812e))
* **deps:** update angular-cli monorepo to v19.2.6 ([#737](https://github.com/glasskube/distr/issues/737)) ([6645a75](https://github.com/glasskube/distr/commit/6645a7529fa76240b60bfdbcc249ddf30857127d))
* **deps:** update axllent/mailpit docker tag to v1.24.0 ([#723](https://github.com/glasskube/distr/issues/723)) ([ed4b8dd](https://github.com/glasskube/distr/commit/ed4b8dd1d4c69673dc1d5c64a2b7429ecbc8de2b))
* **deps:** update dependency @sentry/cli to v2.43.0 ([#726](https://github.com/glasskube/distr/issues/726)) ([7480365](https://github.com/glasskube/distr/commit/7480365ddb6983500008302fb590e688d96c543e))
* **deps:** update dependency go to v1.24.2 ([#734](https://github.com/glasskube/distr/issues/734)) ([b86922e](https://github.com/glasskube/distr/commit/b86922ef8691a29403206751875f8afded2d8959))
* **deps:** update dependency typedoc to v0.28.2 ([#750](https://github.com/glasskube/distr/issues/750)) ([53a3f25](https://github.com/glasskube/distr/commit/53a3f25e18ff5cbb326b2b9239e41d13562bfa37))
* **deps:** update dependency typedoc-plugin-markdown to v4.6.1 ([#735](https://github.com/glasskube/distr/issues/735)) ([a94eb65](https://github.com/glasskube/distr/commit/a94eb658a6104b0326f83d7fd7c9fc2da723ae6f))
* **deps:** update dependency typescript to v5.8.3 ([#748](https://github.com/glasskube/distr/issues/748)) ([f602fd1](https://github.com/glasskube/distr/commit/f602fd1106faabc094d7175ac8ef634b73ec41fd))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to c0f429e ([#720](https://github.com/glasskube/distr/issues/720)) ([f0a3baa](https://github.com/glasskube/distr/commit/f0a3baab964a4873f32ce33a73843ffccfdf14e7))

## [1.4.4](https://github.com/glasskube/distr/compare/1.4.3...1.4.4) (2025-03-28)


### Bug Fixes

* **backend:** return correct download metrics for index manifest artifacts ([#715](https://github.com/glasskube/distr/issues/715)) ([1143363](https://github.com/glasskube/distr/commit/114336371f35cccf55a02f8248008e04aa2fae0f))
* **deps:** update aws-sdk-go-v2 monorepo ([#718](https://github.com/glasskube/distr/issues/718)) ([cb3c481](https://github.com/glasskube/distr/commit/cb3c4810bc23b55c3f4b934e0a8c088f584b794d))
* **deps:** update module github.com/go-chi/jwtauth/v5 to v5.3.3 ([#714](https://github.com/glasskube/distr/issues/714)) ([0e3382b](https://github.com/glasskube/distr/commit/0e3382b06d0a2030ab208ddad3c698f3084e001c))


### Other

* **backend:** add indices for artifact blob digest columns ([#717](https://github.com/glasskube/distr/issues/717)) ([23885ce](https://github.com/glasskube/distr/commit/23885cee90725cd5c4e92b6f359869a9cdc47e6f))

## [1.4.3](https://github.com/glasskube/distr/compare/1.4.2...1.4.3) (2025-03-27)


### Bug Fixes

* **deps:** update angular monorepo to v19.2.4 ([#712](https://github.com/glasskube/distr/issues/712)) ([5d7045a](https://github.com/glasskube/distr/commit/5d7045a9f0cc2172c04c5ebb229a183b46c80f71))


### Other

* **deps:** update dependency @types/semver to v7.7.0 ([#708](https://github.com/glasskube/distr/issues/708)) ([208549e](https://github.com/glasskube/distr/commit/208549e8cfe4c52558c8079ed38fd306f65f1889))


### Performance

* **backend:** additional index for faster status sorting ([#713](https://github.com/glasskube/distr/issues/713)) ([1fbabe5](https://github.com/glasskube/distr/commit/1fbabe57bfe2f6a2f02ff63a57fbe89da3ca9f5d))

## [1.4.2](https://github.com/glasskube/distr/compare/1.4.1...1.4.2) (2025-03-26)


### Bug Fixes

* artifact tag limit for organizations without artifacts ([#709](https://github.com/glasskube/distr/issues/709)) ([a40327b](https://github.com/glasskube/distr/commit/a40327b1a21a292f0c3180210e9cd4c0cf541f1f))


### Other

* **registry:** simplify location header ([#707](https://github.com/glasskube/distr/issues/707)) ([3b45bff](https://github.com/glasskube/distr/commit/3b45bff42757748d51617f2c9e2aa66192719868))

## [1.4.1](https://github.com/glasskube/distr/compare/1.4.0...1.4.1) (2025-03-26)


### Bug Fixes

* **deps:** update aws-sdk-go-v2 monorepo ([#688](https://github.com/glasskube/distr/issues/688)) ([8756b28](https://github.com/glasskube/distr/commit/8756b28ea2c22dc72c2af4d45204d87856cf18d7))
* **deps:** update aws-sdk-go-v2 monorepo ([#695](https://github.com/glasskube/distr/issues/695)) ([9e12f9f](https://github.com/glasskube/distr/commit/9e12f9f9c8fd6ee5755e8ff7496dfa4346ad202d))
* **deps:** update dependency @angular/cdk to v19.2.5 ([#673](https://github.com/glasskube/distr/issues/673)) ([877cd37](https://github.com/glasskube/distr/commit/877cd37ff78e63ed73319990e14638454ba44b75))
* **deps:** update dependency @angular/cdk to v19.2.6 ([#675](https://github.com/glasskube/distr/issues/675)) ([884f887](https://github.com/glasskube/distr/commit/884f8870a6c98cbc022c39eda52654ddbc846706))
* **deps:** update dependency @angular/cdk to v19.2.7 ([#706](https://github.com/glasskube/distr/issues/706)) ([041b18a](https://github.com/glasskube/distr/commit/041b18a631e0a7eb2a02db668c6d25efe179673b))
* **deps:** update dependency @sentry/angular to v9.8.0 ([#683](https://github.com/glasskube/distr/issues/683)) ([1a93b2c](https://github.com/glasskube/distr/commit/1a93b2ca4573b2fe6f1f61547d9ca30a91fc538f))
* **deps:** update dependency globe.gl to v2.41.3 ([#698](https://github.com/glasskube/distr/issues/698)) ([ca6c630](https://github.com/glasskube/distr/commit/ca6c63053bb0aa40e11ddcdfc3115a81e15ed137))
* **deps:** update dependency posthog-js to v1.232.4 ([#685](https://github.com/glasskube/distr/issues/685)) ([d9edc40](https://github.com/glasskube/distr/commit/d9edc4009b8d5ec662641d50a303b182332a01ff))
* **deps:** update module github.com/docker/cli to v28.0.3+incompatible ([#694](https://github.com/glasskube/distr/issues/694)) ([66e6926](https://github.com/glasskube/distr/commit/66e692654d01166ff18cea1d669b2f19c4c58d64))
* **deps:** update module github.com/docker/cli to v28.0.4+incompatible ([#696](https://github.com/glasskube/distr/issues/696)) ([8fc222d](https://github.com/glasskube/distr/commit/8fc222d4afeb4e9514afc73ed13092eac45d5e38))
* **deps:** update module github.com/jackc/pgx/v5 to v5.7.3 ([#680](https://github.com/glasskube/distr/issues/680)) ([6f21c24](https://github.com/glasskube/distr/commit/6f21c24e33b886843bf9dc84febdbdb15a845f8a))
* **deps:** update module github.com/jackc/pgx/v5 to v5.7.4 ([#689](https://github.com/glasskube/distr/issues/689)) ([4b98f8e](https://github.com/glasskube/distr/commit/4b98f8e8cfefe91c5b5e3393d7a992b66a66581e))
* **deps:** update module github.com/onsi/gomega to v1.36.3 ([#679](https://github.com/glasskube/distr/issues/679)) ([8bbe99d](https://github.com/glasskube/distr/commit/8bbe99d36e9c4d5bf9a5c36ebe14f48d9026e2b4))
* **ui:** cancel deployment polling when not in use ([#672](https://github.com/glasskube/distr/issues/672)) ([8c1bb25](https://github.com/glasskube/distr/commit/8c1bb25bb3598d41b6f28636ed1df9b510c97658))


### Other

* add registry settings to helm chart ([#693](https://github.com/glasskube/distr/issues/693)) ([2a1ffc9](https://github.com/glasskube/distr/commit/2a1ffc97bca5dd4e74ea6661fbae256aba580e80))
* add registry to deploy docker compose ([#690](https://github.com/glasskube/distr/issues/690)) ([2dc1080](https://github.com/glasskube/distr/commit/2dc1080fa407aba2e73a43f413367df52a23b35a))
* **chart:** set USER_EMAIL_VERIFICATION_REQUIRED=false ([#700](https://github.com/glasskube/distr/issues/700)) ([dd70cb5](https://github.com/glasskube/distr/commit/dd70cb5f15736db4ec88bf438d065a2b06c9f62b))
* **deps:** update dependency @angular-devkit/build-angular to v19.2.5 ([#699](https://github.com/glasskube/distr/issues/699)) ([8cafb22](https://github.com/glasskube/distr/commit/8cafb2203495d14e5b2185f74c44f35beef8ef45))
* **deps:** update dependency @angular/cli to v19.2.5 ([#701](https://github.com/glasskube/distr/issues/701)) ([90c6a5d](https://github.com/glasskube/distr/commit/90c6a5d3f649b8aad4dac7631293cc9dbd2796e4))
* **deps:** update dependency @sentry/cli to v2.42.4 ([#682](https://github.com/glasskube/distr/issues/682)) ([8470240](https://github.com/glasskube/distr/commit/8470240438168c1ed12ea53537f614b95411d046))
* **deps:** update dependency golangci-lint to v2 ([#686](https://github.com/glasskube/distr/issues/686)) ([4558638](https://github.com/glasskube/distr/commit/45586383b483b2d78b6ead2c3adf6292710446db))
* **deps:** update dependency golangci-lint to v2.0.2 ([#697](https://github.com/glasskube/distr/issues/697)) ([a10a04e](https://github.com/glasskube/distr/commit/a10a04ea653c4b6798dff6cd0f211d54b154df6f))
* **deps:** update dependency typedoc-plugin-markdown to v4.6.0 ([#681](https://github.com/glasskube/distr/issues/681)) ([ace69ec](https://github.com/glasskube/distr/commit/ace69ece97911a505f26398d722fb7fbdb6cf9bb))
* **registry:** introduce optional tag limit per organization ([#703](https://github.com/glasskube/distr/issues/703)) ([0452f7c](https://github.com/glasskube/distr/commit/0452f7cdb18538fe6ffbcf0cc5758e3fe71394ae))
* **registry:** record artifact size ([#705](https://github.com/glasskube/distr/issues/705)) ([43801fd](https://github.com/glasskube/distr/commit/43801fd887cea5bbb82844a1071c77a99f395880))
* **registry:** recover and send internal errors to sentry ([#692](https://github.com/glasskube/distr/issues/692)) ([df542d3](https://github.com/glasskube/distr/commit/df542d35ba51bf9b00b7e48bdb6a5740bb295257))
* **registry:** respond with 416 when uploading chunk multiple times ([#676](https://github.com/glasskube/distr/issues/676)) ([e79f8ef](https://github.com/glasskube/distr/commit/e79f8ef02722156d650ee0a227a91be4c50d6124))
* **registry:** support REGISTRY_ENABLED env variable ([#684](https://github.com/glasskube/distr/issues/684)) ([a4dba7d](https://github.com/glasskube/distr/commit/a4dba7dc638bd4afa6073d633ccc0ef7c1a52db4))
* **ui:** change artifact icon to `faBox` ([#702](https://github.com/glasskube/distr/issues/702)) ([f7472fe](https://github.com/glasskube/distr/commit/f7472fed453bd29af09b9ac838704ccc237fd8da))
* **ui:** show registry instructions ([#691](https://github.com/glasskube/distr/issues/691)) ([d8af16f](https://github.com/glasskube/distr/commit/d8af16f938d22dd6e6a98a699106d45d95c6e3e7))

## [1.4.0](https://github.com/glasskube/distr/compare/1.3.3...1.4.0) (2025-03-20)


### Features

* add organization slug and organization settings ([#625](https://github.com/glasskube/distr/issues/625)) ([6031033](https://github.com/glasskube/distr/commit/6031033624ac739ad31f3527c725baf9bc6ce5d1))
* artifact management ([#618](https://github.com/glasskube/distr/issues/618)) ([99ef6c8](https://github.com/glasskube/distr/commit/99ef6c873c8cd35df1071efe842de854d804f889))


### Bug Fixes

* **deps:** update angular monorepo to v19.2.2 ([#628](https://github.com/glasskube/distr/issues/628)) ([21cbed9](https://github.com/glasskube/distr/commit/21cbed9be8b06ac78d2b9bbfbc1b99c0acd0591e))
* **deps:** update angular monorepo to v19.2.3 ([#663](https://github.com/glasskube/distr/issues/663)) ([6b1d5b5](https://github.com/glasskube/distr/commit/6b1d5b54640a9fcbdf68bb2ac30dc415bf1c996e))
* **deps:** update dependency @angular/cdk to v19.2.3 ([#629](https://github.com/glasskube/distr/issues/629)) ([6dafee9](https://github.com/glasskube/distr/commit/6dafee9ff4c23ebe61329741e68ceee2ed6bb3a0))
* **deps:** update dependency @angular/cdk to v19.2.4 ([#662](https://github.com/glasskube/distr/issues/662)) ([3be5834](https://github.com/glasskube/distr/commit/3be58342d603609de987014bc9490a360efa9ac1))
* **deps:** update dependency @codemirror/language to v6.11.0 ([#635](https://github.com/glasskube/distr/issues/635)) ([52e64b2](https://github.com/glasskube/distr/commit/52e64b22dd00ac0b6234cc3064fe1f6d6995f5be))
* **deps:** update dependency @sentry/angular to v9.5.0 ([#616](https://github.com/glasskube/distr/issues/616)) ([09eb54d](https://github.com/glasskube/distr/commit/09eb54da61f3eff7f4fb8e9ae97d20c20a036c0a))
* **deps:** update dependency globe.gl to v2.41.2 ([#668](https://github.com/glasskube/distr/issues/668)) ([4576f37](https://github.com/glasskube/distr/commit/4576f37aaee1978d5d63d1d338280bda7c51d7e5))
* **deps:** update dependency ngx-markdown to v19.1.1 ([#640](https://github.com/glasskube/distr/issues/640)) ([a405104](https://github.com/glasskube/distr/commit/a40510472e1b0e7ab7361fea44d7181fe31dfe85))
* **deps:** update dependency posthog-js to v1.230.1 ([#617](https://github.com/glasskube/distr/issues/617)) ([fd0a8c4](https://github.com/glasskube/distr/commit/fd0a8c45521c68457eb87b9520f54c32f286de98))
* **deps:** update dependency posthog-js to v1.231.0 ([#648](https://github.com/glasskube/distr/issues/648)) ([f44db54](https://github.com/glasskube/distr/commit/f44db54ecf6c326a6ff073aaad5b7a3953763534))
* **deps:** update kubernetes packages to v0.32.3 ([#621](https://github.com/glasskube/distr/issues/621)) ([e5fe5aa](https://github.com/glasskube/distr/commit/e5fe5aafa4f5b7bbe01403c33e27643c5877c8d4))
* **deps:** update module github.com/aws/aws-sdk-go-v2/service/s3 to v1.78.2 ([#671](https://github.com/glasskube/distr/issues/671)) ([fe1992b](https://github.com/glasskube/distr/commit/fe1992b551dd0917cc9bc2e02ddfb78cb6cc5b20))
* **deps:** update module github.com/docker/cli to v28.0.2+incompatible ([#664](https://github.com/glasskube/distr/issues/664)) ([3eb0aa1](https://github.com/glasskube/distr/commit/3eb0aa19098f3e24140e2b17793ca438be2b40d4))
* **deps:** update module helm.sh/helm/v3 to v3.17.2 ([#636](https://github.com/glasskube/distr/issues/636)) ([6d52afb](https://github.com/glasskube/distr/commit/6d52afb3f37194dddfbe3a8319b6fc8b68b9aa5e))


### Other

* **deps:** update actions/setup-go action to v5.4.0 ([#656](https://github.com/glasskube/distr/issues/656)) ([980f9eb](https://github.com/glasskube/distr/commit/980f9eb615fe5b7e7f6c9035ad124222cefd3242))
* **deps:** update actions/setup-node action to v4.3.0 ([#647](https://github.com/glasskube/distr/issues/647)) ([a8daaf3](https://github.com/glasskube/distr/commit/a8daaf347b75eeab99a8ff35e84cdd5ad01468c0))
* **deps:** update angular-cli monorepo to v19.2.2 ([#630](https://github.com/glasskube/distr/issues/630)) ([19a94b9](https://github.com/glasskube/distr/commit/19a94b923cd49401d36f4bee84bfcade0c99f5ce))
* **deps:** update angular-cli monorepo to v19.2.3 ([#631](https://github.com/glasskube/distr/issues/631)) ([8aa2707](https://github.com/glasskube/distr/commit/8aa270726f40b1605884ba8539b53468f1f75d69))
* **deps:** update angular-cli monorepo to v19.2.4 ([#661](https://github.com/glasskube/distr/issues/661)) ([350dfb3](https://github.com/glasskube/distr/commit/350dfb35bdcd9a925ea591a0543c57de4838073a))
* **deps:** update axllent/mailpit docker tag to v1.23.1 ([#613](https://github.com/glasskube/distr/issues/613)) ([f0cbd3b](https://github.com/glasskube/distr/commit/f0cbd3b5692fc1b720548228d07453d16b288a9b))
* **deps:** update axllent/mailpit docker tag to v1.23.2 ([#644](https://github.com/glasskube/distr/issues/644)) ([4d123ab](https://github.com/glasskube/distr/commit/4d123ab192175f456fceb8cdcb91d4d839a6dc10))
* **deps:** update dependency @sentry/cli to v2.42.3 ([#646](https://github.com/glasskube/distr/issues/646)) ([a44a1a0](https://github.com/glasskube/distr/commit/a44a1a061e4a8627718f2073f9662dfcc79236bb))
* **deps:** update dependency autoprefixer to v10.4.21 ([#615](https://github.com/glasskube/distr/issues/615)) ([4c80bb2](https://github.com/glasskube/distr/commit/4c80bb2197ed8a593991c497c01a14e808c0e784))
* **deps:** update dependency golangci-lint to v1.64.7 ([#620](https://github.com/glasskube/distr/issues/620)) ([8f07438](https://github.com/glasskube/distr/commit/8f074384a8e4e64fde235ded157df3700315dd11))
* **deps:** update dependency golangci-lint to v1.64.8 ([#651](https://github.com/glasskube/distr/issues/651)) ([d7e93d9](https://github.com/glasskube/distr/commit/d7e93d9f3f5d8dc7ea135b082cbf0a0c5937da12))
* **deps:** update dependency typedoc to ^0.28.0 ([#643](https://github.com/glasskube/distr/issues/643)) ([b17706c](https://github.com/glasskube/distr/commit/b17706ca6783e0871f5fa2f8f01f0f91a3b2e1bd))
* **deps:** update dependency typedoc to v0.28.1 ([#665](https://github.com/glasskube/distr/issues/665)) ([e05d997](https://github.com/glasskube/distr/commit/e05d9976bdb8a25ab385d279e96d72ccd6ea70bd))
* **deps:** update dependency typedoc-plugin-markdown to v4.5.1 ([#655](https://github.com/glasskube/distr/issues/655)) ([a4c596e](https://github.com/glasskube/distr/commit/a4c596ef5f8f344b84bd739cba3b87fb51565670))
* **deps:** update dependency typedoc-plugin-markdown to v4.5.2 ([#657](https://github.com/glasskube/distr/issues/657)) ([68484e5](https://github.com/glasskube/distr/commit/68484e53904d1f64039054d382e9f749e65ee380))
* **deps:** update docker/login-action action to v3.4.0 ([#637](https://github.com/glasskube/distr/issues/637)) ([59479c3](https://github.com/glasskube/distr/commit/59479c3c4590893444b76f614bd7d304e30cc2a0))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to b35229a ([#654](https://github.com/glasskube/distr/issues/654)) ([f6c324c](https://github.com/glasskube/distr/commit/f6c324c6a5d5aaa76c8e71659437ae9db23213ce))
* **deps:** update golangci/golangci-lint-action action to v6.5.1 ([#627](https://github.com/glasskube/distr/issues/627)) ([65b79e5](https://github.com/glasskube/distr/commit/65b79e5a3b3272ffda56b488181237596bf071f6))
* **deps:** update golangci/golangci-lint-action action to v6.5.2 ([#652](https://github.com/glasskube/distr/issues/652)) ([fff8bb0](https://github.com/glasskube/distr/commit/fff8bb08078ce896af2381a0efa58b04a8747404))
* **deps:** update googleapis/release-please-action action to v4.2.0 ([#612](https://github.com/glasskube/distr/issues/612)) ([bd4520c](https://github.com/glasskube/distr/commit/bd4520cb2a61f7093d14deeaf81b40b4caa3863d))
* **deps:** upgrade prismjs to v1.30.0 ([#634](https://github.com/glasskube/distr/issues/634)) ([72d93be](https://github.com/glasskube/distr/commit/72d93be0401ceebc124fa9a0fcd7ad09872d4db7))
* improve copy instructions ux ([#638](https://github.com/glasskube/distr/issues/638)) ([d8d04ef](https://github.com/glasskube/distr/commit/d8d04efae944505ad0bb562a710440b23360e832))

## [1.3.3](https://github.com/glasskube/distr/compare/1.3.2...1.3.3) (2025-03-07)


### Bug Fixes

* **deps:** update angular monorepo to v19.2.1 ([#603](https://github.com/glasskube/distr/issues/603)) ([8fd5e45](https://github.com/glasskube/distr/commit/8fd5e45e5deb1d221ca5eaab7dd1fc8b1fe972a7))
* **deps:** update aws-sdk-go-v2 monorepo ([#600](https://github.com/glasskube/distr/issues/600)) ([75a1df1](https://github.com/glasskube/distr/commit/75a1df13ed52f577136851e79138f30b56a12a7d))
* **deps:** update dependency @angular/cdk to v19.2.2 ([#607](https://github.com/glasskube/distr/issues/607)) ([e0d4d0a](https://github.com/glasskube/distr/commit/e0d4d0a2c64dafd659089c5c3bd13986a3590082))
* **deps:** update dependency @codemirror/view to v6.36.4 ([#595](https://github.com/glasskube/distr/issues/595)) ([406f656](https://github.com/glasskube/distr/commit/406f656d9a47c090612ceb7e4fab7e1c20451368))
* **deps:** update module golang.org/x/crypto to v0.36.0 ([#604](https://github.com/glasskube/distr/issues/604)) ([10de92f](https://github.com/glasskube/distr/commit/10de92f10fc6c8db1f3687774dc4bf7665afdb1a))
* only return licensed application if feature is active ([#611](https://github.com/glasskube/distr/issues/611)) ([92cd1b9](https://github.com/glasskube/distr/commit/92cd1b9618b2da68002225c0c52ee384b12ad9fd))


### Other

* **deps:** update angular-cli monorepo to v19.2.1 ([#606](https://github.com/glasskube/distr/issues/606)) ([5916bc1](https://github.com/glasskube/distr/commit/5916bc1a9d0b7ba9bfc6a66cfa6d67430577c4ec))
* **deps:** update dependency go to v1.24.1 ([#599](https://github.com/glasskube/distr/issues/599)) ([e9c4fb3](https://github.com/glasskube/distr/commit/e9c4fb3300ac1fb869b4f4f27b2e12577f61ff1f))
* **deps:** update googleapis/release-please-action action to v4.1.5 ([#605](https://github.com/glasskube/distr/issues/605)) ([9314995](https://github.com/glasskube/distr/commit/9314995762da27a3053cb61e0e95c7cb85bbfd4c))
* support registration mode via environment ([#608](https://github.com/glasskube/distr/issues/608)) ([dd9f747](https://github.com/glasskube/distr/commit/dd9f74702298c4eb77124fe9e958c90d1c0325df))
* **ui:** preview docker compose in deploy form ([#610](https://github.com/glasskube/distr/issues/610)) ([cf293fc](https://github.com/glasskube/distr/commit/cf293fc93448fc4abed8dfafd4951cbcf747acd4))
* **ui:** require additional input when deleting deployments ([#598](https://github.com/glasskube/distr/issues/598)) ([c6a8a5f](https://github.com/glasskube/distr/commit/c6a8a5f075bcda3c183ca1194798ade6fe6570ef))

## [1.3.2](https://github.com/glasskube/distr/compare/1.3.1...1.3.2) (2025-03-03)


### Bug Fixes

* avoid settings override by using invite link multiple times ([#579](https://github.com/glasskube/distr/issues/579)) ([2b738ce](https://github.com/glasskube/distr/commit/2b738ce1877407fce1d07507cc7d781ba88ebfeb))
* **deps:** update aws-sdk-go-v2 monorepo ([#577](https://github.com/glasskube/distr/issues/577)) ([8f64b98](https://github.com/glasskube/distr/commit/8f64b9843b7c951a516164e279d7ddf81158a878))
* **deps:** update dependency @sentry/angular to v9.3.0 ([#584](https://github.com/glasskube/distr/issues/584)) ([6e9d696](https://github.com/glasskube/distr/commit/6e9d696a1e08f74afeeff941fda19f9265aa47ea))
* **deps:** update dependency globe.gl to v2.41.1 ([#590](https://github.com/glasskube/distr/issues/590)) ([6a7d614](https://github.com/glasskube/distr/commit/6a7d6147db70d64e3bfbfd27822b79b64f635c1e))
* **deps:** update dependency posthog-js to v1.224.1 ([#580](https://github.com/glasskube/distr/issues/580)) ([69f5945](https://github.com/glasskube/distr/commit/69f594570e2a724a8fa932096f53a3df3446a16b))
* **deps:** update dependency posthog-js to v1.225.0 ([#589](https://github.com/glasskube/distr/issues/589)) ([22e80bf](https://github.com/glasskube/distr/commit/22e80bf6c8a47ead8b6c4eb63ed69130ba6eed16))
* **deps:** update fontsource monorepo to v5.2.5 ([#588](https://github.com/glasskube/distr/issues/588)) ([6e9dfef](https://github.com/glasskube/distr/commit/6e9dfef881fd814db40e9d40544b89d03e988f55))


### Other

* **backend:** add `SERVER_SHUTDOWN_DELAY_DURATION` config variable ([#583](https://github.com/glasskube/distr/issues/583)) ([5cfd73f](https://github.com/glasskube/distr/commit/5cfd73ff8ebe1619445ca92161817749e1d4155f))
* **backend:** application of existing deployment should not be changeable ([#578](https://github.com/glasskube/distr/issues/578)) ([8155538](https://github.com/glasskube/distr/commit/81555381d636e861d470966e0e8de87738344448))
* **deps:** update axllent/mailpit docker tag to v1.23.0 ([#587](https://github.com/glasskube/distr/issues/587)) ([4c831e3](https://github.com/glasskube/distr/commit/4c831e3d8bcdd05235f921ab048d4d36c6a4ef11))
* **deps:** update dependency golangci-lint to v1.64.6 ([#591](https://github.com/glasskube/distr/issues/591)) ([f98b6e6](https://github.com/glasskube/distr/commit/f98b6e6613640ebe161f01e1b92958fba14fe49a))
* **deps:** update dependency prettier to v3.5.3 ([#592](https://github.com/glasskube/distr/issues/592)) ([9c089c0](https://github.com/glasskube/distr/commit/9c089c04e95da8e8ef57ae072400c70beecdfd1a))
* **deps:** update dependency typescript to ~5.8.0 ([#586](https://github.com/glasskube/distr/issues/586)) ([01a0cb6](https://github.com/glasskube/distr/commit/01a0cb6d9d4ee1d095eea79a2d6cc23b74aa12e0))
* **deps:** update dependency typescript to v5.8.2 ([#585](https://github.com/glasskube/distr/issues/585)) ([8825de8](https://github.com/glasskube/distr/commit/8825de83a5c4c98e61b3439d8730e623110b9496))
* **deps:** update googleapis/release-please-action action to v4.1.4 ([#576](https://github.com/glasskube/distr/issues/576)) ([6541edd](https://github.com/glasskube/distr/commit/6541edd4e37f106eda5c4051c7860cf5de43aa66))
* **ui:** additional routing fallbacks ([#593](https://github.com/glasskube/distr/issues/593)) ([a0f7506](https://github.com/glasskube/distr/commit/a0f7506a799a6f282d4d2323d61136b61134ff3b))
* **ui:** editor should not always expect yaml ([#581](https://github.com/glasskube/distr/issues/581)) ([8f6758e](https://github.com/glasskube/distr/commit/8f6758e911f3715cfa89ddb8daad6ce7b3bfad40))
* **ui:** make cache resettable ([#574](https://github.com/glasskube/distr/issues/574)) ([06ea2af](https://github.com/glasskube/distr/commit/06ea2afe60f801365ef368adf27b1d1a825e56a7))
* **ui:** show app version id in table ([#582](https://github.com/glasskube/distr/issues/582)) ([a5a1f82](https://github.com/glasskube/distr/commit/a5a1f82b20e869ce5183b82d8e601550002e600a))

## [1.3.1](https://github.com/glasskube/distr/compare/1.3.0...1.3.1) (2025-02-27)


### Bug Fixes

* **backend:** improve graceful shutdown procedure ([#573](https://github.com/glasskube/distr/issues/573)) ([8085698](https://github.com/glasskube/distr/commit/80856985ee486be78e50ade0f4933374dc563934))
* **deps:** update angular monorepo to v19.2.0 ([#570](https://github.com/glasskube/distr/issues/570)) ([2728480](https://github.com/glasskube/distr/commit/27284807fd7778dd99404e4c1e06acfa94f84941))
* **deps:** update dependency @angular/cdk to v19.2.1 ([#571](https://github.com/glasskube/distr/issues/571)) ([1a704c9](https://github.com/glasskube/distr/commit/1a704c926289901370afa83049d35313f7a70577))
* **deps:** update dependency posthog-js to v1.223.4 ([#561](https://github.com/glasskube/distr/issues/561)) ([6f7df58](https://github.com/glasskube/distr/commit/6f7df584249f919fe214b38d7921f4c38ef3b881))
* **deps:** update dependency posthog-js to v1.223.5 ([#564](https://github.com/glasskube/distr/issues/564)) ([718b597](https://github.com/glasskube/distr/commit/718b597dbfeca53d51b6785786afcebc536fca84))
* **deps:** update dependency posthog-js to v1.224.0 ([#572](https://github.com/glasskube/distr/issues/572)) ([d8048f9](https://github.com/glasskube/distr/commit/d8048f928285278fb057c97d7d59812feed5ed4d))
* **deps:** update module github.com/docker/cli to v28.0.1+incompatible ([#565](https://github.com/glasskube/distr/issues/565)) ([543f7c1](https://github.com/glasskube/distr/commit/543f7c11129ad6e685c3954cc56d2584f73940ea))


### Other

* **deps:** update angular-cli monorepo to v19.2.0 ([#566](https://github.com/glasskube/distr/issues/566)) ([c936eb6](https://github.com/glasskube/distr/commit/c936eb67ac405c66bb53b779312e6606f0579956))
* **deps:** update docker/build-push-action action to v6.15.0 ([#567](https://github.com/glasskube/distr/issues/567)) ([d619b2a](https://github.com/glasskube/distr/commit/d619b2afdf942890b2ba1e40a2fbab59c7c13848))
* **deps:** update docker/metadata-action action to v5.7.0 ([#568](https://github.com/glasskube/distr/issues/568)) ([37481c8](https://github.com/glasskube/distr/commit/37481c8141e5fb38c0bb790b78cb96f4250f02f6))
* **deps:** update docker/setup-buildx-action action to v3.10.0 ([#569](https://github.com/glasskube/distr/issues/569)) ([0c1abbb](https://github.com/glasskube/distr/commit/0c1abbb44a53f4ae73e3c289d32226b63570365e))

## [1.3.0](https://github.com/glasskube/distr/compare/1.2.1...1.3.0) (2025-02-25)


### Features

* add "undeploying" applications from a target ([#515](https://github.com/glasskube/distr/issues/515)) ([8a86e0b](https://github.com/glasskube/distr/commit/8a86e0be925c18620ae08597aa4789db6113cb6e))
* add application version archiving ([#520](https://github.com/glasskube/distr/issues/520)) ([cdd09e2](https://github.com/glasskube/distr/commit/cdd09e2bc519c24111761b4ce4243a3c793ee6c2))
* **frontend:** add indicator if update is available ([#550](https://github.com/glasskube/distr/issues/550)) ([d87feb1](https://github.com/glasskube/distr/commit/d87feb12189a7a0f62df7b3b4f100566df7b5a34))
* **ui:** add beta feature activation tooltip ([#556](https://github.com/glasskube/distr/issues/556)) ([e5a5d52](https://github.com/glasskube/distr/commit/e5a5d5286f05a15ae439a23b682772c62a0dc27c))


### Bug Fixes

* **deps:** update dependency @sentry/angular to v9.2.0 ([#553](https://github.com/glasskube/distr/issues/553)) ([ffa8dec](https://github.com/glasskube/distr/commit/ffa8dec47131d32e8c85189cf9f0500738f76037))
* **deps:** update dependency posthog-js to v1.221.0 ([#539](https://github.com/glasskube/distr/issues/539)) ([b658f50](https://github.com/glasskube/distr/commit/b658f50de40aca40e3d58caff2bf189a591c0ede))
* **deps:** update dependency posthog-js to v1.222.0 ([#541](https://github.com/glasskube/distr/issues/541)) ([a124c97](https://github.com/glasskube/distr/commit/a124c97c91934c2eb64c5447e0e0cc7539af3ff5))
* **deps:** update dependency posthog-js to v1.223.3 ([#548](https://github.com/glasskube/distr/issues/548)) ([e3d615c](https://github.com/glasskube/distr/commit/e3d615c28421b0a5bb3668a64c51a77e7d3a70d9))
* **deps:** update dependency rxjs to v7.8.2 ([#545](https://github.com/glasskube/distr/issues/545)) ([ef910c2](https://github.com/glasskube/distr/commit/ef910c25023f25b72bc24cf9f03f4070b3fcc1d0))
* **deps:** update module github.com/docker/cli to v28 ([#533](https://github.com/glasskube/distr/issues/533)) ([ae39374](https://github.com/glasskube/distr/commit/ae393740804d11bc4952a69c823e482bc0b7d46d))
* **deps:** update module github.com/lestrrat-go/jwx/v2 to v2.1.4 ([#559](https://github.com/glasskube/distr/issues/559)) ([f9ca0d1](https://github.com/glasskube/distr/commit/f9ca0d1b5077ff43c76b5cf3efa0ff9d09cd3ada))
* **deps:** update module golang.org/x/crypto to v0.34.0 ([#547](https://github.com/glasskube/distr/issues/547)) ([07605e2](https://github.com/glasskube/distr/commit/07605e286803a3ddea12dd4c50ef3ae4e0929615))
* **deps:** update module golang.org/x/crypto to v0.35.0 ([#554](https://github.com/glasskube/distr/issues/554)) ([493203e](https://github.com/glasskube/distr/commit/493203eae8a9757e75e753126ead0b792c923050))
* **frontend:** add deployed application as cell title on deployment targets page ([#557](https://github.com/glasskube/distr/issues/557)) ([fd5e62c](https://github.com/glasskube/distr/commit/fd5e62c5b84d36e2ec84ddd9544487debf04e40a))
* **frontend:** use correct deployments link on globe marker ([#549](https://github.com/glasskube/distr/issues/549)) ([f3bfa4b](https://github.com/glasskube/distr/commit/f3bfa4b61f2490772973d7e7238c03b9a3e22840))
* **ui:** make autotrim work with FormControls ([#535](https://github.com/glasskube/distr/issues/535)) ([6b4c994](https://github.com/glasskube/distr/commit/6b4c994a9aba9466a16ba058bbb5dc8e11990ef3))


### Other

* **backend:** add application version name unique constraint ([#536](https://github.com/glasskube/distr/issues/536)) ([4400491](https://github.com/glasskube/distr/commit/4400491052dd34f3012bc738fe9e7dadbd888be9))
* **backend:** change DeploymentRevision application_version_id reference on delete to `RESTRICT` ([#528](https://github.com/glasskube/distr/issues/528)) ([0ed852f](https://github.com/glasskube/distr/commit/0ed852fc9801dbcce4e4c2d6756913e69c5ceeb9))
* **deps:** update dependency @sentry/cli to v2.42.2 ([#551](https://github.com/glasskube/distr/issues/551)) ([a0abeb3](https://github.com/glasskube/distr/commit/a0abeb30115fbb3e70f096bf5244bdd28e220ad5))
* **deps:** update dependency @types/jasmine to v5.1.7 ([#542](https://github.com/glasskube/distr/issues/542)) ([ca6bd8e](https://github.com/glasskube/distr/commit/ca6bd8e1a6e58151bb7820def71d177784b4c629))
* **deps:** update dependency prettier to v3.5.2 ([#544](https://github.com/glasskube/distr/issues/544)) ([e534237](https://github.com/glasskube/distr/commit/e5342379e12f51716de531c303663b9d7f2c72ee))
* **deps:** update dependency typedoc to v0.27.8 ([#543](https://github.com/glasskube/distr/issues/543)) ([465ac60](https://github.com/glasskube/distr/commit/465ac60d04062d3576a2b09cdca4c7a35c14bbf9))
* **deps:** update dependency typedoc to v0.27.9 ([#555](https://github.com/glasskube/distr/issues/555)) ([6075168](https://github.com/glasskube/distr/commit/607516804ce25636746f88ec7e3d8f88f7099b8e))
* **deps:** update postgres docker tag to v17.4 ([#546](https://github.com/glasskube/distr/issues/546)) ([2077ded](https://github.com/glasskube/distr/commit/2077ded593fa224c8a348e0a2048dd8c2003ce08))
* **frontend:** add disabling "undeploy" button if reported agent version is too old ([#560](https://github.com/glasskube/distr/issues/560)) ([4e77d05](https://github.com/glasskube/distr/commit/4e77d05b289a6e10f5226ae6bd4bb750a1c5236d))
* **frontend:** change table header "owner" to "customer" ([#540](https://github.com/glasskube/distr/issues/540)) ([538bbec](https://github.com/glasskube/distr/commit/538bbec8b7e89aee7c0f37e6b406525bd3feccab))


### Docs

* **repo:** update readme to include macOS guide ([#558](https://github.com/glasskube/distr/issues/558)) ([5352a5a](https://github.com/glasskube/distr/commit/5352a5afe138518e75d58cabcb79583552031fd4))

## [1.2.1](https://github.com/glasskube/distr/compare/1.2.0...1.2.1) (2025-02-20)


### Bug Fixes

* **deps:** update dependency @angular/cdk to v19.1.5 ([#531](https://github.com/glasskube/distr/issues/531)) ([5db3911](https://github.com/glasskube/distr/commit/5db391165ad11d8b46ac4d1b50d2327df26fbd45))

## [1.2.0](https://github.com/glasskube/distr/compare/1.1.6...1.2.0) (2025-02-20)


### Features

* add agents handling registry auth from license  ([#510](https://github.com/glasskube/distr/issues/510)) ([c9ddc78](https://github.com/glasskube/distr/commit/c9ddc7848c2f5f18ea7f009875572e0a908228e1))
* add selecting a license for a deployment ([#463](https://github.com/glasskube/distr/issues/463)) ([f68aeee](https://github.com/glasskube/distr/commit/f68aeeef47ddcb1781cdabbb3bbda107ce04fd4b))
* license management ([#476](https://github.com/glasskube/distr/issues/476)) ([ea0cad2](https://github.com/glasskube/distr/commit/ea0cad2bd188966acbf76ca4b8990b84b144b191))


### Bug Fixes

* **backend:** send correct deployment ID to agents ([#526](https://github.com/glasskube/distr/issues/526)) ([443ae90](https://github.com/glasskube/distr/commit/443ae907a9213ec0a4e49b630bc50b101ba750a9))
* **deps:** update angular monorepo to v19.1.7 ([#530](https://github.com/glasskube/distr/issues/530)) ([1f9041d](https://github.com/glasskube/distr/commit/1f9041d9ee24b3d65f87c1ee7ac82bfdff2a6862))
* **deps:** update aws-sdk-go-v2 monorepo ([#517](https://github.com/glasskube/distr/issues/517)) ([3195363](https://github.com/glasskube/distr/commit/31953637aeae5e9ffbadc828d28cdfcc3c030c7f))
* **deps:** update dependency @codemirror/view to v6.36.3 ([#506](https://github.com/glasskube/distr/issues/506)) ([8a4635e](https://github.com/glasskube/distr/commit/8a4635e50f55ab5994cc9f492c6b0ed5ded6c384))
* **deps:** update dependency apexcharts to v4.5.0 ([#522](https://github.com/glasskube/distr/issues/522)) ([fc47bf4](https://github.com/glasskube/distr/commit/fc47bf43f0f2102b5e153fbaf5f3e6a16a77fc68))
* **deps:** update dependency globe.gl to v2.40.0 ([#519](https://github.com/glasskube/distr/issues/519)) ([d685b07](https://github.com/glasskube/distr/commit/d685b07c7b43e65ff1c2f4ac8d6c0053b2e7b6f1))
* **deps:** update dependency posthog-js to v1.217.6 ([#497](https://github.com/glasskube/distr/issues/497)) ([d95131f](https://github.com/glasskube/distr/commit/d95131f73641e711c1fc1d58359e512059ffc191))
* **deps:** update dependency posthog-js to v1.218.2 ([#500](https://github.com/glasskube/distr/issues/500)) ([01eacdb](https://github.com/glasskube/distr/commit/01eacdb24b16e901c8354010d4d95585c7a0a88d))
* **deps:** update dependency posthog-js to v1.219.2 ([#505](https://github.com/glasskube/distr/issues/505)) ([09ebe68](https://github.com/glasskube/distr/commit/09ebe6882199f26223709fcd54458660fa2d7496))
* **deps:** update dependency posthog-js to v1.219.3 ([#508](https://github.com/glasskube/distr/issues/508)) ([4a8b27b](https://github.com/glasskube/distr/commit/4a8b27bd48f499a4cc292e98f8cdc09c99dd5bcb))
* **deps:** update dependency posthog-js to v1.219.4 ([#518](https://github.com/glasskube/distr/issues/518)) ([31dc7f9](https://github.com/glasskube/distr/commit/31dc7f95dedf02196392e82e3b054f03f7751c85))
* **deps:** update dependency posthog-js to v1.219.6 ([#524](https://github.com/glasskube/distr/issues/524)) ([0f28529](https://github.com/glasskube/distr/commit/0f285295d6cc6d12a30bde1db45680aa8d988297))
* **deps:** update module github.com/docker/cli to v27.5.1+incompatible ([#514](https://github.com/glasskube/distr/issues/514)) ([2b9058f](https://github.com/glasskube/distr/commit/2b9058f13d640a595f46ec02fcdb7f4df519894f))
* **deps:** update module github.com/wneessen/go-mail to v0.6.2 ([#503](https://github.com/glasskube/distr/issues/503)) ([143d34f](https://github.com/glasskube/distr/commit/143d34f8d2030a52ce84abec8531e4138f1f5c86))
* fix error on organization branding creation ([#523](https://github.com/glasskube/distr/issues/523)) ([00ee0a8](https://github.com/glasskube/distr/commit/00ee0a87af57e157c0a9ce788b1ee76421d10268))


### Other

* add SECURITY.md ([#504](https://github.com/glasskube/distr/issues/504)) ([6c13bc8](https://github.com/glasskube/distr/commit/6c13bc80f14cb5b2b591f047efac7922a9726026))
* **deps:** update angular-cli monorepo to v19.1.8 ([#532](https://github.com/glasskube/distr/issues/532)) ([ea39302](https://github.com/glasskube/distr/commit/ea39302a58f28d36ca56c6b37be16aef36fefd2f))
* **deps:** update axllent/mailpit docker tag to v1.22.3 ([#502](https://github.com/glasskube/distr/issues/502)) ([de83857](https://github.com/glasskube/distr/commit/de83857a47ab72773a58ba2353c18f06cf5ead87))
* **deps:** update azure/setup-helm action to v4.3.0 ([#507](https://github.com/glasskube/distr/issues/507)) ([0f6ace1](https://github.com/glasskube/distr/commit/0f6ace185d24938e02dce4c69bb9a225ccb85c96))
* **deps:** update dependency @sentry/cli to v2.42.0 ([#509](https://github.com/glasskube/distr/issues/509)) ([e38093f](https://github.com/glasskube/distr/commit/e38093f617ddecbf2b2a35c20d83439740ad68bf))
* **deps:** update dependency @sentry/cli to v2.42.1 ([#513](https://github.com/glasskube/distr/issues/513)) ([9450635](https://github.com/glasskube/distr/commit/94506350547044e6e6e1d212a61c19a8d15d4b43))
* **deps:** update dependency postcss to v8.5.3 ([#527](https://github.com/glasskube/distr/issues/527)) ([c150499](https://github.com/glasskube/distr/commit/c150499c7f7f8f52a7610adf0234b2113bf1b42d))
* **deps:** update docker/build-push-action action to v6.14.0 ([#529](https://github.com/glasskube/distr/issues/529)) ([4d5a049](https://github.com/glasskube/distr/commit/4d5a049f4ae6fac7a22a5b2e27a3b7f41b76281f))
* **deps:** update dompurify to 3.2.4 ([#512](https://github.com/glasskube/distr/issues/512)) ([d078695](https://github.com/glasskube/distr/commit/d0786954300980a8baa942310a309f5510a091c0))
* **deps:** update golangci/golangci-lint-action action to v6.4.1 ([#499](https://github.com/glasskube/distr/issues/499)) ([724abeb](https://github.com/glasskube/distr/commit/724abeb48bccf0fb031efab036fcf8b14a1a8c57))
* **deps:** update golangci/golangci-lint-action action to v6.5.0 ([#501](https://github.com/glasskube/distr/issues/501)) ([0b5a180](https://github.com/glasskube/distr/commit/0b5a180b0bcdd1ddacd15c0274a53dee93be7002))
* improve invite form auto-fill behavior ([#521](https://github.com/glasskube/distr/issues/521)) ([c20ea22](https://github.com/glasskube/distr/commit/c20ea22a46189d4871c6106eda94b96869de286d))
* resolve some code scanning alerts ([#511](https://github.com/glasskube/distr/issues/511)) ([fed0199](https://github.com/glasskube/distr/commit/fed0199e1e96afe384bebeb205a972bc4c05391b))
* send http exceptions to Sentry ([#525](https://github.com/glasskube/distr/issues/525)) ([31ef7da](https://github.com/glasskube/distr/commit/31ef7da48873f68060c922711cbfd0dbef056d07))

## [1.1.6](https://github.com/glasskube/distr/compare/1.1.5...1.1.6) (2025-02-14)


### Bug Fixes

* **backend:** deployment target update respects empty agent version ([#496](https://github.com/glasskube/distr/issues/496)) ([d8d954a](https://github.com/glasskube/distr/commit/d8d954a9965134387fe44714287dca82f9cae459))
* **deps:** update dependency @sentry/angular to v9.1.0 ([#493](https://github.com/glasskube/distr/issues/493)) ([53b657d](https://github.com/glasskube/distr/commit/53b657de439d8e528b89aa32a7c295008c0a9074))
* **deps:** update dependency posthog-js to v1.217.4 ([#485](https://github.com/glasskube/distr/issues/485)) ([a6f9922](https://github.com/glasskube/distr/commit/a6f9922eac4b8093696cf6233ff173ab4577c272))
* **deps:** update dependency posthog-js to v1.217.5 ([#495](https://github.com/glasskube/distr/issues/495)) ([292df16](https://github.com/glasskube/distr/commit/292df16434510f7bf8a322e69299338e7b6cd01d))
* **deps:** update kubernetes packages to v0.32.2 ([#486](https://github.com/glasskube/distr/issues/486)) ([ec950ce](https://github.com/glasskube/distr/commit/ec950cec66edfa0f2efd7d7575b508e4b8c12502))
* **ui:** deploy modal scroll ([#494](https://github.com/glasskube/distr/issues/494)) ([0cac96f](https://github.com/glasskube/distr/commit/0cac96f7d6c63bc3a876e6b88667f410c0d0a08b))


### Other

* **backend:** increase per-second API rate limit from 5 to 10 ([#481](https://github.com/glasskube/distr/issues/481)) ([b14f8f2](https://github.com/glasskube/distr/commit/b14f8f2459aace5ab767856c9b8bcf2548b5734e))
* **deps:** update dependency @types/jasmine to v5.1.6 ([#489](https://github.com/glasskube/distr/issues/489)) ([dae9719](https://github.com/glasskube/distr/commit/dae9719f50b11c0ab3e87588fa7e50f7f35b12ca))
* **deps:** update dependency golangci-lint to v1.64.5 ([#490](https://github.com/glasskube/distr/issues/490)) ([6c27927](https://github.com/glasskube/distr/commit/6c279275f1060e71aa8528cfdd1e093a76bb8e4d))
* **deps:** update dependency prettier to v3.5.1 ([#482](https://github.com/glasskube/distr/issues/482)) ([41314e7](https://github.com/glasskube/distr/commit/41314e77b4b505402de536aec972ef1099a4cb6e))
* **deps:** update golangci/golangci-lint-action action to v6.3.3 ([#483](https://github.com/glasskube/distr/issues/483)) ([9d48e2b](https://github.com/glasskube/distr/commit/9d48e2bda0f544630b1c866f74f54e420eacdc5c))
* **deps:** update golangci/golangci-lint-action action to v6.4.0 ([#491](https://github.com/glasskube/distr/issues/491)) ([9f58ad9](https://github.com/glasskube/distr/commit/9f58ad9f8ba05aca893320f30c02e766b5058ac5))
* **deps:** update postgres docker tag to v17.3 ([#492](https://github.com/glasskube/distr/issues/492)) ([aa2ab6b](https://github.com/glasskube/distr/commit/aa2ab6bd817640cd4cc4e388a701e114aeea9f37))

## [1.1.5](https://github.com/glasskube/distr/compare/1.1.4...1.1.5) (2025-02-13)


### Bug Fixes

* **deps:** update module helm.sh/helm/v3 to v3.17.1 ([#478](https://github.com/glasskube/distr/issues/478)) ([2c2d77f](https://github.com/glasskube/distr/commit/2c2d77fe230733f6585c369905f8cfa7f416b9d3))


### Other

* **deps:** update dependency go to v1.24.0 ([#466](https://github.com/glasskube/distr/issues/466)) ([9c9b271](https://github.com/glasskube/distr/commit/9c9b27145d2ff2c43e0f59bdae7774dd6604c99e))
* **deps:** update golang docker tag to v1.24 ([#479](https://github.com/glasskube/distr/issues/479)) ([7863713](https://github.com/glasskube/distr/commit/786371356044ed6d7aa97e070b554877978ed83e))


### Performance

* **backend:** optimize database query for deployment target with latest status ([#477](https://github.com/glasskube/distr/issues/477)) ([39f6172](https://github.com/glasskube/distr/commit/39f61724ec98f04bdf354977e18a47bc9f1cd66a))

## [1.1.4](https://github.com/glasskube/distr/compare/1.1.3...1.1.4) (2025-02-13)


### Features

* **backend:** add db models for application licenses ([#455](https://github.com/glasskube/distr/issues/455)) ([2ece672](https://github.com/glasskube/distr/commit/2ece672b87e6c1a4919ad3de93fb0ceaff8a9f85))
* license management feature flag ([#458](https://github.com/glasskube/distr/issues/458)) ([1f3fede](https://github.com/glasskube/distr/commit/1f3fede701a3714b02a52b1626f1dee98fc2a72e))


### Bug Fixes

* **deps:** update angular monorepo to v19.1.6 ([#472](https://github.com/glasskube/distr/issues/472)) ([52cb2a8](https://github.com/glasskube/distr/commit/52cb2a8561cfe9f761babdf8cb1b927f2f7d932d))
* **deps:** update dependency @angular/cdk to v19.1.4 ([#475](https://github.com/glasskube/distr/issues/475)) ([3a1f39e](https://github.com/glasskube/distr/commit/3a1f39e708f37a788a9e5f430896ed27faae733f))
* **deps:** update dependency @sentry/angular to v9.0.1 ([#464](https://github.com/glasskube/distr/issues/464)) ([857802f](https://github.com/glasskube/distr/commit/857802fe48f05f0692159a038888fefaa9faa050))
* **deps:** update dependency posthog-js to v1.217.1 ([#465](https://github.com/glasskube/distr/issues/465)) ([fb1e045](https://github.com/glasskube/distr/commit/fb1e045276cb2459b59ef95fccc3521de27dc3ad))
* **deps:** update dependency posthog-js to v1.217.2 ([#468](https://github.com/glasskube/distr/issues/468)) ([254483f](https://github.com/glasskube/distr/commit/254483f2a7a94ad4a6e305ed6c09ea61a5805672))
* **sdk/js:** correctly use default api base ([#469](https://github.com/glasskube/distr/issues/469)) ([04c4d02](https://github.com/glasskube/distr/commit/04c4d02acb4a9dfb1cd41efdc0978867786624ea))


### Other

* **backend:** use `uuid.UUID` instead of `string` for all IDs internally ([#471](https://github.com/glasskube/distr/issues/471)) ([5459351](https://github.com/glasskube/distr/commit/5459351ee67ade13501d5a3b50dc331ca261b077))
* change next release to 1.1.4 ([cbeec52](https://github.com/glasskube/distr/commit/cbeec5242f4b7213f2bc9187e69e25388bd3c101))
* **deps:** update angular-cli monorepo to v19.1.7 ([#474](https://github.com/glasskube/distr/issues/474)) ([71cdbba](https://github.com/glasskube/distr/commit/71cdbba26fe1b9c753a75c16aa7ef772fa6bc926))
* **deps:** update dependency golangci-lint to v1.64.2 ([#467](https://github.com/glasskube/distr/issues/467)) ([c112701](https://github.com/glasskube/distr/commit/c112701fd472c5492477e174694e032f7d6826d2))
* **deps:** update dependency golangci-lint to v1.64.4 ([#470](https://github.com/glasskube/distr/issues/470)) ([6bfcb9a](https://github.com/glasskube/distr/commit/6bfcb9a98e2c42380d6528ebde27a8c64a61fe2f))
* **deps:** update dependency postcss to v8.5.2 ([#461](https://github.com/glasskube/distr/issues/461)) ([063bdea](https://github.com/glasskube/distr/commit/063bdea16123a7faf6413024f49f45d82917d2b9))


### Performance

* **backend:** optimize database query for deployment with latest status ([#473](https://github.com/glasskube/distr/issues/473)) ([88a6f46](https://github.com/glasskube/distr/commit/88a6f46c1670b61f0ed5bab1f6572f157d809b8f))

## [1.1.3](https://github.com/glasskube/distr/compare/1.1.2...1.1.3) (2025-02-11)


### Bug Fixes

* **backend:** change DeploymentTarget created_by reference on delete to `RESTRICT` ([#460](https://github.com/glasskube/distr/issues/460)) ([2d51073](https://github.com/glasskube/distr/commit/2d51073c1a293feca5bd546e5523a6761236f18c))
* **deps:** update dependency @sentry/angular to v9 ([#457](https://github.com/glasskube/distr/issues/457)) ([469f5e4](https://github.com/glasskube/distr/commit/469f5e4cbc78b05a5bf565779093cdb627ddd5d2))
* **deps:** update dependency ngx-markdown to v19.1.0 ([#449](https://github.com/glasskube/distr/issues/449)) ([153f978](https://github.com/glasskube/distr/commit/153f978fd37f9478fb6299ac2feaf6779fefb13f))
* **deps:** update dependency posthog-js to v1.215.6 ([#442](https://github.com/glasskube/distr/issues/442)) ([937f0c1](https://github.com/glasskube/distr/commit/937f0c1ccf06e60e430bd6b316734ac12611083a))
* **deps:** update dependency posthog-js to v1.215.7 ([#454](https://github.com/glasskube/distr/issues/454)) ([d01d977](https://github.com/glasskube/distr/commit/d01d97725950cc7b541bd25cf7e17036ca452d11))
* **deps:** update dependency posthog-js to v1.217.0 ([#456](https://github.com/glasskube/distr/issues/456)) ([e303642](https://github.com/glasskube/distr/commit/e303642ce9fa30547746c317352435e0d379a01a))
* **deps:** update module golang.org/x/crypto to v0.33.0 ([#444](https://github.com/glasskube/distr/issues/444)) ([e51b18b](https://github.com/glasskube/distr/commit/e51b18bc446a0fb2f7bd249a3d0491958f50dab4))
* **ui:** update deployment targets list immediately ([#452](https://github.com/glasskube/distr/issues/452)) ([8fd358e](https://github.com/glasskube/distr/commit/8fd358e85ddae417c80fdc07b049c0b970482dd5))


### Other

* **deps:** update axllent/mailpit docker tag to v1.22.2 ([#445](https://github.com/glasskube/distr/issues/445)) ([7d249c9](https://github.com/glasskube/distr/commit/7d249c95578a4fcebe1d3aa9d920bf22cecb15fa))
* **deps:** update dependency jasmine-core to ~5.6.0 ([#447](https://github.com/glasskube/distr/issues/447)) ([ae5c2cb](https://github.com/glasskube/distr/commit/ae5c2cb0ef44d5c32a7505179da12ab2c63cf5a8))
* **deps:** update dependency prettier to v3.5.0 ([#448](https://github.com/glasskube/distr/issues/448)) ([f27441e](https://github.com/glasskube/distr/commit/f27441eb42c7fb1d9c6874706d5ce85f4710826e))
* **deps:** update dependency typedoc to v0.27.7 ([#450](https://github.com/glasskube/distr/issues/450)) ([2937d6a](https://github.com/glasskube/distr/commit/2937d6a07f5cedcd2339d70de3fa441718c47846))
* **deps:** update dependency typedoc-plugin-markdown to v4.4.2 ([#451](https://github.com/glasskube/distr/issues/451)) ([eae0de3](https://github.com/glasskube/distr/commit/eae0de3e889e47e72624d33176c49e04a70a9652))
* **deps:** update golangci/golangci-lint-action action to v6.3.1 ([#446](https://github.com/glasskube/distr/issues/446)) ([d2aec76](https://github.com/glasskube/distr/commit/d2aec762f50c85a59485dcb00f2efc3748cf2ce5))
* **deps:** update golangci/golangci-lint-action action to v6.3.2 ([#459](https://github.com/glasskube/distr/issues/459)) ([51451bb](https://github.com/glasskube/distr/commit/51451bb8969d6da12055698f3440986e28fb041c))

## [1.1.2](https://github.com/glasskube/distr/compare/1.1.1...1.1.2) (2025-02-07)


### Bug Fixes

* **docker-agent:** keep same docker config after self-update ([#440](https://github.com/glasskube/distr/issues/440)) ([e2619e5](https://github.com/glasskube/distr/commit/e2619e53f68a966a32f0491bcd899ab7974d1bb0))


### Other

* make customer managed environment manageable for vendor ([#439](https://github.com/glasskube/distr/issues/439)) ([8b634a9](https://github.com/glasskube/distr/commit/8b634a9555d4a5f3177e479e3f9d0512188166e5))

## [1.1.1](https://github.com/glasskube/distr/compare/1.1.0...1.1.1) (2025-02-06)


### Bug Fixes

* **deps:** update angular monorepo to v19.1.5 ([#437](https://github.com/glasskube/distr/issues/437)) ([c435fe1](https://github.com/glasskube/distr/commit/c435fe172c17b49e30e71c2c92e0618c1a7af338))


### Docs

* add register note ([7418bce](https://github.com/glasskube/distr/commit/7418bce7640b08a837be1350ce105b592fb965c5))
* format readme ([5c5791d](https://github.com/glasskube/distr/commit/5c5791d700637fe0a2cbbe11a9e15f9832ab9101))

## [1.1.0](https://github.com/glasskube/distr/compare/1.0.4...1.1.0) (2025-02-06)


### Features

* add showing invite URL after user creation ([#427](https://github.com/glasskube/distr/issues/427)) ([7570373](https://github.com/glasskube/distr/commit/7570373a21adfac900b9d7345c94c7134604b424))
* **docker-agent:** add self-updating (existing agents must be re-deployed) ([#425](https://github.com/glasskube/distr/issues/425)) ([bab6934](https://github.com/glasskube/distr/commit/bab6934a401a1399b45cefd4ecb9a35cbbd5ce8d))
* support environment for docker compose ([#424](https://github.com/glasskube/distr/issues/424)) ([7b8812b](https://github.com/glasskube/distr/commit/7b8812b11399c2d36258ac1721836bce24cda61d))


### Bug Fixes

* **backend:** avoid panic on empty compose file ([#423](https://github.com/glasskube/distr/issues/423)) ([6403124](https://github.com/glasskube/distr/commit/6403124df1aa0a89934b1f00db51b62ab9065e93))
* **backend:** avoid struct parsing error after update statement ([#420](https://github.com/glasskube/distr/issues/420)) ([ddf1d44](https://github.com/glasskube/distr/commit/ddf1d4453510bc263c5b5eebd932958404c8200d))
* **deps:** update aws-sdk-go-v2 monorepo ([#429](https://github.com/glasskube/distr/issues/429)) ([7312439](https://github.com/glasskube/distr/commit/7312439540ce5a05b693757b91986214630d654e))
* **deps:** update dependency @angular/cdk to v19.1.3 ([#428](https://github.com/glasskube/distr/issues/428)) ([6c75a3d](https://github.com/glasskube/distr/commit/6c75a3dea2266edaf85da15b27c8b898b2bae4f1))
* **deps:** update dependency globe.gl to v2.39.7 ([#432](https://github.com/glasskube/distr/issues/432)) ([c22322e](https://github.com/glasskube/distr/commit/c22322ef8642c5a5542f6107070bed7c757b3646))
* **deps:** update dependency posthog-js to v1.215.3 ([#416](https://github.com/glasskube/distr/issues/416)) ([cc8ee90](https://github.com/glasskube/distr/commit/cc8ee900fd78cab791889a697593ded6f8956e98))
* **deps:** update dependency posthog-js to v1.215.5 ([#430](https://github.com/glasskube/distr/issues/430)) ([32eae8f](https://github.com/glasskube/distr/commit/32eae8f7d34325a6dee2e82d94481df2adaae431))
* **deps:** update module github.com/aws/aws-sdk-go-v2/config to v1.29.5 ([#422](https://github.com/glasskube/distr/issues/422)) ([266b56f](https://github.com/glasskube/distr/commit/266b56f91814ead9ca50f711037be65e9325a3ec))


### Other

* allow disabling email verification ([#426](https://github.com/glasskube/distr/issues/426)) ([97301ac](https://github.com/glasskube/distr/commit/97301acf5886813ff77226e87f0812b630fba181))
* **deps:** update angular-cli monorepo to v19.1.6 ([#431](https://github.com/glasskube/distr/issues/431)) ([e24076c](https://github.com/glasskube/distr/commit/e24076c03cf3e8443ce0e76b2d88531c649efa02))
* **deps:** update axllent/mailpit docker tag to v1.22.1 ([#433](https://github.com/glasskube/distr/issues/433)) ([b757541](https://github.com/glasskube/distr/commit/b757541cd512996686e88f1d38f6182f082f7d60))
* **deps:** update docker/setup-buildx-action action to v3.9.0 ([#434](https://github.com/glasskube/distr/issues/434)) ([55e9397](https://github.com/glasskube/distr/commit/55e9397c6e021871a216845496e358cd586d37cf))
* **deps:** update golangci/golangci-lint-action action to v6.3.0 ([#421](https://github.com/glasskube/distr/issues/421)) ([d991b87](https://github.com/glasskube/distr/commit/d991b871a122a8c70fbcdf339d78f3936da6e39b))
* mount host docker config directory ([#435](https://github.com/glasskube/distr/issues/435)) ([13b2dd7](https://github.com/glasskube/distr/commit/13b2dd7b5e969de63a0e372d5e7551e4bace4adf))


### Docs

* link to register endpoint ([fd551b6](https://github.com/glasskube/distr/commit/fd551b64a77a036d3f64182a895c3e4760b860db))

## [1.0.4](https://github.com/glasskube/distr/compare/1.0.3...1.0.4) (2025-02-04)


### Other

* remove console logs ([#417](https://github.com/glasskube/distr/issues/417)) ([9033385](https://github.com/glasskube/distr/commit/90333858fa34a0d65ed6fb60da49de67d42c45bf))

## [1.0.3](https://github.com/glasskube/distr/compare/1.0.2...1.0.3) (2025-02-04)


### Bug Fixes

* **deps:** update module github.com/go-chi/chi/v5 to v5.2.1 ([#413](https://github.com/glasskube/distr/issues/413)) ([3b7ab5d](https://github.com/glasskube/distr/commit/3b7ab5d53f2f730ee240d67057849d88eb65414a))


### Other

* **ci:** fix release build error ([#414](https://github.com/glasskube/distr/issues/414)) ([9bfc472](https://github.com/glasskube/distr/commit/9bfc4729d9259ffcff72bfa6e60271974748d9a4))

## [1.0.2](https://github.com/glasskube/distr/compare/1.0.1...1.0.2) (2025-02-04)


### Bug Fixes

* **deps:** update dependency @codemirror/state to v6.5.2 ([#405](https://github.com/glasskube/distr/issues/405)) ([92933ff](https://github.com/glasskube/distr/commit/92933ff34fb110baa7619333b56da1e54ffe9793))
* **deps:** update dependency @sentry/angular to v8.54.0 ([#406](https://github.com/glasskube/distr/issues/406)) ([d2d4795](https://github.com/glasskube/distr/commit/d2d47959c0639ff7a3fcb89e138c17ec2e6bbcbf))
* **deps:** update dependency globe.gl to v2.39.6 ([#410](https://github.com/glasskube/distr/issues/410)) ([6df822d](https://github.com/glasskube/distr/commit/6df822de466800298730c4df1c1b2efa97aa25aa))
* **deps:** update dependency posthog-js to v1.215.2 ([#411](https://github.com/glasskube/distr/issues/411)) ([b6dde86](https://github.com/glasskube/distr/commit/b6dde8680af463e0982e03c05fa417944b544618))
* **deps:** update dependency semver to v7.7.1 ([#412](https://github.com/glasskube/distr/issues/412)) ([99d46ad](https://github.com/glasskube/distr/commit/99d46ad59864222d285f590d91a6a1ce97039de5))
* **ui:** improve token handling failures ([#407](https://github.com/glasskube/distr/issues/407)) ([6aeaa56](https://github.com/glasskube/distr/commit/6aeaa56b94247c677deaca21bbf53c0410b18335))


### Other

* fix helm chart build error ([#408](https://github.com/glasskube/distr/issues/408)) ([c3a0a8c](https://github.com/glasskube/distr/commit/c3a0a8c391d8a0d8b07c522e243d8ba3d2f729f5))

## [1.0.1](https://github.com/glasskube/distr/compare/1.0.0...1.0.1) (2025-02-03)


### Bug Fixes

* **deps:** update angular monorepo to v19.1.4 ([#390](https://github.com/glasskube/distr/issues/390)) ([9f5c90c](https://github.com/glasskube/distr/commit/9f5c90c2c30caabc1cc68145bf5bc4181e1f4f4d))
* **deps:** update aws-sdk-go-v2 monorepo ([#394](https://github.com/glasskube/distr/issues/394)) ([0a53571](https://github.com/glasskube/distr/commit/0a53571001a683c4b460b544ceb5f0e9696d551b))
* **deps:** update aws-sdk-go-v2 monorepo ([#402](https://github.com/glasskube/distr/issues/402)) ([d27e2f0](https://github.com/glasskube/distr/commit/d27e2f07f0c3b7577d31e6b8b6e36abf169a42c2))
* **deps:** update dependency @angular/cdk to v19.1.2 ([#391](https://github.com/glasskube/distr/issues/391)) ([e512198](https://github.com/glasskube/distr/commit/e512198ceea5bc0e3d0a43848e359114358e2414))
* **deps:** update dependency @sentry/angular to v8.52.1 ([#392](https://github.com/glasskube/distr/issues/392)) ([e93952b](https://github.com/glasskube/distr/commit/e93952bd934e869ba1ecda1bf9b26b52faa6e5e4))
* **deps:** update dependency @sentry/angular to v8.53.0 ([#398](https://github.com/glasskube/distr/issues/398)) ([2f9b802](https://github.com/glasskube/distr/commit/2f9b8025a9cf4395788c34cdabdf842f8319e536))
* **deps:** update dependency globe.gl to v2.39.3 ([#403](https://github.com/glasskube/distr/issues/403)) ([4e0f6c2](https://github.com/glasskube/distr/commit/4e0f6c296b2855c464024e13878cfdb112e50dc3))
* **deps:** update dependency globe.gl to v2.39.5 ([#404](https://github.com/glasskube/distr/issues/404)) ([30d68ec](https://github.com/glasskube/distr/commit/30d68ec1ae3a4b3ece5c5159f3b572698fa2b4bb))
* **deps:** update dependency posthog-js to v1.212.1 ([#387](https://github.com/glasskube/distr/issues/387)) ([d4436b8](https://github.com/glasskube/distr/commit/d4436b8438dad5c4f2e06da4a2f6a50f23be2f07))
* **deps:** update dependency posthog-js to v1.214.1 ([#395](https://github.com/glasskube/distr/issues/395)) ([bfac7c2](https://github.com/glasskube/distr/commit/bfac7c23b481cd3894bec65bd36085f4bda51723))
* **deps:** update dependency posthog-js to v1.215.1 ([#401](https://github.com/glasskube/distr/issues/401)) ([1723795](https://github.com/glasskube/distr/commit/1723795c95d8ce6124b80ca481c984d3b7afb63b))
* **deps:** update dependency semver to v7.7.0 ([#388](https://github.com/glasskube/distr/issues/388)) ([53ecf08](https://github.com/glasskube/distr/commit/53ecf0813a09200d42ccdc38ad16095c3e6c070a))
* **deps:** update module github.com/spf13/pflag to v1.0.6 ([#389](https://github.com/glasskube/distr/issues/389)) ([0b185f5](https://github.com/glasskube/distr/commit/0b185f5bae26966f2a23783ef0bd49df1faae243))
* **sdk/js:** make chartName optional ([#400](https://github.com/glasskube/distr/issues/400)) ([9a49a38](https://github.com/glasskube/distr/commit/9a49a38b4aa22353fe6f009aa4671cded956b239))


### Other

* **deps:** update angular-cli monorepo to v19.1.5 ([#385](https://github.com/glasskube/distr/issues/385)) ([fca184e](https://github.com/glasskube/distr/commit/fca184e570156b434e4772f6d6e5ed58a829b281))


### Docs

* add about section to README.md ([#397](https://github.com/glasskube/distr/issues/397)) ([73a7da6](https://github.com/glasskube/distr/commit/73a7da648cc247cd71ea4b13649b462ae74d0528))
* add Helm chart README.md ([#399](https://github.com/glasskube/distr/issues/399)) ([0768637](https://github.com/glasskube/distr/commit/07686379c2acdf582863cca578a6fe8712de1d76))

## [1.0.0](https://github.com/glasskube/distr/compare/0.13.2...1.0.0) (2025-01-29)


### Features

* **chart:** add initial version of the helm chart ([#383](https://github.com/glasskube/distr/issues/383)) ([78f9d58](https://github.com/glasskube/distr/commit/78f9d5817d40c377c1a116d90e11ec82b2bc8386))


### Bug Fixes

* **deps:** update dependency @sentry/angular to v8.52.0 ([#377](https://github.com/glasskube/distr/issues/377)) ([e80424e](https://github.com/glasskube/distr/commit/e80424ece7dc08a9ec5ac3ea546882453bd2f01e))
* **deps:** update dependency posthog-js to v1.211.1 ([#376](https://github.com/glasskube/distr/issues/376)) ([567e821](https://github.com/glasskube/distr/commit/567e8211dfc7e20cc0620111b50393b44a043db1))
* **deps:** update dependency posthog-js to v1.211.2 ([#379](https://github.com/glasskube/distr/issues/379)) ([1fe8839](https://github.com/glasskube/distr/commit/1fe88391f1a5c173453154ba683d7ad3fbb8ff83))
* **deps:** update dependency posthog-js to v1.211.3 ([#381](https://github.com/glasskube/distr/issues/381)) ([c225ef9](https://github.com/glasskube/distr/commit/c225ef97fd336e0618ce13ce25d72ae159ed73b0))
* **ui:** link to login at password reset ([#378](https://github.com/glasskube/distr/issues/378)) ([216c7c2](https://github.com/glasskube/distr/commit/216c7c28fc32833149610bd8e81e2996335ce152))


### Other

* change homepage url to distr.sh ([77d042b](https://github.com/glasskube/distr/commit/77d042b185206cc91b28148201dc66cc5d38fdd6))
* rename api key prefix ([7e1e32e](https://github.com/glasskube/distr/commit/7e1e32eb7bb0e9252898e82da32e4f25b60fb6ca))
* rename from "distr.sh" to "Distr" ([#380](https://github.com/glasskube/distr/issues/380)) ([801c274](https://github.com/glasskube/distr/commit/801c2741c2a65b44f1b28b3a6a7b5f292f64f20f))
* set next release to 1.0.0 ([719830e](https://github.com/glasskube/distr/commit/719830eb06ecaeb054a8204e75d10fd50aa31e4c))
* update sample app version ([#382](https://github.com/glasskube/distr/issues/382)) ([bd39bea](https://github.com/glasskube/distr/commit/bd39bea7ef62ac86d5b757e94be0cf62acc8ae3f))


### Docs

* add your application to architecture diagram ([df35110](https://github.com/glasskube/distr/commit/df3511050a3b9d8f845e5f40e420776171435322))

## [0.13.2](https://github.com/glasskube/distr/compare/0.13.1...0.13.2) (2025-01-28)


### Bug Fixes

* **deps:** update dependency posthog-js to v1.211.0 ([#371](https://github.com/glasskube/distr/issues/371)) ([3e3e751](https://github.com/glasskube/distr/commit/3e3e7513547d08e69ea259db49302134ce8b0c10))


### Other

* add sdk docs ([#368](https://github.com/glasskube/distr/issues/368)) ([4a93550](https://github.com/glasskube/distr/commit/4a93550dcc36081039fb632ed0110f5505549a47))
* rename distr docker agent ([#374](https://github.com/glasskube/distr/issues/374)) ([ee7cb07](https://github.com/glasskube/distr/commit/ee7cb071307520377af1cd0427cd1284fca8f22d))
* replace icons and agent variable names ([#373](https://github.com/glasskube/distr/issues/373)) ([daa4141](https://github.com/glasskube/distr/commit/daa4141eb1011cd38eeb225bf0689fc85a3fa0a8))

## [0.13.1](https://github.com/glasskube/distr/compare/0.13.0...0.13.1) (2025-01-27)


### Other

* force new release (no changes) ([826983a](https://github.com/glasskube/distr/commit/826983a716eed866c8eb6a39fda07e421a53a589))

## [0.13.0](https://github.com/glasskube/distr/compare/0.12.0...0.13.0) (2025-01-27)


### Features

* api token authentication ([#333](https://github.com/glasskube/distr/issues/333)) ([95ff8d0](https://github.com/glasskube/distr/commit/95ff8d08e70225aa39fa6eb5eecffcbc03ac43e2))
* **js-sdk:** add initial version of the JavaScript SDK ([#329](https://github.com/glasskube/distr/issues/329)) ([a0c9b07](https://github.com/glasskube/distr/commit/a0c9b07ac66073fff794011076a6a521c21245fc))


### Bug Fixes

* **deps:** update angular monorepo to v19.1.3 ([#343](https://github.com/glasskube/distr/issues/343)) ([b48d68d](https://github.com/glasskube/distr/commit/b48d68d91c4bd9c01a5686d2520ff514d50b9d7a))
* **deps:** update aws-sdk-go-v2 monorepo ([#358](https://github.com/glasskube/distr/issues/358)) ([0568a82](https://github.com/glasskube/distr/commit/0568a823461ef4d2a1870db44d854102e1dcd301))
* **deps:** update dependency @angular/cdk to v19.1.1 ([#341](https://github.com/glasskube/distr/issues/341)) ([a505618](https://github.com/glasskube/distr/commit/a50561810801330a0f5fbcad40fdac3f85f0eb00))
* **deps:** update dependency @sentry/angular to v8.51.0 ([#342](https://github.com/glasskube/distr/issues/342)) ([4a67853](https://github.com/glasskube/distr/commit/4a67853f9a6dbc35b10b53dd0c267e8fdf63547d))
* **deps:** update dependency apexcharts to v4.4.0 ([#336](https://github.com/glasskube/distr/issues/336)) ([ec7af06](https://github.com/glasskube/distr/commit/ec7af0678375d551f2ec69ac84f13ff7e7d1845a))
* **deps:** update dependency globe.gl to v2.39.2 ([#344](https://github.com/glasskube/distr/issues/344)) ([93a7a79](https://github.com/glasskube/distr/commit/93a7a7961abea98e8148f2c5af604d2d9496bd84))
* **deps:** update dependency posthog-js to v1.207.8 ([#340](https://github.com/glasskube/distr/issues/340)) ([b25ae79](https://github.com/glasskube/distr/commit/b25ae791533b728931cc62be862f8cf37b9d627e))
* **deps:** update dependency posthog-js to v1.209.0 ([#352](https://github.com/glasskube/distr/issues/352)) ([12dcb72](https://github.com/glasskube/distr/commit/12dcb720d038f6184000111354c102d2b3bd4721))
* **deps:** update dependency posthog-js to v1.210.2 ([#357](https://github.com/glasskube/distr/issues/357)) ([3cdd140](https://github.com/glasskube/distr/commit/3cdd140ed5951d66cd1d941ada10f1c601c1ad00))
* **deps:** update module github.com/golang-migrate/migrate/v4 to v4.18.2 ([#360](https://github.com/glasskube/distr/issues/360)) ([0f96a60](https://github.com/glasskube/distr/commit/0f96a607a45090f29a09014f5df380777e21b712))
* **ui:** customers can create api tokens too ([#349](https://github.com/glasskube/distr/issues/349)) ([ffb8cdb](https://github.com/glasskube/distr/commit/ffb8cdbcf319ffd3ea73c96451baeacb5e40697c))


### Other

* add deploy compose file to release please extra files ([#361](https://github.com/glasskube/distr/issues/361)) ([80de58e](https://github.com/glasskube/distr/commit/80de58e6e72221ca696881996e5ae90ce94cd9cf))
* add docker compose deployment option ([#338](https://github.com/glasskube/distr/issues/338)) ([cf30e4b](https://github.com/glasskube/distr/commit/cf30e4b975750359f61b341c95d60498dbcb5ffd))
* add LICENSE ([#363](https://github.com/glasskube/distr/issues/363)) ([476f567](https://github.com/glasskube/distr/commit/476f5674caa2112d81e22d6f08f74a77bd2f32a7))
* additional api request limits ([#355](https://github.com/glasskube/distr/issues/355)) ([43be173](https://github.com/glasskube/distr/commit/43be173aa65fa5c60d29c27c1b1244b25ae6558e))
* change agent secret encoding from base64 to hex ([#350](https://github.com/glasskube/distr/issues/350)) ([2907357](https://github.com/glasskube/distr/commit/29073576ef5ad7e412e541aa116bf0c912e5188e))
* **deps:** update actions/setup-node digest to 1d0ff46 ([#362](https://github.com/glasskube/distr/issues/362)) ([f3c28b4](https://github.com/glasskube/distr/commit/f3c28b4143acac92a6c63afd335c4ec933f1c280))
* **deps:** update angular-cli monorepo to v19.1.4 ([#345](https://github.com/glasskube/distr/issues/345)) ([db9e885](https://github.com/glasskube/distr/commit/db9e8851bf48d461bbb092e7f5e68e4b2c186b1b))
* **deps:** update axllent/mailpit docker tag to v1.22.0 ([#359](https://github.com/glasskube/distr/issues/359)) ([49fd374](https://github.com/glasskube/distr/commit/49fd3742557baee91ff73aeb9fbd55a7ef1371bf))
* **deps:** update docker/build-push-action action to v6.13.0 ([#356](https://github.com/glasskube/distr/issues/356)) ([0d8ac4e](https://github.com/glasskube/distr/commit/0d8ac4e8d325356d26528b660b221f786bf97a89))
* fix logo in readme ([#367](https://github.com/glasskube/distr/issues/367)) ([2226916](https://github.com/glasskube/distr/commit/22269166cb5b26a4f2cb40eca58eaad89c990ff8))
* rebrand to distr.sh ([#365](https://github.com/glasskube/distr/issues/365)) ([c614f1a](https://github.com/glasskube/distr/commit/c614f1a2529e2809e4effab5a528eb2af423fb20))
* remove hardcoded sentry dsn and posthog token ([#351](https://github.com/glasskube/distr/issues/351)) ([1ed34c7](https://github.com/glasskube/distr/commit/1ed34c7191c14c71cd381f0393cfbd93a9eedf2c))
* remove token logging ([#339](https://github.com/glasskube/distr/issues/339)) ([7715f25](https://github.com/glasskube/distr/commit/7715f255c33fe233d486dcefe71918217c8f0667))
* remove version distribution chart ([#364](https://github.com/glasskube/distr/issues/364)) ([be0a0c0](https://github.com/glasskube/distr/commit/be0a0c078a351b6798faf363e4c8781ea51f49ca))


### Docs

* update README.md ([#347](https://github.com/glasskube/distr/issues/347)) ([91e7ae1](https://github.com/glasskube/distr/commit/91e7ae1a000005485140006b7f9f50fb4c55210d))

## [0.12.0](https://github.com/glasskube/cloud/compare/0.11.0...0.12.0) (2025-01-22)


### Features

* add copy id button ([#331](https://github.com/glasskube/cloud/issues/331)) ([4cf2eef](https://github.com/glasskube/cloud/commit/4cf2eefb9b39d5761fa95142908f180323c06a23))
* always enable status logs for vendors ([#335](https://github.com/glasskube/cloud/issues/335)) ([de7aa33](https://github.com/glasskube/cloud/commit/de7aa33e509337745d50f266f4acf55815feee40))
* generate docker compose project name  ([#325](https://github.com/glasskube/cloud/issues/325)) ([f8537c9](https://github.com/glasskube/cloud/commit/f8537c9f14717b4b5b4102568fa818a11903a8ec))


### Bug Fixes

* **deps:** update dependency posthog-js to v1.207.1 ([#330](https://github.com/glasskube/cloud/issues/330)) ([17a16cf](https://github.com/glasskube/cloud/commit/17a16cfaf08abb39b2d9b44e8b3f1ebf2fbe58c4))
* **deps:** update dependency posthog-js to v1.207.2 ([#334](https://github.com/glasskube/cloud/issues/334)) ([614db6c](https://github.com/glasskube/cloud/commit/614db6c060af6e88fe2584bcb4f98a848663a54b))


### Other

* **deps:** update angular-cli monorepo to v19.1.3 ([#327](https://github.com/glasskube/cloud/issues/327)) ([0795b53](https://github.com/glasskube/cloud/commit/0795b53783d24a0f8bae79a613f45ab1b800a521))
* **deps:** update dependency @sentry/cli to v2.41.1 ([#332](https://github.com/glasskube/cloud/issues/332)) ([7f249b6](https://github.com/glasskube/cloud/commit/7f249b64a38012763569be4dfca293bd9195d4a7))

## [0.11.0](https://github.com/glasskube/cloud/compare/0.10.0...0.11.0) (2025-01-21)


### Features

* add agent docker config for kubernetes agent ([#304](https://github.com/glasskube/cloud/issues/304)) ([c1b4c0a](https://github.com/glasskube/cloud/commit/c1b4c0ac9e3825af82dff75cf5b5865fc1e8b3b2))
* add connecting deployment target again ([#307](https://github.com/glasskube/cloud/issues/307)) ([fb18211](https://github.com/glasskube/cloud/commit/fb1821112ed8f9b59e40643b2c38e39b0b25ab64))
* add organization branding in invite emails ([#312](https://github.com/glasskube/cloud/issues/312)) ([fbd014a](https://github.com/glasskube/cloud/commit/fbd014a54117b663e5d719f583ebf9cac5ec6f34))
* **ui:** add application version copy ([#283](https://github.com/glasskube/cloud/issues/283)) ([97de1f3](https://github.com/glasskube/cloud/commit/97de1f3bcddeca1c40aacad785c5e95b44940462))
* **ui:** add autotrim on blur ([#310](https://github.com/glasskube/cloud/issues/310)) ([34a1632](https://github.com/glasskube/cloud/commit/34a1632746cd3aac0373fd022055480b41aa560d))
* **ui:** add backoff-retry for deployment target polling ([#311](https://github.com/glasskube/cloud/issues/311)) ([d734f09](https://github.com/glasskube/cloud/commit/d734f091d4de3ee5285e7733d4109528e091c6e1))
* **ui:** use text input for helm values and template ([#308](https://github.com/glasskube/cloud/issues/308)) ([96b6651](https://github.com/glasskube/cloud/commit/96b6651c50790e7565b5ceb72e76957c56f9d602))


### Bug Fixes

* add password validation on update settings endpoint ([#318](https://github.com/glasskube/cloud/issues/318)) ([3176f50](https://github.com/glasskube/cloud/commit/3176f50c0c537df0f54fce29d1d31c3133ac422a))
* **backend:** improve sorting of deployment targets ([#305](https://github.com/glasskube/cloud/issues/305)) ([a597b14](https://github.com/glasskube/cloud/commit/a597b1444fe91f94ade0eb38712c20778ed309a2))
* **deps:** update angular monorepo to v19.1.2 ([#326](https://github.com/glasskube/cloud/issues/326)) ([2a0d0ae](https://github.com/glasskube/cloud/commit/2a0d0ae50a8b6e0c43c80c6d04ebfd379eb21610))
* **deps:** update aws-sdk-go-v2 monorepo ([#313](https://github.com/glasskube/cloud/issues/313)) ([0af2600](https://github.com/glasskube/cloud/commit/0af26004eb715cd46bdc8f9663a4dd7671a42ed7))
* **deps:** update dependency @angular/cdk to v19.1.0 ([#303](https://github.com/glasskube/cloud/issues/303)) ([f09e767](https://github.com/glasskube/cloud/commit/f09e767ab0ae4f65afcd1f55bebca75dd78bb771))
* **deps:** update dependency globe.gl to v2.38.1 ([#316](https://github.com/glasskube/cloud/issues/316)) ([6573b45](https://github.com/glasskube/cloud/commit/6573b45e45ac69e9d441416b85a77c0ab414ad1c))
* **deps:** update dependency globe.gl to v2.39.0 ([#317](https://github.com/glasskube/cloud/issues/317)) ([d95a122](https://github.com/glasskube/cloud/commit/d95a122f628ec842f4f519e778a28b22c03ae89d))
* **deps:** update dependency posthog-js to v1.207.0 ([#309](https://github.com/glasskube/cloud/issues/309)) ([5569b9f](https://github.com/glasskube/cloud/commit/5569b9facd6085f7104444ef3cb846baf09a6c20))
* **ui:** yaml editor respects changes ([#322](https://github.com/glasskube/cloud/issues/322)) ([7fafdfa](https://github.com/glasskube/cloud/commit/7fafdfa0a4108ddada158d86f346fbb37323a06c))


### Other

* **deps:** update angular-cli monorepo to v19.1.2 ([#314](https://github.com/glasskube/cloud/issues/314)) ([8759eed](https://github.com/glasskube/cloud/commit/8759eed79cf1cce2e32f0eb421701452757346a6))
* **deps:** update dependency @sentry/cli to v2.41.0 ([#324](https://github.com/glasskube/cloud/issues/324)) ([adb46fb](https://github.com/glasskube/cloud/commit/adb46fbb63ef4c1aeccdd69b3ef575f0bdef9714))
* remove API Keys from side bar as they will be in the user dropdown ([8399d6b](https://github.com/glasskube/cloud/commit/8399d6b31859cefb521400352e7066a487d49d02))
* remove logging of tokens for invites ([#321](https://github.com/glasskube/cloud/issues/321)) ([2741b47](https://github.com/glasskube/cloud/commit/2741b47a1049d3a1e94de251e15170d4c58cbeb9))
* **ui:** remove "deployment notes" text field ([#320](https://github.com/glasskube/cloud/issues/320)) ([90cd7be](https://github.com/glasskube/cloud/commit/90cd7be74421ab3a48929233d4cd5d7840d1d72b))
* **ui:** remove "terms & conditions" checkboxes ([#319](https://github.com/glasskube/cloud/issues/319)) ([859a784](https://github.com/glasskube/cloud/commit/859a784ed4c58fc28d2cee3466b4e0b0af722b6b))
* **ui:** support scope in onboarding wizard ([#323](https://github.com/glasskube/cloud/issues/323)) ([ae914ec](https://github.com/glasskube/cloud/commit/ae914ec282cd0f8b583440d6f4cc064eccc42aae))

## [0.10.0](https://github.com/glasskube/cloud/compare/0.9.2...0.10.0) (2025-01-16)


### Features

* add support for cluster scope in  kubernetes agent ([#298](https://github.com/glasskube/cloud/issues/298)) ([c3479ad](https://github.com/glasskube/cloud/commit/c3479adfd0a1579a96b91adaaf570b0fdfbc7e76))
* **ui:** hide chart name when repo type is oci ([#282](https://github.com/glasskube/cloud/issues/282)) ([55fc754](https://github.com/glasskube/cloud/commit/55fc754b03ca11b3d1a526f2ba97c83aebf6cbcd))


### Bug Fixes

* **deps:** update angular monorepo to v19.1.1 ([#300](https://github.com/glasskube/cloud/issues/300)) ([bedc1a3](https://github.com/glasskube/cloud/commit/bedc1a32aa70ed137ab96cec0f04f7c1886c01a9))
* **deps:** update dependency globe.gl to v2.38.0 ([#301](https://github.com/glasskube/cloud/issues/301)) ([298c0df](https://github.com/glasskube/cloud/commit/298c0dfc3b18e917fd0ec148f97b0b039f0ea957))
* **ui:** coordinates can be 0 ([#297](https://github.com/glasskube/cloud/issues/297)) ([f952ccf](https://github.com/glasskube/cloud/commit/f952ccff2aa2f88a0c99a5aa9c373c9f0f6c2ee6))
* use revision id from resource ([#302](https://github.com/glasskube/cloud/issues/302)) ([bc4bd6e](https://github.com/glasskube/cloud/commit/bc4bd6e422715c7c6f02d8dd1053784c4b83517d))


### Other

* **deps:** update angular-cli monorepo to v19.1.1 ([#293](https://github.com/glasskube/cloud/issues/293)) ([8d9058e](https://github.com/glasskube/cloud/commit/8d9058e0882fe58fa9d73d7278b4a910e51b4fc9))
* indicate stale deployments as stale after 10 seconds ([215365e](https://github.com/glasskube/cloud/commit/215365eb8c8e451bc502d7595bb73e3f08c58179))

## [0.9.2](https://github.com/glasskube/cloud/compare/0.9.1...0.9.2) (2025-01-16)


### Bug Fixes

* share polling subscription ([#295](https://github.com/glasskube/cloud/issues/295)) ([3a95449](https://github.com/glasskube/cloud/commit/3a954493d6c17eeedfe554e6a07075ab156a0dc6))
* update deployment target without agent update ([#294](https://github.com/glasskube/cloud/issues/294)) ([94b5f58](https://github.com/glasskube/cloud/commit/94b5f58cd8292c3b19abdfe139b037ba387c4870))

## [0.9.1](https://github.com/glasskube/cloud/compare/v0.9.0...0.9.1) (2025-01-16)


### Bug Fixes

* also set chart version for oci charts ([#288](https://github.com/glasskube/cloud/issues/288)) ([c21c003](https://github.com/glasskube/cloud/commit/c21c0033b48eefb6e1c06a90afb2b59803354ef3))
* **deps:** update angular monorepo to v19.1.0 ([#284](https://github.com/glasskube/cloud/issues/284)) ([164ebb4](https://github.com/glasskube/cloud/commit/164ebb4465823de01bf74659cb870e7dfb93146e))
* **deps:** update aws-sdk-go-v2 monorepo ([#285](https://github.com/glasskube/cloud/issues/285)) ([d4e2490](https://github.com/glasskube/cloud/commit/d4e2490545ba0eb8f883508b5a90244e76599569))
* **deps:** update dependency @sentry/angular to v8.50.0 ([#281](https://github.com/glasskube/cloud/issues/281)) ([f0044b9](https://github.com/glasskube/cloud/commit/f0044b96c24d6fafb9f0ace70f5e920c23d12d9c))
* **deps:** update module helm.sh/helm/v3 to v3.17.0 ([#286](https://github.com/glasskube/cloud/issues/286)) ([3141f97](https://github.com/glasskube/cloud/commit/3141f9715d8ccbe0f61a49d6a9c1b68f47898e26))
* redirect customer to home instead of deployments after login ([122f814](https://github.com/glasskube/cloud/commit/122f81485e4b53f40857ace282ae241412c2b7e0))


### Other

* **deps:** update angular-cli monorepo to v19.1.0 ([#290](https://github.com/glasskube/cloud/issues/290)) ([a5a8e0f](https://github.com/glasskube/cloud/commit/a5a8e0f13292a0e9a158b9ad524b4cb4db9ba7e7))
* do not use "v" prefix in version tag ([#292](https://github.com/glasskube/cloud/issues/292)) ([0e38939](https://github.com/glasskube/cloud/commit/0e389397d10f6c626f18d0db25d3d05448bc1ecc))
* remove unused pipe import ([#291](https://github.com/glasskube/cloud/issues/291)) ([03ffc69](https://github.com/glasskube/cloud/commit/03ffc696ff331be60d93c9354e74aa611e9b2a23))

## [0.9.0](https://github.com/glasskube/cloud/compare/v0.8.2...v0.9.0) (2025-01-16)


### Features

* add "connect" support for the kubernetes agent ([#254](https://github.com/glasskube/cloud/issues/254)) ([6fe5cee](https://github.com/glasskube/cloud/commit/6fe5ceeff5ec2096af44f455a57da3be72e869c5))
* add deployment status and deployment polling ([#270](https://github.com/glasskube/cloud/issues/270)) ([c97c776](https://github.com/glasskube/cloud/commit/c97c77637690e65ef0707f9ef2222b3af26b1d31))
* add helm deployment support for applications ([#232](https://github.com/glasskube/cloud/issues/232)) ([67ebe05](https://github.com/glasskube/cloud/commit/67ebe05c9c58556e364dc0514ae9895bd4dc8a84))
* add helm deployment support for deployment targets and deployments ([#227](https://github.com/glasskube/cloud/issues/227)) ([77ed23c](https://github.com/glasskube/cloud/commit/77ed23cd4b7f04ba5eb605b9615b1a727fcc3ffd))
* add using deployment revision status in deploymen table ([#280](https://github.com/glasskube/cloud/issues/280)) ([4ae1f89](https://github.com/glasskube/cloud/commit/4ae1f899b33ce674bfc74d456bcd12f41ab0a59b))
* data generator uses postgres bulk copy feature ([#271](https://github.com/glasskube/cloud/issues/271)) ([a3864ed](https://github.com/glasskube/cloud/commit/a3864ed2681ce1e62a3ef6ba3c13bb34ffd11eb4))
* enable logout on verify page ([#237](https://github.com/glasskube/cloud/issues/237)) ([cd087da](https://github.com/glasskube/cloud/commit/cd087da4f6b977257c468dfb43085f0344d8c3c4))
* increase deploy modal width ([#279](https://github.com/glasskube/cloud/issues/279)) ([54773be](https://github.com/glasskube/cloud/commit/54773becdcc873d1a2d9884df4a2a33bfd2bad40))
* kubernetes agent first version ([#242](https://github.com/glasskube/cloud/issues/242)) ([cb0fdc4](https://github.com/glasskube/cloud/commit/cb0fdc4fc52e2709e05e4643da2a31d887c89658))
* kubernetes agent self updates ([#276](https://github.com/glasskube/cloud/issues/276)) ([adbf313](https://github.com/glasskube/cloud/commit/adbf313ef86841fdc09faf98cad72c9382602d1b))
* kubernetes agent status check ([#265](https://github.com/glasskube/cloud/issues/265)) ([0a95aa3](https://github.com/glasskube/cloud/commit/0a95aa37f405bf15644d26580468f3f4f3f25429))
* organization branding ([#258](https://github.com/glasskube/cloud/issues/258)) ([489217f](https://github.com/glasskube/cloud/commit/489217f9913ac1e4dd0637d1a564338529155a70))
* rate limit agent and auth endpoints ([#239](https://github.com/glasskube/cloud/issues/239)) ([e48c06b](https://github.com/glasskube/cloud/commit/e48c06b7bd00948a31be4252958ce77655edd8d0))
* send merged values to kubernetes agent ([#263](https://github.com/glasskube/cloud/issues/263)) ([969e554](https://github.com/glasskube/cloud/commit/969e554699c7912e41761301c0e190f6bae945ce))
* show deployment status details ([#262](https://github.com/glasskube/cloud/issues/262)) ([0a028a1](https://github.com/glasskube/cloud/commit/0a028a1e9875ca3c2700a0bd461de8e8daee836c))
* **ui:** make deployment target selectable in uptime chart ([#236](https://github.com/glasskube/cloud/issues/236)) ([0693272](https://github.com/glasskube/cloud/commit/069327254d43ea717299e032c37360c6436ecb1f))
* **ui:** yaml editor ([#277](https://github.com/glasskube/cloud/issues/277)) ([c237e05](https://github.com/glasskube/cloud/commit/c237e05562da80495260e62169ccdea138e09487))


### Bug Fixes

* customer and user email verification ([#238](https://github.com/glasskube/cloud/issues/238)) ([b3528d3](https://github.com/glasskube/cloud/commit/b3528d302dd0a404279f9e743f6c0a788dd67dd8))
* **deps:** update angular monorepo to v19.0.6 ([#244](https://github.com/glasskube/cloud/issues/244)) ([2d241a6](https://github.com/glasskube/cloud/commit/2d241a6bfc0bc571755018f23c6361a246a43a3c))
* **deps:** update aws-sdk-go-v2 monorepo ([#251](https://github.com/glasskube/cloud/issues/251)) ([c27b3e6](https://github.com/glasskube/cloud/commit/c27b3e6e5d93d53227f70a955a332cea745ccd1e))
* **deps:** update aws-sdk-go-v2 monorepo ([#272](https://github.com/glasskube/cloud/issues/272)) ([0cadbd9](https://github.com/glasskube/cloud/commit/0cadbd9f96a3f4c788f9694152ef2fb524e7e250))
* **deps:** update dependency @angular/cdk to v19.0.5 ([#245](https://github.com/glasskube/cloud/issues/245)) ([b9f233d](https://github.com/glasskube/cloud/commit/b9f233d3a7f3dbaa5bbdd444bb298c8d8c5b985b))
* **deps:** update dependency @sentry/angular to v8.48.0 ([#233](https://github.com/glasskube/cloud/issues/233)) ([7c2df39](https://github.com/glasskube/cloud/commit/7c2df396de3d0d22201176f95891656d838379dc))
* **deps:** update dependency @sentry/angular to v8.49.0 ([#275](https://github.com/glasskube/cloud/issues/275)) ([8de2f9d](https://github.com/glasskube/cloud/commit/8de2f9ddf8dbfff4016bd917b292e9e2da2bca16))
* **deps:** update dependency globe.gl to v2.35.0 ([#252](https://github.com/glasskube/cloud/issues/252)) ([adfc232](https://github.com/glasskube/cloud/commit/adfc232db2ae1ecc236d7ea7e580fbc30d4e1e02))
* **deps:** update dependency globe.gl to v2.35.1 ([#253](https://github.com/glasskube/cloud/issues/253)) ([c993c74](https://github.com/glasskube/cloud/commit/c993c745f7326f718614707e924859a8df0c937b))
* **deps:** update dependency globe.gl to v2.36.0 ([#256](https://github.com/glasskube/cloud/issues/256)) ([4a8cf2f](https://github.com/glasskube/cloud/commit/4a8cf2fc4c983edfcf872e122974ccdcfcdb6c03))
* **deps:** update dependency globe.gl to v2.36.1 ([#266](https://github.com/glasskube/cloud/issues/266)) ([5892469](https://github.com/glasskube/cloud/commit/58924699a839fd4a63e258a12e4f5456441af5a3))
* **deps:** update dependency globe.gl to v2.37.0 ([#267](https://github.com/glasskube/cloud/issues/267)) ([e43d3a3](https://github.com/glasskube/cloud/commit/e43d3a33bf32432b995819e944ffeb949151fe9f))
* **deps:** update dependency posthog-js to v1.203.3 ([#225](https://github.com/glasskube/cloud/issues/225)) ([674848d](https://github.com/glasskube/cloud/commit/674848d514711e8ebef1013affd916ae7e82dc4d))
* **deps:** update dependency posthog-js to v1.204.0 ([#230](https://github.com/glasskube/cloud/issues/230)) ([3fd67ff](https://github.com/glasskube/cloud/commit/3fd67ff9695b9f19c254b6c53e1c5a0f0b0f9dac))
* **deps:** update dependency posthog-js to v1.205.0 ([#234](https://github.com/glasskube/cloud/issues/234)) ([f96e2f1](https://github.com/glasskube/cloud/commit/f96e2f1a5663ef2a0fd7b01cb0703df956425efd))
* **deps:** update dependency posthog-js to v1.205.1 ([#264](https://github.com/glasskube/cloud/issues/264)) ([7917ffc](https://github.com/glasskube/cloud/commit/7917ffc4b84e66839877e2d808581b786e40989d))
* **deps:** update dependency posthog-js to v1.206.1 ([#273](https://github.com/glasskube/cloud/issues/273)) ([f296cac](https://github.com/glasskube/cloud/commit/f296cac2bbd94390ebf8b9608f11986defd6d32b))
* **deps:** update kubernetes packages to v0.32.1 ([#287](https://github.com/glasskube/cloud/issues/287)) ([02625a4](https://github.com/glasskube/cloud/commit/02625a4aa16544d051891754928f5a19f56f5299))
* **deps:** update module github.com/aws/aws-sdk-go-v2/config to v1.28.10 ([#255](https://github.com/glasskube/cloud/issues/255)) ([75f6c80](https://github.com/glasskube/cloud/commit/75f6c802789ae4be5fd8d66dc95bf865f75facbe))
* **deps:** update module github.com/aws/aws-sdk-go-v2/config to v1.28.8 ([#247](https://github.com/glasskube/cloud/issues/247)) ([14c53e3](https://github.com/glasskube/cloud/commit/14c53e3466959069c40e56a12c6ea52ed5cfa6f9))
* **deps:** update module github.com/wneessen/go-mail to v0.6.0 ([#250](https://github.com/glasskube/cloud/issues/250)) ([45b47b8](https://github.com/glasskube/cloud/commit/45b47b893cc1ceb92c54ce6f2acc008d159bd9c6))
* **deps:** update module github.com/wneessen/go-mail to v0.6.1 ([#257](https://github.com/glasskube/cloud/issues/257)) ([0472d41](https://github.com/glasskube/cloud/commit/0472d416f0316086c1ccd4ce4c81ba893440a3a4))
* **deps:** update module golang.org/x/crypto to v0.32.0 ([#231](https://github.com/glasskube/cloud/issues/231)) ([bc0ef91](https://github.com/glasskube/cloud/commit/bc0ef917cb4e982c9571bae2cf87b98832bedd47))
* **deps:** update module k8s.io/cli-runtime to v0.32.0 ([#260](https://github.com/glasskube/cloud/issues/260)) ([1c47e4f](https://github.com/glasskube/cloud/commit/1c47e4ffe3048be76fbc9b3f48b2930c0a724417))
* down migrations ([#248](https://github.com/glasskube/cloud/issues/248)) ([2b19ed8](https://github.com/glasskube/cloud/commit/2b19ed850f5dd1f8b25a316df63c18646e36f843))
* **ui:** error handling ([#249](https://github.com/glasskube/cloud/issues/249)) ([aec5fcf](https://github.com/glasskube/cloud/commit/aec5fcf98c9d66f55d2992e44dc5562277059587))
* uptime calculation of current hour ([#240](https://github.com/glasskube/cloud/issues/240)) ([b36a2a0](https://github.com/glasskube/cloud/commit/b36a2a09a2bf6ab4ca5fed1b4a3bc31c36bd3124))


### Other

* **deps:** update angular-cli monorepo to v19.0.7 ([#246](https://github.com/glasskube/cloud/issues/246)) ([7a9af7e](https://github.com/glasskube/cloud/commit/7a9af7e663f36996f2da040a2b635d71585c3686))
* **deps:** update cgr.dev/chainguard/static:latest docker digest to 7e1e8a0 ([#243](https://github.com/glasskube/cloud/issues/243)) ([dd3b3a6](https://github.com/glasskube/cloud/commit/dd3b3a6d71a97f2d30158b2b85cfff513cd4628f))
* **deps:** update cgr.dev/chainguard/static:latest docker digest to f96b5a6 ([#229](https://github.com/glasskube/cloud/issues/229)) ([4cf3404](https://github.com/glasskube/cloud/commit/4cf3404433823999aa81c3df2184ee6b4841e87d))
* **deps:** update dependency golangci-lint to v1.63.4 ([#228](https://github.com/glasskube/cloud/issues/228)) ([a4fc4a6](https://github.com/glasskube/cloud/commit/a4fc4a653fc58b805d3894c1d556fed30f6f18b2))
* **deps:** update dependency postcss to v8.5.0 ([#268](https://github.com/glasskube/cloud/issues/268)) ([8885324](https://github.com/glasskube/cloud/commit/8885324faca9dc176cda68959f9033cd62f32a7e))
* **deps:** update dependency postcss to v8.5.1 ([#269](https://github.com/glasskube/cloud/issues/269)) ([247e34e](https://github.com/glasskube/cloud/commit/247e34e7303dbc1815d792cc66661b31eda2839f))
* **deps:** update docker/build-push-action action to v6.11.0 ([#235](https://github.com/glasskube/cloud/issues/235)) ([0a69297](https://github.com/glasskube/cloud/commit/0a69297818619449560b63ee1be27707d20ef082))
* **deps:** update docker/build-push-action action to v6.12.0 ([#278](https://github.com/glasskube/cloud/issues/278)) ([c4a4070](https://github.com/glasskube/cloud/commit/c4a40706947fddb5001026a84f533e2aed43a403))
* **deps:** update gcr.io/distroless/static-debian12:nonroot docker digest to 6ec5aa9 ([#259](https://github.com/glasskube/cloud/issues/259)) ([dfbdf3a](https://github.com/glasskube/cloud/commit/dfbdf3add47f03beb75081266ae8e9b38e463d6e))
* **deps:** update ghcr.io/go-shiori/shiori docker tag to v1.7.4 ([#261](https://github.com/glasskube/cloud/issues/261)) ([1edfac2](https://github.com/glasskube/cloud/commit/1edfac2ac78fb3101330835ebcaad79b8a406bbf))


### Refactoring

* deployment revisions ([#274](https://github.com/glasskube/cloud/issues/274)) ([05490d3](https://github.com/glasskube/cloud/commit/05490d38e6fb306f00480eb2f0f44eb514bae95d))
* move latest deployment to deployment target ([#241](https://github.com/glasskube/cloud/issues/241)) ([cef7f7c](https://github.com/glasskube/cloud/commit/cef7f7c6cd9e78407e971ede50a9f0e1cdf16f57))

## [0.8.2](https://github.com/glasskube/cloud/compare/v0.8.1...v0.8.2) (2025-01-03)


### Other

* force new release (no changes) ([ba0d2ed](https://github.com/glasskube/cloud/commit/ba0d2ed95a82d7c255c84b2d60a803ab32294745))

## [0.8.1](https://github.com/glasskube/cloud/compare/v0.8.0...v0.8.1) (2025-01-03)


### Bug Fixes

* **deps:** update dependency globe.gl to v2.34.6 ([#210](https://github.com/glasskube/cloud/issues/210)) ([e3ca127](https://github.com/glasskube/cloud/commit/e3ca1276c2feb75b062476010336bcfab8211fd2))
* **deps:** update dependency posthog-js to v1.203.2 ([#208](https://github.com/glasskube/cloud/issues/208)) ([87ecf27](https://github.com/glasskube/cloud/commit/87ecf275b72f22d98fa909c45f2efa1d9bf5bd47))
* **deps:** update fontsource monorepo to v5.1.1 ([#211](https://github.com/glasskube/cloud/issues/211)) ([32eeec3](https://github.com/glasskube/cloud/commit/32eeec389d145bc71928d6e4bdccad68944253d2))
* **deps:** update module github.com/getsentry/sentry-go to v0.31.0 ([#219](https://github.com/glasskube/cloud/issues/219)) ([b43ff6e](https://github.com/glasskube/cloud/commit/b43ff6e340ac47be4b1e96c4a1ba2b0624bd6f79))
* **deps:** update module github.com/getsentry/sentry-go to v0.31.1 ([#222](https://github.com/glasskube/cloud/issues/222)) ([6c1cc75](https://github.com/glasskube/cloud/commit/6c1cc75c4c57282337083c8d5c45fc15d8f860d5))
* **deps:** update module github.com/jackc/pgx/v5 to v5.7.2 ([#205](https://github.com/glasskube/cloud/issues/205)) ([7af02c4](https://github.com/glasskube/cloud/commit/7af02c447796807d3f1bc9021ddb5303e6c6ba68))
* **deps:** update module github.com/onsi/gomega to v1.36.2 ([#207](https://github.com/glasskube/cloud/issues/207)) ([5a9eab6](https://github.com/glasskube/cloud/commit/5a9eab6ea4e8f2e6e04a0f2605fe4d3784b826da))
* **ui:** correct step sequence on onboarding wizard ([#214](https://github.com/glasskube/cloud/issues/214)) ([9953db5](https://github.com/glasskube/cloud/commit/9953db52f4d1b06b6380e686354080994cbd571c))
* **ui:** put wizard showing in AfterViewInit to prevent undefined error ([#203](https://github.com/glasskube/cloud/issues/203)) ([3176dae](https://github.com/glasskube/cloud/commit/3176daeefd8dd13a903affb40b63500680554d82))
* **ui:** use correct application version in deploy form ([#215](https://github.com/glasskube/cloud/issues/215)) ([1474d4d](https://github.com/glasskube/cloud/commit/1474d4d97168b192d61be8b3b558684123ac7b70))


### Other

* cleanup obsolete deployment and deployment target statuses ([#216](https://github.com/glasskube/cloud/issues/216)) ([c64bb2e](https://github.com/glasskube/cloud/commit/c64bb2ebaf2f25abe352ec13d892c45b4dc9f8a3))
* **deps:** update dependency @sentry/cli to v2.40.0 ([#218](https://github.com/glasskube/cloud/issues/218)) ([ed26a2a](https://github.com/glasskube/cloud/commit/ed26a2a268fa3697e9717ea67969a7d736f30b1a))
* **deps:** update dependency golangci-lint to v1.63.1 ([#213](https://github.com/glasskube/cloud/issues/213)) ([00704c3](https://github.com/glasskube/cloud/commit/00704c3b813c1cc9b641e48b9ba204034935c3fb))
* **deps:** update dependency golangci-lint to v1.63.2 ([#217](https://github.com/glasskube/cloud/issues/217)) ([c6ddbce](https://github.com/glasskube/cloud/commit/c6ddbcee77080e374fd6ea43ff07b885124d0521))
* **deps:** update dependency golangci-lint to v1.63.3 ([#221](https://github.com/glasskube/cloud/issues/221)) ([112c8f1](https://github.com/glasskube/cloud/commit/112c8f19f3f50cf8ba779cb186317a098f357bcd))
* **deps:** update jdx/mise-action action to v2.1.10 ([#206](https://github.com/glasskube/cloud/issues/206)) ([81ed617](https://github.com/glasskube/cloud/commit/81ed6173170ae27889e6c12d4b374882e0da0b14))
* **deps:** update jdx/mise-action action to v2.1.11 ([#212](https://github.com/glasskube/cloud/issues/212)) ([e2a0526](https://github.com/glasskube/cloud/commit/e2a0526c1a275ad3e19c60e6df2e54e12f79fd1e))
* fix typos in context and middleware naming ([#209](https://github.com/glasskube/cloud/issues/209)) ([18bd9fb](https://github.com/glasskube/cloud/commit/18bd9fb7b0477bf62d84cbd0b00fa3ae16c3bc13))

## [0.8.0](https://github.com/glasskube/cloud/compare/v0.7.0...v0.8.0) (2024-12-20)


### Features

* add email verify resend ([#182](https://github.com/glasskube/cloud/issues/182)) ([7c15950](https://github.com/glasskube/cloud/commit/7c15950b8d366a6cbb30bd3a2f4a711897ba98c4))
* deployment charts ([#194](https://github.com/glasskube/cloud/issues/194)) ([5e63e9b](https://github.com/glasskube/cloud/commit/5e63e9b312e7a075dff20542f19ee31c599035a6))
* release info and sentry sourcemaps ([#200](https://github.com/glasskube/cloud/issues/200)) ([9e942fa](https://github.com/glasskube/cloud/commit/9e942fa0f18e418be3d6e416337c079893c9e793))
* **ui:** add closing overlay with escape ([#191](https://github.com/glasskube/cloud/issues/191)) ([eeb6fca](https://github.com/glasskube/cloud/commit/eeb6fca59198ccc9d15aa73b3b01a1e6e89aa529))


### Bug Fixes

* **deps:** update angular monorepo to v19.0.5 ([#184](https://github.com/glasskube/cloud/issues/184)) ([e64c739](https://github.com/glasskube/cloud/commit/e64c73990a5efbea056eef75f3a5e82dfdb3b99d))
* **deps:** update aws-sdk-go-v2 monorepo ([#197](https://github.com/glasskube/cloud/issues/197)) ([91d4fc6](https://github.com/glasskube/cloud/commit/91d4fc6cf73b8d49ed4a8beed338b7783450f8a9))
* **deps:** update dependency @angular/cdk to v19.0.4 ([#185](https://github.com/glasskube/cloud/issues/185)) ([fa2768d](https://github.com/glasskube/cloud/commit/fa2768ddb82645f527eb74782fbd7d03bf1cc83c))
* **deps:** update dependency @sentry/angular to v8.47.0 ([#183](https://github.com/glasskube/cloud/issues/183)) ([001d07f](https://github.com/glasskube/cloud/commit/001d07face4bf6c9b37763b07309af106ff17acc))
* **deps:** update dependency apexcharts to v4.3.0 ([#196](https://github.com/glasskube/cloud/issues/196)) ([4c49508](https://github.com/glasskube/cloud/commit/4c49508ff37d849feb51a446805a6c77ab99469b))
* **deps:** update dependency globe.gl to v2.34.5 ([#188](https://github.com/glasskube/cloud/issues/188)) ([53a217a](https://github.com/glasskube/cloud/commit/53a217ac8defee826f6e4039401e37dbaed79134))
* **deps:** update dependency posthog-js to v1.202.5 ([#195](https://github.com/glasskube/cloud/issues/195)) ([3713ca4](https://github.com/glasskube/cloud/commit/3713ca4ff3c257d56898d8db224cc9e351f33f80))
* **deps:** update dependency posthog-js to v1.203.1 ([#199](https://github.com/glasskube/cloud/issues/199)) ([0dd46b4](https://github.com/glasskube/cloud/commit/0dd46b48b28eafc868354a67a0fd9de618f6101e))
* **deps:** update module golang.org/x/net to v0.33.0 [security] ([#190](https://github.com/glasskube/cloud/issues/190)) ([a6813d9](https://github.com/glasskube/cloud/commit/a6813d937e66db8f555f47eb2ef8ecd68c99024e))
* sort deployment targets by createdBy ([#189](https://github.com/glasskube/cloud/issues/189)) ([eb3f7d5](https://github.com/glasskube/cloud/commit/eb3f7d5fcb571ce1096a54a1645392705aab0b54))
* **ui:** add height to charts ([#202](https://github.com/glasskube/cloud/issues/202)) ([81311db](https://github.com/glasskube/cloud/commit/81311db0f217d5b7f17568acb26c837e6e53cfed))
* **ui:** fix alignment issue in deployment targets table ([#192](https://github.com/glasskube/cloud/issues/192)) ([be9f450](https://github.com/glasskube/cloud/commit/be9f4506569686827f0929f0d6567426a86c84d1))
* **ui:** fix close animation not shown for some overlays ([#193](https://github.com/glasskube/cloud/issues/193)) ([8ca3f4f](https://github.com/glasskube/cloud/commit/8ca3f4f430910150792cec77e6c6f282fd7a92f3))
* user account update syntax error and error logging ([#201](https://github.com/glasskube/cloud/issues/201)) ([d1665fe](https://github.com/glasskube/cloud/commit/d1665fe227ad07aec10d8b52738d9dbbe3a58753))


### Other

* **deps:** update angular-cli monorepo to v19.0.6 ([#187](https://github.com/glasskube/cloud/issues/187)) ([a5f23bf](https://github.com/glasskube/cloud/commit/a5f23bfa379438e4295ca787e6e4806e4ff1ba63))
* **deps:** update axllent/mailpit docker tag to v1.21.8 ([#198](https://github.com/glasskube/cloud/issues/198)) ([11f76c7](https://github.com/glasskube/cloud/commit/11f76c73cf69be2e1146f99bab2217e8a0317d5a))
* **deps:** update cgr.dev/chainguard/static:latest docker digest to f5fe67a ([#140](https://github.com/glasskube/cloud/issues/140)) ([266bd00](https://github.com/glasskube/cloud/commit/266bd003e60f2017ba1b5d165d823a5b9c104b1d))

## [0.7.0](https://github.com/glasskube/cloud/compare/v0.6.1...v0.7.0) (2024-12-18)


### Features

* add entity sorting ([#179](https://github.com/glasskube/cloud/issues/179)) ([6737060](https://github.com/glasskube/cloud/commit/6737060290373577cbc4d2df7dba7adda031f2c7))
* add password reset ([#171](https://github.com/glasskube/cloud/issues/171)) ([1329d51](https://github.com/glasskube/cloud/commit/1329d512509e667aebd1d5de1a9a051132ea4135))
* only reopen dialog if aborted ([#174](https://github.com/glasskube/cloud/issues/174)) ([e7addc4](https://github.com/glasskube/cloud/commit/e7addc4766609a457c34dc892f54869efd51a5d0))


### Bug Fixes

* **frontend:** disable all action buttons for customer managed deployments ([#180](https://github.com/glasskube/cloud/issues/180)) ([007ced2](https://github.com/glasskube/cloud/commit/007ced21e8272f157574a0e4aab00a8adcf8243e))
* **ui:** guard routes by user role and redirect / depending on role ([#181](https://github.com/glasskube/cloud/issues/181)) ([d929744](https://github.com/glasskube/cloud/commit/d929744853a39382a2761f63cf0e5686f7f53045))


### Other

* update demo data ([#178](https://github.com/glasskube/cloud/issues/178)) ([5dc3fd0](https://github.com/glasskube/cloud/commit/5dc3fd08c175b425916cee9f8a994734e34c2c85))

## [0.6.1](https://github.com/glasskube/cloud/compare/v0.6.0...v0.6.1) (2024-12-18)


### Performance

* **backend:** optimize deployment targets query ([#175](https://github.com/glasskube/cloud/issues/175)) ([f300fcc](https://github.com/glasskube/cloud/commit/f300fcc54143bce7f78cad6e20674937e4e68d81))

## [0.6.0](https://github.com/glasskube/cloud/compare/v0.5.0...v0.6.0) (2024-12-18)


### Features

* **agent:** restart cloud agent ([#173](https://github.com/glasskube/cloud/issues/173)) ([cf5d667](https://github.com/glasskube/cloud/commit/cf5d66764722154feba5ef367e29c292594be803))
* **backend:** add sentry ([#169](https://github.com/glasskube/cloud/issues/169)) ([716987e](https://github.com/glasskube/cloud/commit/716987e80e9e8e2a1d5b0b7a545bf6148f1da614))
* **ui:** text search in tables ([#164](https://github.com/glasskube/cloud/issues/164)) ([4864ee0](https://github.com/glasskube/cloud/commit/4864ee0beef161c8a230f029e6b0e0ea3ac9beed))


### Bug Fixes

* **deps:** update dependency @sentry/angular to v8.46.0 ([#166](https://github.com/glasskube/cloud/issues/166)) ([8f112ac](https://github.com/glasskube/cloud/commit/8f112ac8cecad40a275341467b3cd9ad7eb2d6e4))
* **deps:** update dependency posthog-js to v1.202.2 ([#165](https://github.com/glasskube/cloud/issues/165)) ([57a4524](https://github.com/glasskube/cloud/commit/57a4524460a8771fba8221a2128bf6ef2f6bbf06))


### Other

* add docker-compose project name ([15d62ed](https://github.com/glasskube/cloud/commit/15d62ed7e0ebef213b0ef3f876a4cd79212f8f70))
* **deps:** update dependency tailwindcss to v3.4.17 ([#172](https://github.com/glasskube/cloud/issues/172)) ([9cb7c75](https://github.com/glasskube/cloud/commit/9cb7c759b6448f2263505abc4223b94c16bc8df8))
* log verification mails ([#170](https://github.com/glasskube/cloud/issues/170)) ([9272c2d](https://github.com/glasskube/cloud/commit/9272c2d2511d9eb985e6fe969837e5b80d44a576))

## [0.5.0](https://github.com/glasskube/cloud/compare/v0.4.0...v0.5.0) (2024-12-17)


### Features

* add conditionally disabling deploy button for vendors ([#147](https://github.com/glasskube/cloud/issues/147)) ([cee1f06](https://github.com/glasskube/cloud/commit/cee1f06d4e5973764ba3a76094bcd912d6f80030))
* add deleting applications, deployment targets, user accounts ([#139](https://github.com/glasskube/cloud/issues/139)) ([975ade7](https://github.com/glasskube/cloud/commit/975ade724e8d6450d15b44ae62b384adbffb6c64))
* add login error handling ([#160](https://github.com/glasskube/cloud/issues/160)) ([f5f0a41](https://github.com/glasskube/cloud/commit/f5f0a419d961becfaa2bcd8f79d186e255a9f251))
* add option to copy & paste the verification link ([e7fe821](https://github.com/glasskube/cloud/commit/e7fe8213716ba0872fd8955ccf57e8cb2e845209))
* add vendor as bcc and reply-to in customer invite mail ([#163](https://github.com/glasskube/cloud/issues/163)) ([e1cc4d8](https://github.com/glasskube/cloud/commit/e1cc4d8b99a1b51811e3c24a003017a828bce62b))
* **backend:** add db migrations ([#155](https://github.com/glasskube/cloud/issues/155)) ([87ffd6f](https://github.com/glasskube/cloud/commit/87ffd6f09744ea5b46e29cbc979f7a315e6b46ca))
* don't use a placeholder for password inputs ([66fe23a](https://github.com/glasskube/cloud/commit/66fe23a664b8ea51e6ba27577e6cb6d964fe231c))
* email verification ([#145](https://github.com/glasskube/cloud/issues/145)) ([e78be22](https://github.com/glasskube/cloud/commit/e78be22276abd99d0a9c23701f3baecb65ba1aac))
* make agent interval configurable on backend ([#154](https://github.com/glasskube/cloud/issues/154)) ([78ea860](https://github.com/glasskube/cloud/commit/78ea860d38ef4908ce6f1c9278be91c7a65f8925))
* **ui:** custom confirm dialog ([#151](https://github.com/glasskube/cloud/issues/151)) ([b14ca14](https://github.com/glasskube/cloud/commit/b14ca14252998916a039b99c78c508f03fa4e765))
* use jwt for agent requests ([#149](https://github.com/glasskube/cloud/issues/149)) ([b5329d6](https://github.com/glasskube/cloud/commit/b5329d6ca5628feccc862a2c74fbb2f2415ea950))


### Bug Fixes

* **deps:** update dependency @sentry/angular to v8.45.1 ([#150](https://github.com/glasskube/cloud/issues/150)) ([0e5c81b](https://github.com/glasskube/cloud/commit/0e5c81ba927e83ee0f906a858eae0c0d7824b795))
* **deps:** update dependency globe.gl to v2.34.4 ([#142](https://github.com/glasskube/cloud/issues/142)) ([fa66356](https://github.com/glasskube/cloud/commit/fa663564fb5d0d04a1abae04c52e6776a590ac93))
* **deps:** update dependency posthog-js to v1.200.1 ([#138](https://github.com/glasskube/cloud/issues/138)) ([50b0c4c](https://github.com/glasskube/cloud/commit/50b0c4cd8e8901e65e0e3b65edb3101e6382dd8b))
* **deps:** update dependency posthog-js to v1.200.2 ([#148](https://github.com/glasskube/cloud/issues/148)) ([cf17cb3](https://github.com/glasskube/cloud/commit/cf17cb35841faccc692172b0ab22f965ba5b4e16))
* **deps:** update dependency posthog-js to v1.201.0 ([#153](https://github.com/glasskube/cloud/issues/153)) ([2f6e751](https://github.com/glasskube/cloud/commit/2f6e7510efc57143186204c4594950d8bda5c488))
* **deps:** update dependency posthog-js to v1.201.1 ([#157](https://github.com/glasskube/cloud/issues/157)) ([d35be88](https://github.com/glasskube/cloud/commit/d35be88795e5f0d8ded291b7835993f595e95223))
* **deps:** update dependency posthog-js to v1.202.0 ([#159](https://github.com/glasskube/cloud/issues/159)) ([6ec9af5](https://github.com/glasskube/cloud/commit/6ec9af554cbd3feee3d0a81636be5ec13d93bada))
* **deps:** update dependency posthog-js to v1.202.1 ([#161](https://github.com/glasskube/cloud/issues/161)) ([ec23a25](https://github.com/glasskube/cloud/commit/ec23a2578b4cfdbdaa8b8f06e4c4d06558a8ffd2))
* **deps:** update font awesome to v6.7.2 ([#156](https://github.com/glasskube/cloud/issues/156)) ([adfc25f](https://github.com/glasskube/cloud/commit/adfc25fa85edf554d698902599146c0f2b8ddb08))
* **deps:** update module github.com/go-chi/chi/v5 to v5.2.0 ([#143](https://github.com/glasskube/cloud/issues/143)) ([701b7e3](https://github.com/glasskube/cloud/commit/701b7e306ca35abf3a2f7b9ea456e10f0046d37d))
* don't overwrite user name with empty string during token verification ([#167](https://github.com/glasskube/cloud/issues/167)) ([684bb83](https://github.com/glasskube/cloud/commit/684bb839c181e7567e3b0257ceef927ebc48a306))
* escape query params ([#146](https://github.com/glasskube/cloud/issues/146)) ([156cc1e](https://github.com/glasskube/cloud/commit/156cc1e002b03eda32cd9597e879ce2db91a66c7))
* revert posthog token ([#144](https://github.com/glasskube/cloud/issues/144)) ([9defc58](https://github.com/glasskube/cloud/commit/9defc58213b30c53f40ab523ec2c772ef67d3bfd))
* **ui:** fix wizard dialog on small screens ([#152](https://github.com/glasskube/cloud/issues/152)) ([d5b3e82](https://github.com/glasskube/cloud/commit/d5b3e820cdd2e0479485d95ea1d89469d3ab89e3))
* **ui:** show registration form errors after submit ([#162](https://github.com/glasskube/cloud/issues/162)) ([22eba8c](https://github.com/glasskube/cloud/commit/22eba8cf99fb9203ef01277c86092f39ec8bf302))


### Other

* **deps:** update axllent/mailpit docker tag to v1.21.7 ([#141](https://github.com/glasskube/cloud/issues/141)) ([1c1406e](https://github.com/glasskube/cloud/commit/1c1406ee21651ce35bedbe98b677ee057eed1eca))
* remove unused var ([3acfb20](https://github.com/glasskube/cloud/commit/3acfb209162e501d5855f6f6b98abbcbafc701a1))


### Docs

* add Getting started section to README ([#158](https://github.com/glasskube/cloud/issues/158)) ([987ca22](https://github.com/glasskube/cloud/commit/987ca220be3d6d0dfa8a2f928d4590160b437061))
* remove not needed database init for Getting Started ([dcbf3f0](https://github.com/glasskube/cloud/commit/dcbf3f0e82d69882df0dcd88f3650d510f8b10d6))

## [0.4.0](https://github.com/glasskube/cloud/compare/v0.3.0...v0.4.0) (2024-12-13)


### Features

* **backend:** use transactions where multiple database writes happen ([#129](https://github.com/glasskube/cloud/issues/129)) ([ef23a9f](https://github.com/glasskube/cloud/commit/ef23a9f2d560546150bd210e2c57c18c7b393fd5))
* change email footer ([cb45326](https://github.com/glasskube/cloud/commit/cb45326c5e8b5a0b9d65e7767197919de054fa63))
* **frontend:** add step 0 in onboarding wizard ([#136](https://github.com/glasskube/cloud/issues/136)) ([fda3126](https://github.com/glasskube/cloud/commit/fda312625abd87267db396354e13a42d599c94f6))


### Bug Fixes

* **deps:** update dependency @sentry/angular to v8.45.0 ([#134](https://github.com/glasskube/cloud/issues/134)) ([14ba8ee](https://github.com/glasskube/cloud/commit/14ba8eeb6d1446e6df99ee6699c0997e55b7c209))
* improve customer invite mail ([#132](https://github.com/glasskube/cloud/issues/132)) ([32f121e](https://github.com/glasskube/cloud/commit/32f121e2eee079d7f1c897259f8b5112740dc75b))


### Other

* **deps:** update jdx/mise-action action to v2.1.8 ([#133](https://github.com/glasskube/cloud/issues/133)) ([34c7b30](https://github.com/glasskube/cloud/commit/34c7b300a1b89ccbc13f08792aa691d13d834216))
* **frontend:** subheading to indicate user role ([#135](https://github.com/glasskube/cloud/issues/135)) ([bb316c0](https://github.com/glasskube/cloud/commit/bb316c06ebc2c1e9d6f61be34e4a31e9b046c1a5))

## [0.3.0](https://github.com/glasskube/cloud/compare/v0.2.0...v0.3.0) (2024-12-13)


### Features

* **frontend:** sentry and posthog user identification ([#111](https://github.com/glasskube/cloud/issues/111)) ([f592617](https://github.com/glasskube/cloud/commit/f5926177c13df51a33c91ecd8d01afa580d18ad2))
* **frontend:** version modal ([#122](https://github.com/glasskube/cloud/issues/122)) ([d9305f1](https://github.com/glasskube/cloud/commit/d9305f10e6c7f807ec58c072c60851fd50a3a9de))
* switch table header ordering and placeholders ([#121](https://github.com/glasskube/cloud/issues/121)) ([3d9f5c3](https://github.com/glasskube/cloud/commit/3d9f5c3d12c5d13bb15875d40be47fc8237ad2ea))
* **ui:** update onboarding flow to create customer user ([#123](https://github.com/glasskube/cloud/issues/123)) ([0293620](https://github.com/glasskube/cloud/commit/029362039a0dc3e6dcd336d1749719f1778aff8a))


### Bug Fixes

* **deps:** update angular monorepo to v19.0.4 ([#116](https://github.com/glasskube/cloud/issues/116)) ([f374746](https://github.com/glasskube/cloud/commit/f3747469b3dc094dedebe1fa963fe618001b095c))
* **deps:** update dependency @angular/cdk to v19.0.3 ([#106](https://github.com/glasskube/cloud/issues/106)) ([b7cf95f](https://github.com/glasskube/cloud/commit/b7cf95f3bafd141f76ee67a3792fc64898157c1d))
* **deps:** update dependency @sentry/angular to v8.44.0 ([#112](https://github.com/glasskube/cloud/issues/112)) ([3ec3176](https://github.com/glasskube/cloud/commit/3ec3176a174dc8e855817565e1a01bd8029f3a32))
* **deps:** update dependency globe.gl to v2.34.3 ([#127](https://github.com/glasskube/cloud/issues/127)) ([e7481cf](https://github.com/glasskube/cloud/commit/e7481cf6167b03e9620bbda33ecfc4b56e2a3ef7))
* **deps:** update dependency posthog-js to v1.196.1 ([#114](https://github.com/glasskube/cloud/issues/114)) ([56741a4](https://github.com/glasskube/cloud/commit/56741a4ac741daad6c0ddd2f7d60cb6b2ad63aa4))
* **deps:** update dependency posthog-js to v1.198.0 ([#118](https://github.com/glasskube/cloud/issues/118)) ([d5c3b1b](https://github.com/glasskube/cloud/commit/d5c3b1b9d4c941fa6d90e3e96735da462d267e17))
* **deps:** update dependency posthog-js to v1.199.0 ([#126](https://github.com/glasskube/cloud/issues/126)) ([edd89de](https://github.com/glasskube/cloud/commit/edd89de48f30aa869ceb1d52617df5631c3872a1))
* **deps:** update module github.com/go-chi/jwtauth/v5 to v5.3.2 ([#110](https://github.com/glasskube/cloud/issues/110)) ([55dce33](https://github.com/glasskube/cloud/commit/55dce3327664b171d163057d2ea38990b7810dc0))
* **deps:** update module golang.org/x/crypto to v0.31.0 [security] ([#109](https://github.com/glasskube/cloud/issues/109)) ([2b73573](https://github.com/glasskube/cloud/commit/2b73573129ac7cb1da8f049463955185f06fbb7b))
* **frontend:** success feedback ([#128](https://github.com/glasskube/cloud/issues/128)) ([29d9de9](https://github.com/glasskube/cloud/commit/29d9de94ace771b01ef919b64293e668bc9a1bdb))
* **frontend:** use wizard for new deployments ([#124](https://github.com/glasskube/cloud/issues/124)) ([a8b4bfa](https://github.com/glasskube/cloud/commit/a8b4bfa36e14527811a4d904224d41b3b41ab1d5))
* **ui:** add maxwith and text overflow to name columns ([#119](https://github.com/glasskube/cloud/issues/119)) ([4616a78](https://github.com/glasskube/cloud/commit/4616a781d265ea9ca6ad035e1e37b70475c78857))
* **ui:** move modal submit button and general modal improvements ([#120](https://github.com/glasskube/cloud/issues/120)) ([9b6899d](https://github.com/glasskube/cloud/commit/9b6899df3565efda55dc12e23194421f57ff2f15))


### Other

* add agent tag ([#130](https://github.com/glasskube/cloud/issues/130)) ([804eda3](https://github.com/glasskube/cloud/commit/804eda34ca3f3c1e6f4a00a4666123a66e621754))
* change 'distributor' to 'vendor' everywhere ([#115](https://github.com/glasskube/cloud/issues/115)) ([ea7ed57](https://github.com/glasskube/cloud/commit/ea7ed57db968a767b0d978f3009813c258bebc75))
* **deps:** update angular-cli monorepo to v19.0.5 ([#125](https://github.com/glasskube/cloud/issues/125)) ([e027fd6](https://github.com/glasskube/cloud/commit/e027fd6d78b5bc8773e2f1f39c9be5545ee23e86))
* **ui:** consistent naming ([#117](https://github.com/glasskube/cloud/issues/117)) ([51487f2](https://github.com/glasskube/cloud/commit/51487f2f5c4bd484228f1fdd8c39ec9877dda6de))

## [0.2.0](https://github.com/glasskube/cloud/compare/v0.1.0...v0.2.0) (2024-12-11)


### Features

* add user creation and inviting customers ([#103](https://github.com/glasskube/cloud/issues/103)) ([c2c1d8c](https://github.com/glasskube/cloud/commit/c2c1d8c2dc9ae3d2f288b1a5d576dc193b8b842b))
* customer installation wizard, dashboard charts ([#82](https://github.com/glasskube/cloud/issues/82)) ([cfa74c6](https://github.com/glasskube/cloud/commit/cfa74c67d338e84e5c00e338f00b90f7109fb8ca))


### Bug Fixes

* **frontend:** toast error message should not be 'OK' ([#104](https://github.com/glasskube/cloud/issues/104)) ([10f7a27](https://github.com/glasskube/cloud/commit/10f7a2759021ea82a33ac977a5c3e981d440e838))

## 0.1.0 (2024-12-11)


### Features

* add create application endpoint ([#15](https://github.com/glasskube/cloud/issues/15)) ([fc1f81e](https://github.com/glasskube/cloud/commit/fc1f81ee2229c2c282387a403630f2b1f804e1c4))
* add db tables and load pgx custom types ([#20](https://github.com/glasskube/cloud/issues/20)) ([5678806](https://github.com/glasskube/cloud/commit/5678806573e4186e2cac6b1c58359cfc141bc6e2))
* add foreign key indices and more dummy data ([#49](https://github.com/glasskube/cloud/issues/49)) ([ab36b2a](https://github.com/glasskube/cloud/commit/ab36b2a1a2d9d5e9649fd9cda393d9d145c3aeb6))
* add globe component ([#42](https://github.com/glasskube/cloud/issues/42)) ([1f67c2b](https://github.com/glasskube/cloud/commit/1f67c2b12ae78a55331ee1db6086bc550886b526))
* add limit to body size and file type for compose file ([#44](https://github.com/glasskube/cloud/issues/44)) ([2b1290f](https://github.com/glasskube/cloud/commit/2b1290f7d0fb1b4f9c4db52356ffea8eb50ea2bc))
* add stale status for deployment targets ([#53](https://github.com/glasskube/cloud/issues/53)) ([b48903d](https://github.com/glasskube/cloud/commit/b48903d84866d4dd0aa1b8b5ff05aab37a438ed4))
* add update application endpoint ([#17](https://github.com/glasskube/cloud/issues/17)) ([18ea0ca](https://github.com/glasskube/cloud/commit/18ea0ca547728c7efe6e0b93a8a27e415801497a))
* add user authentication ([#59](https://github.com/glasskube/cloud/issues/59)) ([d77dd22](https://github.com/glasskube/cloud/commit/d77dd221d1b75a9233650fd293c1bff484e8b451))
* add user role ([#94](https://github.com/glasskube/cloud/issues/94)) ([692395a](https://github.com/glasskube/cloud/commit/692395a4a4fe51fdf87edb1a6f1b4e433b251af6))
* agent reports status ([#93](https://github.com/glasskube/cloud/issues/93)) ([51ada87](https://github.com/glasskube/cloud/commit/51ada87221daeb9aa0fe00da7d73f9693fd7fdcb))
* application versions endpoints ([#27](https://github.com/glasskube/cloud/issues/27)) ([f2347bf](https://github.com/glasskube/cloud/commit/f2347bf852ed184057edf2c97e7543e92f657fd1))
* **backend:** add deployment target api ([#34](https://github.com/glasskube/cloud/issues/34)) ([4932d69](https://github.com/glasskube/cloud/commit/4932d69f6e78ec007a1462601e2995d5633028d6))
* **backend:** add mail sending capabilities ([#84](https://github.com/glasskube/cloud/issues/84)) ([1180e55](https://github.com/glasskube/cloud/commit/1180e55fddd704cfe887c4c825e4cadb23518d10))
* **backend:** GET /applications ([#13](https://github.com/glasskube/cloud/issues/13)) ([003f808](https://github.com/glasskube/cloud/commit/003f80876a5aa25fde83f686b6088f63ecd50288))
* cloud agent ([#83](https://github.com/glasskube/cloud/issues/83)) ([570cb1e](https://github.com/glasskube/cloud/commit/570cb1ed16c7cc04df0434541cb0b02931373eda))
* **cloud-ui:** add color scheme switcher ([#19](https://github.com/glasskube/cloud/issues/19)) ([51d2907](https://github.com/glasskube/cloud/commit/51d29077c45246d6ff47784c23152cf7cfc9071c))
* **cloud-ui:** add initial flowbite dashboard layout ([#5](https://github.com/glasskube/cloud/issues/5)) ([e6916fc](https://github.com/glasskube/cloud/commit/e6916fc7770d5ca704370bad909d37370f114fd0))
* deploy applications to deployment targets ([#54](https://github.com/glasskube/cloud/issues/54)) ([85ab2b4](https://github.com/glasskube/cloud/commit/85ab2b412652eff54f6d0d5c81198cffb2ac0ba4))
* **frontend:** form validations ([#95](https://github.com/glasskube/cloud/issues/95)) ([37564d0](https://github.com/glasskube/cloud/commit/37564d0743bc0ad32804fc6088b58fd1b5e4a4fb))
* **frontend:** global error handling ([#100](https://github.com/glasskube/cloud/issues/100)) ([e5b0bbc](https://github.com/glasskube/cloud/commit/e5b0bbcc9db69db686328e94a49e3351e41b9e8a))
* **frontend:** integrate sentry ([#101](https://github.com/glasskube/cloud/issues/101)) ([ad0cbbb](https://github.com/glasskube/cloud/commit/ad0cbbb08742e35a282609215b3a6002bb4aeb8e))
* manage versions ([#38](https://github.com/glasskube/cloud/issues/38)) ([e3f8ad2](https://github.com/glasskube/cloud/commit/e3f8ad24a415646e488d7da428f2b281cbab9109))
* migrate dropdowns to cdk overlay ([#51](https://github.com/glasskube/cloud/issues/51)) ([f222a04](https://github.com/glasskube/cloud/commit/f222a044110dab03ccf7cad2d83f04547b8f58a2))
* onboarding wizard ([#64](https://github.com/glasskube/cloud/issues/64)) ([2d3cd03](https://github.com/glasskube/cloud/commit/2d3cd034a17a2e491f33bd29e9aafa149fcf91db))
* show applications ([#16](https://github.com/glasskube/cloud/issues/16)) ([b9842c3](https://github.com/glasskube/cloud/commit/b9842c3b60b442b6d02f1e1aca92b9aa15ec984b))
* show deployment target status ([#46](https://github.com/glasskube/cloud/issues/46)) ([79cb6fa](https://github.com/glasskube/cloud/commit/79cb6fa8d03ca26c05972bd0dfd8cf8a3c2242e7))
* show deployment target status on globe ([e7002fd](https://github.com/glasskube/cloud/commit/e7002fd5a2709b3530bc4371f7df8e16ed734e5b))
* small ui improvements, add demo data sql ([#61](https://github.com/glasskube/cloud/issues/61)) ([9f257bf](https://github.com/glasskube/cloud/commit/9f257bf2725d912d68e3c6c6566cacc7c86c0129))
* support environment variables ([#58](https://github.com/glasskube/cloud/issues/58)) ([da472f2](https://github.com/glasskube/cloud/commit/da472f22b550854310b2f29634deb121775baac5))
* **ui:** add applications and deployment targets to dashboard ([#43](https://github.com/glasskube/cloud/issues/43)) ([a605e39](https://github.com/glasskube/cloud/commit/a605e39458f7ecbe2b2d1e1c1cb52a62ac6aa40b))
* **ui:** add deployment targets ui ([#36](https://github.com/glasskube/cloud/issues/36)) ([68b7b25](https://github.com/glasskube/cloud/commit/68b7b253f3f291867be7950b5746281a1f658f5b))
* **ui:** add register page ([#71](https://github.com/glasskube/cloud/issues/71)) ([4497334](https://github.com/glasskube/cloud/commit/44973348ba5ed4e0ab55dfe1cabb28fe0aac897c))
* **ui:** edit and create applications ([#25](https://github.com/glasskube/cloud/issues/25)) ([04f45d8](https://github.com/glasskube/cloud/commit/04f45d8e93e4e6c29ab2693aeeefffc6d9778f72))
* **ui:** show deployment target instructions ([#47](https://github.com/glasskube/cloud/issues/47)) ([88d6c38](https://github.com/glasskube/cloud/commit/88d6c38e85e57c55f332a1440de7d5c3c67e1a79))
* use cdk overlay for drawer overlays ([#52](https://github.com/glasskube/cloud/issues/52)) ([85e6456](https://github.com/glasskube/cloud/commit/85e6456b7d9619cbf633dda654fe7f3989b3ecc8))


### Bug Fixes

* align globe center ([#70](https://github.com/glasskube/cloud/issues/70)) ([deece53](https://github.com/glasskube/cloud/commit/deece53c252cfb45d8302dea7130405e1d7c1c79))
* always use index.html for not found files ([301d049](https://github.com/glasskube/cloud/commit/301d0497e2284a568055f270c4e129e0c8ec0442))
* **deps:** update angular monorepo to v19.0.1 ([#7](https://github.com/glasskube/cloud/issues/7)) ([fa628ee](https://github.com/glasskube/cloud/commit/fa628ee078e93edcbd12c18f6205b3e8d985935f))
* **deps:** update angular monorepo to v19.0.2 ([#72](https://github.com/glasskube/cloud/issues/72)) ([14bb184](https://github.com/glasskube/cloud/commit/14bb1843d5c36dd1b4e99e3e4d170f5adcc2770f))
* **deps:** update angular monorepo to v19.0.3 ([#75](https://github.com/glasskube/cloud/issues/75)) ([7f7679f](https://github.com/glasskube/cloud/commit/7f7679fdbf6ea82acdd665068c4c5b834945fcf4))
* **deps:** update dependency @angular/cdk to v19.0.2 ([#76](https://github.com/glasskube/cloud/issues/76)) ([408fbf2](https://github.com/glasskube/cloud/commit/408fbf2cd9b7c59e685c221d88b38dd649730354))
* **deps:** update dependency globe.gl to v2.34.2 ([#57](https://github.com/glasskube/cloud/issues/57)) ([f3a0265](https://github.com/glasskube/cloud/commit/f3a02653c2297dc90d1f4b465e0aaba57caddb42))
* **deps:** update dependency posthog-js to v1.188.0 ([#24](https://github.com/glasskube/cloud/issues/24)) ([afa94e7](https://github.com/glasskube/cloud/commit/afa94e739e8ba314791f1409bb8e3706507a44c1))
* **deps:** update dependency posthog-js to v1.188.1 ([#29](https://github.com/glasskube/cloud/issues/29)) ([67e7448](https://github.com/glasskube/cloud/commit/67e74485d95cce3fbd3443a054f6a910023bfa6c))
* **deps:** update dependency posthog-js to v1.189.0 ([#37](https://github.com/glasskube/cloud/issues/37)) ([b6c00c5](https://github.com/glasskube/cloud/commit/b6c00c5556d39e0dc07f2ddab077d7fe62688aba))
* **deps:** update dependency posthog-js to v1.190.1 ([#39](https://github.com/glasskube/cloud/issues/39)) ([87358d6](https://github.com/glasskube/cloud/commit/87358d69e095ab0fdc63219490706357584c04bf))
* **deps:** update dependency posthog-js to v1.190.2 ([#40](https://github.com/glasskube/cloud/issues/40)) ([fc1c433](https://github.com/glasskube/cloud/commit/fc1c433acf19429a4308eb779085ce0401c53715))
* **deps:** update dependency posthog-js to v1.191.0 ([#45](https://github.com/glasskube/cloud/issues/45)) ([255ca0a](https://github.com/glasskube/cloud/commit/255ca0a83d6cbed5f5260f7cbb549821aae29ec5))
* **deps:** update dependency posthog-js to v1.192.1 ([#50](https://github.com/glasskube/cloud/issues/50)) ([1b00f6c](https://github.com/glasskube/cloud/commit/1b00f6c9cb4b445d3f87b477b44d1b4ea355a809))
* **deps:** update dependency posthog-js to v1.193.1 ([#55](https://github.com/glasskube/cloud/issues/55)) ([dfe3cb8](https://github.com/glasskube/cloud/commit/dfe3cb8538030a2baba0b32f4448146b27a64390))
* **deps:** update dependency posthog-js to v1.194.1 ([#56](https://github.com/glasskube/cloud/issues/56)) ([6eecdec](https://github.com/glasskube/cloud/commit/6eecdecbdd641d21833b5c7251c4321af383b358))
* **deps:** update dependency posthog-js to v1.194.2 ([#60](https://github.com/glasskube/cloud/issues/60)) ([c145062](https://github.com/glasskube/cloud/commit/c1450628dc512df68125cca73ba1aaaad5b798b1))
* **deps:** update dependency posthog-js to v1.194.3 ([#63](https://github.com/glasskube/cloud/issues/63)) ([89f1928](https://github.com/glasskube/cloud/commit/89f1928c50e0a79880d0e7b8022ab89508c6a872))
* **deps:** update dependency posthog-js to v1.194.4 ([#85](https://github.com/glasskube/cloud/issues/85)) ([f47bbb5](https://github.com/glasskube/cloud/commit/f47bbb5ef8c02a71a7bb82a869831639faf6412a))
* **deps:** update dependency posthog-js to v1.194.5 ([#87](https://github.com/glasskube/cloud/issues/87)) ([f8d4dc0](https://github.com/glasskube/cloud/commit/f8d4dc0d4b8007adc82277759f39fa8d19a90236))
* **deps:** update dependency posthog-js to v1.194.6 ([#92](https://github.com/glasskube/cloud/issues/92)) ([524dec4](https://github.com/glasskube/cloud/commit/524dec4f9b96dd821d56b7f566a518d27a1258e6))
* **deps:** update dependency posthog-js to v1.195.0 ([#102](https://github.com/glasskube/cloud/issues/102)) ([4ce0930](https://github.com/glasskube/cloud/commit/4ce0930dd31f410467f20fc703f689fd0b41608c))
* **deps:** update module github.com/lestrrat-go/jwx/v2 to v2.0.21 [security] ([#68](https://github.com/glasskube/cloud/issues/68)) ([f4042fd](https://github.com/glasskube/cloud/commit/f4042fd781b15a2b2754193dce39984511a5490f))
* **deps:** update module github.com/lestrrat-go/jwx/v2 to v2.1.3 ([#73](https://github.com/glasskube/cloud/issues/73)) ([9c65205](https://github.com/glasskube/cloud/commit/9c65205798d343ee3d816754093032002431fd82))
* **deps:** update module github.com/onsi/gomega to v1.36.1 ([#97](https://github.com/glasskube/cloud/issues/97)) ([133a623](https://github.com/glasskube/cloud/commit/133a623b1b483219033037643375e1a779d7dfe7))
* **deps:** update module golang.org/x/crypto to v0.30.0 ([#77](https://github.com/glasskube/cloud/issues/77)) ([3d8d925](https://github.com/glasskube/cloud/commit/3d8d9259e2c744f56b99adfb47201f1140641561))
* **frontend:** logout if token expired or 401 response ([#99](https://github.com/glasskube/cloud/issues/99)) ([fac37da](https://github.com/glasskube/cloud/commit/fac37da215b46ffc2afa0cdac8fd18c127c78e21))
* toggle sidebar on tiny screens ([#96](https://github.com/glasskube/cloud/issues/96)) ([0e31121](https://github.com/glasskube/cloud/commit/0e3112185754d25bc953d5c54aef808cc1a62071))
* **ui:** application cache ([#41](https://github.com/glasskube/cloud/issues/41)) ([a50fb26](https://github.com/glasskube/cloud/commit/a50fb26ad17d3a9d1a165bb290a310dd91eedcd9))


### Other

* add basic go app with file server ([3a45ef9](https://github.com/glasskube/cloud/commit/3a45ef9a6b10ba528a6a4200528ac3b42353c440))
* add CHANGELOG.md to prettierignore file ([8da206b](https://github.com/glasskube/cloud/commit/8da206b851bad3514bd7b3a9e460219f6e0aec73))
* add dockerfile ([efc29b5](https://github.com/glasskube/cloud/commit/efc29b58352d988eed295d85140f76ee8c9a85f4))
* add golangci-lint ([622bf04](https://github.com/glasskube/cloud/commit/622bf04c4cf696328dbe77bad92b8b03104ebcf8))
* add missing fonts ([#21](https://github.com/glasskube/cloud/issues/21)) ([e114de9](https://github.com/glasskube/cloud/commit/e114de917d5dbe77e88a24f51fff4a8b1033a9b9))
* add posthog-js ([#8](https://github.com/glasskube/cloud/issues/8)) ([32b1ad0](https://github.com/glasskube/cloud/commit/32b1ad0ecb06eb57570b706144cabd4ab015e8cf))
* add prettier ([#11](https://github.com/glasskube/cloud/issues/11)) ([8c91f1d](https://github.com/glasskube/cloud/commit/8c91f1d35ec0952a0a7fe05f8070b57ee96201d7))
* add proxy config for dev ([449eb7f](https://github.com/glasskube/cloud/commit/449eb7f833ec20b920eafb366de2897cd560a047))
* add renovate automerge config ([4b3887a](https://github.com/glasskube/cloud/commit/4b3887a03fd0bbd0eef776ec529a3cbfb8ceecf7))
* add routing with chi ([090536f](https://github.com/glasskube/cloud/commit/090536f64b57752208e08320f8d1206a13603f36))
* add tailwind ([b5ee219](https://github.com/glasskube/cloud/commit/b5ee2194756b0211699ddb778c41f5bed1183ed8))
* change docker base to chainguard ([7457a87](https://github.com/glasskube/cloud/commit/7457a87102e404f139b56196712061817113e0d7))
* change fs embed path ([c0122eb](https://github.com/glasskube/cloud/commit/c0122eb66638e609b485cc670e77fa7800bb604c))
* configure Renovate ([43a2617](https://github.com/glasskube/cloud/commit/43a26179dcecb419f0fef9bb7571a7b5f1fd62c3))
* create angular app ([aedf40c](https://github.com/glasskube/cloud/commit/aedf40c85797f79cb136a21b65fea4e841d1b045))
* delete duplicate html ([69ea918](https://github.com/glasskube/cloud/commit/69ea918ead9a52e9423f3065d0729cba76caa5a9))
* **deps:** update Angular to v19.0.0 ([45db8c9](https://github.com/glasskube/cloud/commit/45db8c9bbcb5f86da5372c71b9e82aff8ba56eb7))
* **deps:** update angular-cli monorepo to v19.0.1 ([#18](https://github.com/glasskube/cloud/issues/18)) ([50a8c4e](https://github.com/glasskube/cloud/commit/50a8c4ef67bacb00514a00ddb8f4ce2cc2b6b65c))
* **deps:** update angular-cli monorepo to v19.0.2 ([#26](https://github.com/glasskube/cloud/issues/26)) ([136fb8f](https://github.com/glasskube/cloud/commit/136fb8fe694ec92e581f8fd90f82175771795822))
* **deps:** update angular-cli monorepo to v19.0.3 ([#74](https://github.com/glasskube/cloud/issues/74)) ([44a0551](https://github.com/glasskube/cloud/commit/44a0551a34e244d0bcf6df8c4cf60a734255d138))
* **deps:** update angular-cli monorepo to v19.0.4 ([#86](https://github.com/glasskube/cloud/issues/86)) ([361e448](https://github.com/glasskube/cloud/commit/361e448cbd6205c0c3c6d54a38d16ef0e656baa4))
* **deps:** update axllent/mailpit docker tag to v1.21.5 ([#88](https://github.com/glasskube/cloud/issues/88)) ([c583e74](https://github.com/glasskube/cloud/commit/c583e74da60d94354db6f26865d32d8f91b9e78e))
* **deps:** update axllent/mailpit docker tag to v1.21.6 ([#91](https://github.com/glasskube/cloud/issues/91)) ([b1d694c](https://github.com/glasskube/cloud/commit/b1d694c61a9fec0254911a7d19be0981f93651d6))
* **deps:** update cgr.dev/chainguard/static:latest docker digest to 5ff428f ([#14](https://github.com/glasskube/cloud/issues/14)) ([aab5ae8](https://github.com/glasskube/cloud/commit/aab5ae8e22c9f850c38e5498d43423fc1e2ac9b2))
* **deps:** update dependency @types/jasmine to v5.1.5 ([#48](https://github.com/glasskube/cloud/issues/48)) ([056df29](https://github.com/glasskube/cloud/commit/056df296ee25664672904434a57d0d76e863df4f))
* **deps:** update dependency jasmine-core to ~5.4.0 ([#2](https://github.com/glasskube/cloud/issues/2)) ([f4123f4](https://github.com/glasskube/cloud/commit/f4123f4011b53e02c750e761165d60487d8c930d))
* **deps:** update dependency jasmine-core to ~5.5.0 ([#62](https://github.com/glasskube/cloud/issues/62)) ([e5c43d9](https://github.com/glasskube/cloud/commit/e5c43d9e60b24fb1afb8b9de92fe271256001aac))
* **deps:** update dependency prettier to v3.4.0 ([#28](https://github.com/glasskube/cloud/issues/28)) ([198dc31](https://github.com/glasskube/cloud/commit/198dc31114c1155294172b4b058847e2043bb9b8))
* **deps:** update dependency prettier to v3.4.1 ([#35](https://github.com/glasskube/cloud/issues/35)) ([7cf4e6c](https://github.com/glasskube/cloud/commit/7cf4e6c595c582442c20a8a3c22e3d692f748a56))
* **deps:** update dependency prettier to v3.4.2 ([#67](https://github.com/glasskube/cloud/issues/67)) ([11db641](https://github.com/glasskube/cloud/commit/11db6415e65a7330e27a48bdc05687273c6c46bb))
* **deps:** update dependency tailwindcss to v3.4.16 ([#65](https://github.com/glasskube/cloud/issues/65)) ([5732d9f](https://github.com/glasskube/cloud/commit/5732d9f29566997dcf25de7dc189f9b4b79ea183))
* **deps:** update dependency typescript to ~5.6.0 ([#3](https://github.com/glasskube/cloud/issues/3)) ([de8296a](https://github.com/glasskube/cloud/commit/de8296a3fd0e9939ba66e1462d60f21592b48e24))
* **deps:** update node.js to v22.11.0 ([#6](https://github.com/glasskube/cloud/issues/6)) ([af79add](https://github.com/glasskube/cloud/commit/af79add744c4b344706d89eb68c1a779fc603322))
* **deps:** update node.js to v22.12.0 ([#66](https://github.com/glasskube/cloud/issues/66)) ([575d176](https://github.com/glasskube/cloud/commit/575d1763266b76faaef4b0763d804b1e57827331))
* **deps:** update postgres docker tag to v17.2 ([#23](https://github.com/glasskube/cloud/issues/23)) ([eef377c](https://github.com/glasskube/cloud/commit/eef377c541649b9e6e2df6f3379621e8ffcdcfa7))
* init angular ([a3450af](https://github.com/glasskube/cloud/commit/a3450af130564068110ee346abb8fcec2eed85f2))
* rename frontend to cloud-ui ([a685e98](https://github.com/glasskube/cloud/commit/a685e985b209d475b835764aee4bdcaa7409f2cf))
* run go mod tidy ([9f53a60](https://github.com/glasskube/cloud/commit/9f53a60fb20d911dafd30b565eb16c9c23c0fcc5))
* set angular analytics to false ([0c9b912](https://github.com/glasskube/cloud/commit/0c9b912e8aae9dc3cc1ec48c8a52f8cd908bc3ca))
* set next release to 0.1.0 ([f0a6d3e](https://github.com/glasskube/cloud/commit/f0a6d3e87c23098a3f7d07c214a65d10f5a7a6b5))
* update dummy data ([3acd07e](https://github.com/glasskube/cloud/commit/3acd07e4b89f8cf194d117df5d4e8fa254077278))


### Docs

* remove outdated information from README ([fb05128](https://github.com/glasskube/cloud/commit/fb051280c48ae6890ec7a66e2713010790280c93))


### Refactoring

* **cloud-ui:** use *ngIf to display alerts, don't use flowbite-angular, remove duplicate code ([#12](https://github.com/glasskube/cloud/issues/12)) ([fca2bc2](https://github.com/glasskube/cloud/commit/fca2bc2ddb40fd4f28ec59b1c5e73005eb0c5abf))
* reorganize backend modules to better separate service init, routing and serving ([#90](https://github.com/glasskube/cloud/issues/90)) ([f9dd232](https://github.com/glasskube/cloud/commit/f9dd232afef547a4242bd4349383a1dfae65a683))
