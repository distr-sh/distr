# Contributing

Thank you for your interest in contributing to Distr!

Distr is open-source software licensed under the [Apache 2.0 license](https://github.com/distr-sh/distr/blob/main/LICENSE) and accepts contributions via GitHub pull requests.

## Communications

To avoid unnecessary and redundant work, please reach out before you start working on your contribution.

You can either create an issue on GitHub or join our [community forum](https://github.com/distr-sh/distr/discussions) to get in touch with the community.

## How to run distr for development

To run the Distr Hub locally, clone the repository and make sure that all necessary tools defined in [mise.toml](mise.toml) are installed.
We recommend that you use [mise](https://mise.jdx.dev/) to install these (run `mise install` in the current directory) but you don't have to.

We set the environment variable `DISTR_ENV` via `mise` (apply with `mise env` in the current directory),
which points to the `.env.development.local` file containing reasonable defaults.
However, you are also free to use any other way to provide your environment variables to the Distr Hub.

You can then start the necessary containers and the Distr Hub with:

```shell
# Start the database and a mock SMTP server
docker compose up -d
# Start Distr Hub
mise watch serve -r
```

Open your browser and navigate to [`http://localhost:8080/register`](http://localhost:8080/register) to register a user
and receive the E-Mail verification link via Mailpit on [`http://localhost:8025`](http://localhost:8025).

### Stripe Integration

In order to use the Stripe integration during development, the `stripe` CLI tool must be installed.
If you use `mise` for setup, it will be installed automatically.
Use `stripe login` to authenticate with Stripe (make sure to use a sandbox account).

Use `stripe listen --forward-to localhost:8080/api/v1/webhook/stripe` to forward Stripe webhook events to your local server.
Create an API key on the [Stripe Dashboard](https://dashboard.stripe.com/test/apikeys) with write access to "Checkout Sessions", "Customer Portal",
"Subscription" and read access to "Prices".
Add the `STRIPE_WEBHOOK_SECRET` and `STRIPE_API_KEY` variables to your `.env.development.local.secret` file.

In your sandbox account, you need at least some of the price resources with the following lookup keys:

- `distr_starter_customer_monthly`
- `distr_starter_customer_yearly`
- `distr_starter_user_monthly`
- `distr_starter_user_yearly`
- `distr_pro_customer_monthly`
- `distr_pro_customer_yearly`
- `distr_pro_user_monthly`
- `distr_pro_user_yearly`

## Backporting bugfixes

If the `main` branch already contains changes that would warrant a major or minor version bump but there is need to create a patch release only,
it is possible to backport commits by pushing to the relevant `v*.*.x` branch.
For example, if a commit should be added to version 1.2.3, it must be pushed to the `v1.2.x` branch.

**Important:** Please keep in mind the following rules for backporting:

1. Do not backport changes that would require an inappropriate version bump. For example, do not add new features to the `v1.2.x` branch, only bugfixes.
2. Only backport changes that are already in `main`. Ideally, use `git cherry-pick`.

## Pre-Releases

Creating pre-releases can be useful if you need to test the current state of the main branch in a demo environment or test artifact building.
To do this, create a `v*.*.*-rc.x` branch (for example, `v2.1.0-rc.x`).
For release-please to create a new release branch in the correct format, change `"prerelease": true` and `"versioning": "prerelease"` in the `release-please-config.json`.
This will also automatically mark your GitHub release as pre-release.
