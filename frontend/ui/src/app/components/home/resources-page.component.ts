// Customer Portal Resources & Docs page — surfaces additional resources from all application versions.
// Expects data from: ApplicationsService.getResources() for version-attached resources.
// Tab 1: Resources grouped by version. Tab 2: Setup & Installation instructions.
import {Component, computed, signal} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBoxOpen,
  faCubes,
  faDownload,
  faExternalLinkAlt,
  faFileLines,
  faFileShield,
  faKey,
  faTerminal,
} from '@fortawesome/free-solid-svg-icons';
import {MOCK_RESOURCES, MockResource} from './mock-data';

@Component({
  selector: 'app-resources-page',
  imports: [FaIconComponent, RouterLink],
  template: `
    <div class="bg-gray-50 dark:bg-gray-900 min-h-screen">
      <div class="mx-auto max-w-screen-xl px-4 py-6 sm:px-6 lg:px-8">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Resources & Docs</h1>

        <!-- Tabs -->
        <div class="border-b border-gray-200 dark:border-gray-700 mb-6">
          <ul class="flex gap-4 -mb-px text-sm font-medium">
            <li>
              <button
                (click)="activeTab.set('resources')"
                class="inline-flex items-center gap-2 px-1 pb-3 border-b-2 transition-colors"
                [class.border-primary-600]="activeTab() === 'resources'"
                [class.text-primary-600]="activeTab() === 'resources'"
                [class.dark:border-primary-400]="activeTab() === 'resources'"
                [class.dark:text-primary-400]="activeTab() === 'resources'"
                [class.border-transparent]="activeTab() !== 'resources'"
                [class.text-gray-500]="activeTab() !== 'resources'"
                [class.hover:text-gray-700]="activeTab() !== 'resources'"
                [class.dark:text-gray-400]="activeTab() !== 'resources'"
                [class.dark:hover:text-gray-300]="activeTab() !== 'resources'">
                <fa-icon [icon]="faFileLines"></fa-icon>
                Resources
              </button>
            </li>
            <li>
              <button
                (click)="activeTab.set('setup')"
                class="inline-flex items-center gap-2 px-1 pb-3 border-b-2 transition-colors"
                [class.border-primary-600]="activeTab() === 'setup'"
                [class.text-primary-600]="activeTab() === 'setup'"
                [class.dark:border-primary-400]="activeTab() === 'setup'"
                [class.dark:text-primary-400]="activeTab() === 'setup'"
                [class.border-transparent]="activeTab() !== 'setup'"
                [class.text-gray-500]="activeTab() !== 'setup'"
                [class.hover:text-gray-700]="activeTab() !== 'setup'"
                [class.dark:text-gray-400]="activeTab() !== 'setup'"
                [class.dark:hover:text-gray-300]="activeTab() !== 'setup'">
                <fa-icon [icon]="faTerminal"></fa-icon>
                Setup & Installation
              </button>
            </li>
          </ul>
        </div>

        @if (activeTab() === 'resources') {
          @if (groupedResources().length > 0) {
            <!-- MOCK DATA — replace with ApplicationsService.getResources() per version -->
            @for (group of groupedResources(); track group.version) {
              <div class="mb-6">
                <h3 class="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3">
                  <span
                    class="font-mono bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded text-gray-900 dark:text-white"
                    >{{ group.version }}</span
                  >
                </h3>
                <div class="grid grid-cols-1 gap-3">
                  @for (resource of group.resources; track resource.name) {
                    <div
                      class="flex items-center justify-between p-4 rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                      <div class="flex items-center gap-3 min-w-0">
                        <fa-icon
                          [icon]="getResourceIcon(resource)"
                          class="text-gray-400 dark:text-gray-500 flex-shrink-0"></fa-icon>
                        <div class="min-w-0">
                          <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
                            {{ resource.name }}
                          </div>
                          <div class="text-xs text-gray-500 dark:text-gray-400">
                            {{ getResourceTypeLabel(resource) }}
                          </div>
                        </div>
                      </div>
                      @if (resource.format === 'download') {
                        <button
                          class="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded-lg transition-colors">
                          <fa-icon [icon]="faDownload" size="sm"></fa-icon>
                          Download
                        </button>
                      } @else {
                        <button
                          class="flex-shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded-lg transition-colors">
                          <fa-icon [icon]="faExternalLinkAlt" size="sm"></fa-icon>
                          Open
                        </button>
                      }
                    </div>
                  }
                </div>
              </div>
            }
          } @else {
            <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 p-12">
              <div class="flex flex-col items-center justify-center text-center text-gray-500 dark:text-gray-400">
                <fa-icon [icon]="faBoxOpen" class="text-4xl text-gray-300 dark:text-gray-600 mb-3"></fa-icon>
                <p class="text-sm font-medium">Your vendor hasn't attached any resources yet.</p>
                <p class="text-xs mt-1">Documentation, release notes, and other files will appear here when added.</p>
              </div>
            </div>
          }
        }

        @if (activeTab() === 'setup') {
          <div class="space-y-6">
            <!-- Installation instructions -->
            <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
              <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Installation
                </h2>
              </div>
              <div class="px-5 py-4">
                <p class="text-sm text-gray-700 dark:text-gray-300 mb-4">
                  Follow the instructions below to set up the Distr agent in your environment. The agent connects to the
                  control plane and manages deployments automatically.
                </p>
                <div
                  class="rounded-lg bg-gray-900 dark:bg-gray-950 p-4 font-mono text-sm text-gray-100 overflow-x-auto">
                  <div class="text-gray-500"># Install the Distr agent</div>
                  <div>curl -fsSL https://get.distr.sh | bash</div>
                  <div class="mt-2 text-gray-500"># Or use Docker</div>
                  <div>docker run -d --name distr-agent \\</div>
                  <div class="pl-4">-e DISTR_TOKEN=&lt;your-token&gt; \\</div>
                  <div class="pl-4">ghcr.io/glasskube/distr/agent:latest</div>
                </div>
              </div>
            </div>

            <!-- Access tokens -->
            <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
              <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Access Tokens
                </h2>
              </div>
              <div class="px-5 py-4">
                <p class="text-sm text-gray-700 dark:text-gray-300 mb-3">
                  Manage personal access tokens for API and agent authentication.
                </p>
                <a
                  routerLink="/settings/access-tokens"
                  class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-900 bg-white border border-gray-200 rounded-lg hover:bg-gray-100 hover:text-primary-700 focus:z-10 focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:text-white dark:hover:bg-gray-700">
                  <fa-icon [icon]="faKey"></fa-icon>
                  Manage Access Tokens
                </a>
              </div>
            </div>
          </div>
        }
      </div>
    </div>
  `,
})
export class ResourcesPageComponent {
  protected readonly faFileLines = faFileLines;
  protected readonly faFileShield = faFileShield;
  protected readonly faCubes = faCubes;
  protected readonly faDownload = faDownload;
  protected readonly faExternalLinkAlt = faExternalLinkAlt;
  protected readonly faTerminal = faTerminal;
  protected readonly faKey = faKey;
  protected readonly faBoxOpen = faBoxOpen;

  protected readonly activeTab = signal<'resources' | 'setup'>('resources');

  // MOCK DATA — replace with ApplicationsService.getResources() for each version
  private readonly allResources = signal<MockResource[]>(MOCK_RESOURCES);

  protected readonly groupedResources = computed(() => {
    const resources = this.allResources();
    const grouped = new Map<string, MockResource[]>();
    for (const r of resources) {
      const existing = grouped.get(r.version) ?? [];
      existing.push(r);
      grouped.set(r.version, existing);
    }
    return Array.from(grouped.entries())
      .map(([version, resources]) => ({version, resources}))
      .sort((a, b) => b.version.localeCompare(a.version));
  });

  protected getResourceIcon(resource: MockResource) {
    switch (resource.type) {
      case 'release-notes':
        return this.faFileLines;
      case 'helm-chart':
        return this.faCubes;
      case 'sbom-spdx':
      case 'sbom-cyclonedx':
        return this.faFileShield;
      default:
        return this.faFileLines;
    }
  }

  protected getResourceTypeLabel(resource: MockResource): string {
    switch (resource.type) {
      case 'release-notes':
        return 'Release Notes';
      case 'helm-chart':
        return 'Helm Chart';
      case 'sbom-spdx':
        return 'SBOM (SPDX)';
      case 'sbom-cyclonedx':
        return 'SBOM (CycloneDX)';
      case 'guide':
        return 'Guide';
      case 'document':
        return 'Document';
      case 'cve-report':
        return 'CVE Report';
    }
  }
}
