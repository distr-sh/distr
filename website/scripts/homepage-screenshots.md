# Homepage Screenshot Capture

Use `pnpm screenshots:homepage` from the `website` directory to recreate the
Distr product screenshots used on the homepage. The script captures at
`1920x1080` output size with a 125% browser-equivalent zoom by using a
`1536x864` CSS viewport and `deviceScaleFactor: 1.25`.

Credentials are not stored in git. When environment variables are not set, the
script prompts for them.

```sh
pnpm screenshots:homepage
```

For non-interactive runs:

```sh
DISTR_VENDOR_EMAIL=vendor@example.com \
DISTR_CUSTOMER_EMAIL=customer@example.com \
DISTR_DEMO_PASSWORD='...' \
pnpm screenshots:homepage
```

If the customer and vendor accounts use different passwords, set
`DISTR_VENDOR_PASSWORD` and `DISTR_CUSTOMER_PASSWORD` instead of
`DISTR_DEMO_PASSWORD`.

Useful options:

```sh
pnpm screenshots:homepage -- --list
pnpm screenshots:homepage -- --out-dir /tmp/distr-homepage-screenshots
pnpm screenshots:homepage -- --headed
pnpm screenshots:homepage -- --reuse-storage
```

## Screenshot Plan

| Asset                                               | Login    | Route                                               | What must be visible                                                                                                                                                 |
| --------------------------------------------------- | -------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `distr-deployments-{light,dark}.webp`               | Vendor   | `/deployments`                                      | Deployment overview grouped by customer, current sidebar, deployment target cards, app versions, health states, gauges, and deploy/update/inspect controls.          |
| `distr-artifacts-{light,dark}.webp`                 | Vendor   | `/artifact-pulls`                                   | Registry > Downloads table with pull date, customer, user, address, artifact, and version so viewers can see who downloaded what and when.                           |
| `distr-artifact-licenses-{light,dark}.webp`         | Vendor   | `/licenses/ebdc5a9f-ea69-4344-9006-0ec1d882141a`    | Artifact entitlement drawer open with the artifact tag selector expanded.                                                                                            |
| `distr-customer-portal-artifacts-{light,dark}.webp` | Customer | `/home`                                             | White-label customer portal home page with onboarding instructions and install snippets. The light capture is also copied to `distr-customer-portal-artifacts.webp`. |
| `distr-dashboard-{light,dark}.webp`                 | Vendor   | `/dashboard`                                        | Vendor dashboard with support bundles, agent cards, customer names, health states, and current navigation.                                                           |
| `distr-customer-portal-{light,dark}.webp`           | Customer | `/deployments`                                      | Customer deployments page with the Create New Deployment modal open on the application-selection step.                                                               |
| `distr-log-viewer-{light,dark}.webp`                | Vendor   | `/deployments/2ad1125e-1d38-4457-80bf-5c8d043686a8` | Deployment target Agent Logs view with timestamps, log levels, source files, metrics, filters, sorting, and export controls.                                         |
