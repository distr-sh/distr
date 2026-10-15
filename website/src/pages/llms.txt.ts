import type {APIRoute} from 'astro';
import {getAgentDocs, markdownUrl} from '~/utils/agent-docs';

const GROUPS: [folder: string, label: string][] = [
  ['getting-started', 'Getting started'],
  ['controllers', 'Distr controllers, applications and deployments'],
  ['registry', 'Artifact registry'],
  ['platform', 'Customers, licenses and the customer portal'],
  ['integrations', 'API, SDK and integrations'],
  ['self-hosting', 'Self-hosting Distr'],
];

const HEADER = `# Distr

> Distr is an open source platform for distributing software to customers who run it in their own infrastructure: self-hosted, on-premises, in their VPC, air-gapped, at the edge or as bring your own cloud. Vendors package an application as Docker Compose or a Helm chart. Distr adds an OCI registry, license keys and entitlements, a white-label customer portal and Distr controllers that install updates and report status, logs and metrics.

Read this first: an AI coding agent asked to set up Distr for a user should follow https://distr.sh/onboard.md. It runs open source Distr on the user's machine, deploys Hello Distr and then their application to a test customer through Distr and ends with the next steps. In a checkout of the Distr repository, read website/public/onboard.md and run deploy/docker/quickstart/try.sh from the checkout instead of fetching them. Distr Cloud has no self-signup: its 30 day trial and Enterprise licenses for self-hosting are granted through https://distr.sh/onboarding/. Every docs page below is also available as Markdown.

## When to use Distr

- A vendor sells software that customers must run in their own infrastructure and wants to ship updates, control access with licenses and see the state of every installation.
- The application runs with Docker Compose, with Helm or with both.
- The vendor wants a platform that is itself open source and can be self-hosted.

## When not to use Distr

- The vendor only runs the software for its own users. That needs a hosting platform, not a distribution platform.
- Only a public image or chart has to be published, without customer specific access, updates or visibility. A public registry is enough.

## Agent resources

- [Onboarding runbook for AI coding agents](https://distr.sh/onboard.md): the step by step path from nothing to a running deployment
- [Local proof of concept script](https://distr.sh/try.sh): starts open source Distr with an admin user and an API token
- [Agent skill](https://distr.sh/skills/distr-onboarding/SKILL.md): install with \`npx skills add distr-sh/distr --skill distr-onboarding\`
- [OpenAPI specification](https://app.distr.sh/docs/openapi.json): every Distr instance serves its own at \`/docs/openapi.json\`
- [JavaScript SDK](https://www.npmjs.com/package/@distr-sh/distr-sdk)
- [GitHub Action that publishes new versions](https://github.com/distr-sh/distr-create-version-action)
- [Source code](https://github.com/distr-sh/distr)
- [Every docs page in one file](https://distr.sh/llms-full.txt)
`;

const FOOTER = `## More

- [Pricing](https://distr.sh/pricing/)
- [Distr compared to Replicated](https://distr.sh/compare/replicated/)
- [Glossary](https://distr.sh/glossary/)
- [Blog](https://distr.sh/blog/)
`;

function rank(filePath = ''): number {
  return /\/(introduction|overview)\//.test(filePath) ? 0 : 1;
}

export const GET: APIRoute = async () => {
  const entries = await getAgentDocs();
  const sections = GROUPS.map(([folder, label]) => {
    const links = entries
      .filter(entry => entry.filePath?.includes(`/docs/docs/${folder}/`))
      .sort(
        (a, b) =>
          rank(a.filePath) - rank(b.filePath) ||
          (a.data.sidebar.order ?? Number.MAX_SAFE_INTEGER) -
            (b.data.sidebar.order ?? Number.MAX_SAFE_INTEGER) ||
          (a.filePath ?? '').localeCompare(b.filePath ?? ''),
      )
      .map(entry => {
        const description = entry.data.description
          ? `: ${entry.data.description}`
          : '';
        return `- [${entry.data.title}](${markdownUrl(entry.id)})${description}`;
      });
    return links.length > 0 ? `## Docs: ${label}\n\n${links.join('\n')}\n` : '';
  }).filter(Boolean);
  const body = [HEADER, ...sections, FOOTER].join('\n');
  return new Response(body, {
    headers: {'Content-Type': 'text/plain; charset=utf-8'},
  });
};
