---
company: 'Lerian'
person:
  name: 'Jefferson Rodrigues'
  role: 'Co-Founder & CTO'
  image: '/src/assets/customers/lerian/jefferson-rodrigues.jpg'
quote: 'Our main goal is to simplify the daily operations. No more manual installations, updates, or rollbacks — everything can now be handled with a single click with Distr.'
industry: 'Banking/Financial Infrastructure'
useCase: 'Lifecycle Management Platform'
featured: true
outcome: 'Manual operations become one-click workflows'
caseStudy:
  title: 'Lerian Studio'
  logo: '/src/assets/customers/lerian/logo.png'
  pageTitle: 'Lerian Studio Case Study'
  pageDescription: 'How Lerian uses Distr to power their Lifecycle Management platform for banking and financial infrastructure'
---

## Challenge

[Lerian](https://lerian.studio) provides banking and financial infrastructure solutions that need to run in highly regulated, secure environments. Their customers in the financial sector require on-premises deployments with strict compliance, data protection, and security standards. Traditional deployment approaches created significant operational friction: manual installations, complex update procedures, error-prone rollbacks, and limited visibility into system health across multiple customer environments.

"**In the financial sector, infrastructure shouldn't be a barrier to innovation,**" explains Jefferson Rodrigues, Co-Founder & CTO at Lerian. "**Our customers need the same speed and agility of cloud-native deployments, but within their own Kubernetes environments. We were spending too much time on operational overhead—manual deployments, coordinating updates with customer IT teams, and troubleshooting issues without proper visibility.**"

The team needed a solution that would:

- Enable standardized, repeatable deployments across multiple customer environments
- Provide real-time visibility into deployment status and application health
- Support instant rollbacks when issues occurred
- Maintain complete traceability for compliance and audit requirements
- Reduce the operational burden on both Lerian's team and their customers' DevOps teams

## Solution

Lerian adopted Distr to power their Lifecycle Management platform, transforming how they distribute and manage applications in customer-controlled Kubernetes environments. By building on Distr's open-source foundation, Lerian created a comprehensive lifecycle management system that handles installations, updates, rollbacks, and monitoring—all while maintaining the security and control their financial services customers require.

**Key implementation highlights:**

- **Bring Your Own Cluster (BYOC) deployments:** Customers run Lerian services in their own Kubernetes environments, meeting strict compliance requirements while leveraging standardized deployment workflows
- **Declarative deployments with versioned templates:** All installations are predictable, fully traceable operations using Helm charts and OCI images, eliminating the inefficiency of manual scripts
- **Integrated monitoring dashboard:** Real-time visibility into deployed versions, application health, container logs, and agent status—providing 100% visibility for internal teams without compromising customer autonomy
- **One-click rollbacks:** Instant reversion to previous versions with automatic rollback in seconds, dramatically reducing Mean Time To Recovery (MTTR) and eliminating long investigation windows
- **Token-protected distribution:** Secure access to Helm repositories and OCI images ensures deployment integrity across all customer environments

By leveraging Distr's infrastructure, Lerian can focus on their core banking and financial services features while providing enterprise-grade deployment capabilities. Their [comprehensive documentation](https://docs.lerian.studio/en/lifecycle-management) demonstrates how customers can deploy, update, and manage Lerian services with the same ease as SaaS products—while maintaining complete control over their infrastructure.

## Result

Lerian's Lifecycle Management platform, powered by Distr, has transformed their operational efficiency and customer experience:

- **Smoother internal operations:** Standardized deployments mean any squad can deploy new versions without opening tickets, validating features in staging with full traceability
- **Faster development cycles:** Execution teams gained more control and autonomy, accelerating the entire development lifecycle
- **Reduced operational load:** DevOps teams at both Lerian and their customers spend significantly less time on deployment coordination and troubleshooting
- **100% guaranteed traceability:** All changes are versioned and visually organized, bringing governance to operations and improving collaboration across engineering, product, and ops teams
- **Elimination of deployment risks:** Automatic rollback capabilities reduce recovery time from hours or days to seconds

The solution has proven particularly valuable in the financial sector, where Lerian operates. Banking, messaging, and latency-sensitive services require precision, control, and efficiency in managing distributed applications. By adopting Distr, Lerian ensures infrastructure is no longer a barrier to innovation—instead, it's an enabler.

Today, Lerian's customers benefit from the transparency, control, and collaboration that comes with open-source solutions, aligned with Lerian's philosophy of building trustworthy financial infrastructure. Learn more about how they implemented Lifecycle Management in their [documentation](https://docs.lerian.studio/en/lifecycle-management) and [user guides](https://docs.lerian.studio/en/using-lifecycle-management).
