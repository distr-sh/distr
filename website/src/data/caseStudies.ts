import type {ImageMetadata} from 'astro';

// Import images
import basedashLogoLight from '../assets/case-studies/basedash/Basedash_Light.svg';
import derekReynolds from '../assets/case-studies/basedash/reynolds.jpeg';
import lerianLogo from '../assets/case-studies/lerian-logo.png';
import anshGupta from '../assets/case-studies/sophris/ansh-gupta.jpeg';
import sophrisLogo from '../assets/case-studies/sophris/sophris-ai-logo.png';
import jeffersonRodrigues from '../assets/testimonials/testimonial-7.jpg';

export interface CaseStudy {
  slug: string;
  title: string;
  company: string;
  industry: string;
  useCase: string;
  ctoName: string;
  ctoTitle: string;
  ctoQuote: string;
  logo: ImageMetadata;
  ctoImage: ImageMetadata;
  challenge: string;
  solution: string;
  result: string;
  pageTitle: string;
  pageDescription: string;
}

export const caseStudies: CaseStudy[] = [
  {
    slug: 'lerian',
    title: 'Lerian Studio',
    company: 'Lerian',
    industry: 'Banking/Financial Infrastructure',
    useCase: 'Lifecycle Management Platform',
    ctoName: 'Jefferson Rodrigues',
    ctoTitle: 'Co-Founder & CTO',
    ctoQuote:
      'Our main goal is to simplify the daily operations. No more manual installations, updates, or rollbacks — everything can now be handled with a single click with Distr.',
    logo: lerianLogo,
    ctoImage: jeffersonRodrigues,
    pageTitle: 'Lerian Studio Case Study',
    pageDescription:
      'How Lerian uses Distr to power their Lifecycle Management platform for banking and financial infrastructure',
    challenge: `<a href="https://lerian.studio" target="_blank" rel="noopener noreferrer">Lerian</a> provides banking and financial infrastructure solutions that need to run in highly regulated, secure environments. Their customers in the financial sector require on-premises deployments with strict compliance, data protection, and security standards. Traditional deployment approaches created significant operational friction: manual installations, complex update procedures, error-prone rollbacks, and limited visibility into system health across multiple customer environments.

"<strong>In the financial sector, infrastructure shouldn't be a barrier to innovation,</strong>" explains Jefferson Rodrigues, Co-Founder & CTO at Lerian. "<strong>Our customers need the same speed and agility of cloud-native deployments, but within their own Kubernetes environments. We were spending too much time on operational overhead—manual deployments, coordinating updates with customer IT teams, and troubleshooting issues without proper visibility.</strong>"

The team needed a solution that would:

<ul class="list-disc list-inside">
  <li>Enable standardized, repeatable deployments across multiple customer environments</li>
  <li>Provide real-time visibility into deployment status and application health</li>
  <li>Support instant rollbacks when issues occurred</li>
  <li>Maintain complete traceability for compliance and audit requirements</li>
  <li>Reduce the operational burden on both Lerian's team and their customers' DevOps teams</li>
</ul>`,

    solution: `Lerian adopted Distr to power their Lifecycle Management platform, transforming how they distribute and manage applications in customer-controlled Kubernetes environments. By building on Distr's open-source foundation, Lerian created a comprehensive lifecycle management system that handles installations, updates, rollbacks, and monitoring—all while maintaining the security and control their financial services customers require.

<strong>Key implementation highlights:</strong>

<ul class="list-disc list-inside">
  <li><strong>Bring Your Own Cluster (BYOC) deployments:</strong> Customers run Lerian services in their own Kubernetes environments, meeting strict compliance requirements while leveraging standardized deployment workflows</li>
  <li><strong>Declarative deployments with versioned templates:</strong> All installations are predictable, fully traceable operations using Helm charts and OCI images, eliminating the inefficiency of manual scripts</li>
  <li><strong>Integrated monitoring dashboard:</strong> Real-time visibility into deployed versions, application health, container logs, and agent status—providing 100% visibility for internal teams without compromising customer autonomy</li>
  <li><strong>One-click rollbacks:</strong> Instant reversion to previous versions with automatic rollback in seconds, dramatically reducing Mean Time To Recovery (MTTR) and eliminating long investigation windows</li>
  <li><strong>Token-protected distribution:</strong> Secure access to Helm repositories and OCI images ensures deployment integrity across all customer environments</li>
</ul>

By leveraging Distr's infrastructure, Lerian can focus on their core banking and financial services features while providing enterprise-grade deployment capabilities. Their <a href="https://docs.lerian.studio/en/lifecycle-management" target="_blank" rel="noopener noreferrer">comprehensive documentation</a> demonstrates how customers can deploy, update, and manage Lerian services with the same ease as SaaS products—while maintaining complete control over their infrastructure.`,

    result: `Lerian's Lifecycle Management platform, powered by Distr, has transformed their operational efficiency and customer experience:

<ul class="list-disc list-inside">
  <li><strong>Smoother internal operations:</strong> Standardized deployments mean any squad can deploy new versions without opening tickets, validating features in staging with full traceability</li>
  <li><strong>Faster development cycles:</strong> Execution teams gained more control and autonomy, accelerating the entire development lifecycle</li>
  <li><strong>Reduced operational load:</strong> DevOps teams at both Lerian and their customers spend significantly less time on deployment coordination and troubleshooting</li>
  <li><strong>100% guaranteed traceability:</strong> All changes are versioned and visually organized, bringing governance to operations and improving collaboration across engineering, product, and ops teams</li>
  <li><strong>Elimination of deployment risks:</strong> Automatic rollback capabilities reduce recovery time from hours or days to seconds</li>
</ul>

The solution has proven particularly valuable in the financial sector, where Lerian operates. Banking, messaging, and latency-sensitive services require precision, control, and efficiency in managing distributed applications. By adopting Distr, Lerian ensures infrastructure is no longer a barrier to innovation—instead, it's an enabler.

Today, Lerian's customers benefit from the transparency, control, and collaboration that comes with open-source solutions, aligned with Lerian's philosophy of building trustworthy financial infrastructure. Learn more about how they implemented Lifecycle Management in their <a href="https://docs.lerian.studio/en/lifecycle-management" target="_blank" rel="noopener noreferrer">documentation</a> and <a href="https://docs.lerian.studio/en/using-lifecycle-management" target="_blank" rel="noopener noreferrer">user guides</a>.`,
  },
  {
    slug: 'sophris',
    title: 'Sophris.ai',
    company: 'Sophris.ai',
    industry: 'AI/Engineering Tools',
    useCase: 'Circuit Board Validation',
    ctoName: 'Ansh Gupta',
    ctoTitle: 'CTO',
    ctoQuote:
      'Distr eliminated nearly all deployment headaches. Updates that used to take days now take minutes.',
    logo: sophrisLogo,
    ctoImage: anshGupta,
    pageTitle: 'Sophris.ai Case Study',
    pageDescription:
      'How Sophris.ai uses Distr to streamline on-premises software distribution',
    challenge: `<a href="https://www.sophris.ai/" target="_blank" rel="noopener noreferrer">Sophris</a> uses AI to automate error detection in circuit board schematics. Their platform automates what traditionally is a highly manual and error-prone process of verifying hundreds of intricate components against complex data sheets.

Initially, Sophris deployed their software directly onto virtual machines within customer environments, which quickly proved challenging. Deployments relied heavily on customers' internal IT teams, resulting in delays, misconfigurations, and slow iterations. Their engineering team often spent valuable hours troubleshooting simple file transfer and deployment issues. Sophris needed a smoother, more efficient deployment solution to maintain agility, guarantee customer success, and reduce reliance on slow-moving customer IT teams.

"<strong>At the start, we would send zip files directly to the customer IT team,</strong>" says Ansh Gupta, CTO at Sophris. "<strong>Often, simple mistakes, like not extracting a file correctly, caused significant deployment delays, draining our resources and affecting our speed. We needed a better way.</strong>"`,

    solution: `After evaluating multiple deployment solutions, Sophris chose Glasskube's Distr platform. Distr provided a straightforward yet powerful alternative, simplifying on-premises software distribution through an intuitive Docker Compose-based approach.

"<strong>We initially considered other solutions but found them overly complex and cost-prohibitive for our stage,</strong>" Gupta explained. "<strong>Distr was simple, intuitive, and exactly what we needed.</strong>"

With Distr, Sophris quickly standarized their deployment workflow. Instead of manual file transfers and troubleshooting deployment scripts, their engineers could now deploy software updates with a few clicks. Updates became significantly faster, enabling Sophris to iterate at a speed previously hard to achieve.

"<strong>Distr eliminated nearly all deployment headaches. Updates that used to take days now take minutes,</strong>" Gupta added. "<strong>This was especially crucial when we have limited access to client infrastructure.</strong>"`,

    result: `Distr significantly reduced Sophris's deployment time, enabling them to rapidly iterate and deliver continuous improvements to their clients. Sophris went from tedious, manual deployments dependent on external IT teams to seamless, self-managed updates.

By choosing Distr, Sophris improved their on-premises distribution experience, simplified updates, and freed up valuable engineering time, allowing them to focus on innovating and enhancing their core offering.`,
  },
  {
    slug: 'basedash',
    title: 'Basedash',
    company: 'Basedash',
    industry: 'Developer Tools',
    useCase: 'Self-Hosted Deployment',
    ctoName: 'Derek Reynolds',
    ctoTitle: 'Product Engineer',
    ctoQuote:
      'Having a dedicated space for all our self-hosted customers that can manage authenticated registry access is great.',
    logo: basedashLogoLight,
    ctoImage: derekReynolds,
    pageTitle: 'Basedash Case Study',
    pageDescription:
      'How Basedash uses Distr to deliver and manage self-hosted deployments for their customers',
    challenge: `<a href="https://basedash.com" target="_blank" rel="noopener noreferrer">Basedash</a> is AI-native business intelligence: teams use natural language to generate dashboards, reports, insights, and charts in seconds—no SQL required. Many of their customers run Basedash in their own environments for data control and compliance. Supporting those self-hosted deployments used to mean handing out registry credentials manually, keeping spreadsheets of who had access to what, and answering the same setup questions over and over.

"<strong>We needed one place where every self-hosted customer could get in, grab what they need, and deploy without us having to send tokens or run custom scripts,</strong>" says Derek Reynolds, Product Engineer at Basedash.

The team was looking for:

<ul class="list-disc list-inside">
  <li>A single platform where customers could manage their own deployment targets and credentials</li>
  <li>Private container registry with fine-grained access control</li>
  <li>Flexibility: some customers want a fully-managed feeling or self-hosting assistance; others already have their own bespoke setup for deploying self-hosted apps and just want artifact access.</li>
</ul>`,

    solution: `Basedash chose Distr to run their self-hosted distribution. Distr supports all deployment use cases out of the box—from fully-managed agent-based deployments where you can push updates directly into customer infrastructure to teams who pull images from the registry themselves—and works for self-hosted customers at every level. Most importantly, it gives Basedash one central place for all their self-managed customers. Distr is the "dedicated space" their team and customers use every day.

<strong>How they use it:</strong>

<ul class="list-disc list-inside">
  <li><strong>Agent deployment (recommended):</strong> Customers install the Distr agent with a single command from the customer portal. The agent pulls images, runs Docker Compose, and reports status. Basedash gets automatic updates and health visibility without touching customer servers.</li>
  <li><strong>Container deployment:</strong> Teams that already have a setup for deploying self-hosted apps can pull images from the Distr registry using a PAT. They get full control over when and how to deploy.</li>
  <li><strong>One place for all self-hosted customers:</strong> Whether a customer uses the agent or pulls images themselves, every self-hosted deployment is visible and manageable from the same platform.</li>
</ul>

Basedash documents the full flow—agent install, registry auth, and machine specs—in their <a href="https://docs.basedash.com/self-hosting/deploy" target="_blank" rel="noopener noreferrer">self-hosting deploy guide</a>, so customers can get up and running without back-and-forth.`,

    result: `Basedash's self-hosted offering now runs through Distr. Customers get a clear path to deploy (agent or registry), and the team has a single place to manage self-hosted customers.

<ul class="list-disc list-inside">
  <li><strong>Less manual work:</strong> No more ad-hoc token generation or credential spreadsheets. The Distr platform handles it.</li>
  <li><strong>Better security:</strong> Each target has its own credentials. Revoking access or rotating secrets is straightforward.</li>
  <li><strong>Fewer support loops:</strong> The customer portal gives self-hosted customers a place to generate access tokens and get installation commands on their own—no back-and-forth with the Basedash team.</li>
</ul>

For a product that ships both as SaaS and self-hosted, having a dedicated space for self-hosted customers has made the whole process simpler for everyone.`,
  },
];
