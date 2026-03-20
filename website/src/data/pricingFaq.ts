export type PricingFAQ = {
  id: string;
  question: string;
  answer: string;
};

export const PricingFAQs: PricingFAQ[] = [
  {
    id: 'what-is-distr',
    question: 'What is Distr?',
    answer:
      'Distr is an Open Source software distribution platform that provides a ready-to-use setup with prebuilt components to help software and AI companies distribute applications to customers in complex, self-managed environments.',
  },
  {
    id: 'plan-choice',
    question: 'Which plan should I choose?',
    answer: `
    <strong>Starter</strong><br/>
    Choose Starter if you're validating customer-install GTM for the first time. Usually this is one internal champion testing the flow. Agent-based installs and Docker make setup extremely fast — you ship updates quickly, iterate rapidly, and customers get the newest version immediately. Ideal for fast learning and proving distribution works before scaling.<br/><br/>

    <strong>Pro</strong><br/>
    Choose Pro once you're past POC and rolling out to multiple customers. Platform/DevOps teams typically take over here. If scalabitly is important, deployments move to Kubernetes/Helm or Artifact-based distribution. Focus shifts from pure speed to control: version visibility, license enforcement, SSO/RBAC access control across both your team and your customers' teams.<br/><br/>

    <strong>Enterprise</strong><br/>
    Choose Enterprise when Distr becomes your entire commercial self-hosted delivery suite. You're not just distributing software anymore — but also documentation, support portals, public images, automated workflows, dynamic licensing, usage-based billing and more — all inside one platform. Enterprise includes dedicated infrastructure isolation and full white-label control.
    `,
  },
  {
    id: 'pricing-model',
    question: 'How does pricing work?',
    answer:
      'Pricing is based on two metrics: internal users (your team operating Distr) and customer organizations (install targets). Your monthly price is calculated based on the number of internal users AND the number of customer organizations.',
  },
  {
    id: 'internal-user',
    question: 'What is an internal user?',
    answer:
      'An internal user is a member of your team who operates Distr. Internal users can manage applications, deployments, licenses, customer organizations, and other platform settings. In Pro, Enterprise, and Pro Trial plans, internal users can be assigned different roles (Administrator, User, or Viewer) with role-based access control (RBAC) to control what they can access and modify. In the Starter plan, all internal users automatically have Administrator privileges. Learn more about <a href="/docs/product/rbac/" class="text-[#00b5eb] hover:underline">roles and user management</a>.',
  },
  {
    id: 'customer-organization',
    question: 'What is a customer?',
    answer:
      'A customer represents one of your end customers organizations who will install and use your software in their own environment. Each customer organization gets access to their own Customer Portal where they can view deployments, download artifacts, and manage their installation. Customer users (multiple users per customer organization with role-based access control) are only available in Pro, Enterprise, and Pro Trial plans. In the Starter plan, each customer organization is limited to one user. Learn more about <a href="/docs/product/rbac/" class="text-[#00b5eb] hover:underline">customer roles and user management</a>.',
  },

  {
    id: 'how-long-to-integrate',
    question: 'How long does it take to integrate Distr?',
    answer:
      'Most teams ship their first customer install within right after our onboarding. We support GitHub release automation, pre/post install scripts, and agent based distributions out of the box. To make sure you get unlocked fast — Starter includes an onboarding call, and Pro includes white glove onboarding.',
  },
  {
    id: 'self-hosting',
    question: 'Can I self-host Distr?',
    answer:
      'Yes, Distr is fully Open Source and you can self-host it. We offer a Community Edition that you can self-host for free, and an Enterprise Edition with advanced features. For details on self-hosting options, deployment methods, and getting started, see our <a href="/docs/self-hosting/getting-started/" class="text-[#00b5eb] hover:underline">self-hosting documentation</a>.',
  },
  {
    id: 'support',
    question: 'Where do I get support?',
    answer:
      'Starter includes basic email support. Pro adds private Slack support. Enterprise includes a dedicated support engineer and SLA.',
  },
  {
    id: 'change-plan',
    question: 'How do I upgrade or downgrade my plan?',
    answer:
      'You can add customers and internal users within your current plan limits directly through the Vendor Portal. However, to upgrade or downgrade your subscription plan (e.g., from Starter to Pro, or Pro to Enterprise), please contact us via email at support@disr.sh. Our team will help you change your plan.',
  },
  {
    id: 'change-billing-cycle',
    question:
      'Can I change my billing cycle from monthly to yearly (or vice versa)?',
    answer:
      'To change your billing cycle (e.g., from monthly to yearly or yearly to monthly), please contact us via email at support@disr.sh. Our team will help you switch between billing cycles.',
  },
];
