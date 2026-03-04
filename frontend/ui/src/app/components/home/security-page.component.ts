// Customer Portal Security page — displays CVE reports uploaded by vendor.
// Vendor uploads CVE reports as ApplicationVersionResources with type 'cve-report'.
// This page parses those reports and displays Docker Hub-style severity breakdown.
import {NgClass, TitleCasePipe} from '@angular/common';
import {Component, computed, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faChevronUp, faDownload, faFileShield, faShieldHalved} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {MOCK_RESOURCES, MockResource} from './mock-data';

@Component({
  selector: 'app-security-page',
  imports: [FaIconComponent, TitleCasePipe, NgClass],
  template: `
    <div class="bg-gray-50 dark:bg-gray-900 min-h-screen">
      <div class="mx-auto max-w-screen-xl px-4 py-6 sm:px-6 lg:px-8">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Security</h1>

        <!-- CVE Reports by Version -->
        @if (cveReports().length > 0) {
          <div class="space-y-6">
            @for (report of cveReports(); track report.resource.version) {
              <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
                <!-- Header with version and severity breakdown -->
                <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
                  <div class="flex items-start justify-between mb-3">
                    <div>
                      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ report.resource.version }}</h2>
                      <p class="text-sm text-gray-500 dark:text-gray-400">
                        Scanned {{ formatDate(report.data.reportDate) }}
                      </p>
                    </div>
                    <button
                      (click)="toggleVersion(report.resource.version)"
                      class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors">
                      <fa-icon [icon]="faDownload" size="sm"></fa-icon>
                      Download Report
                    </button>
                  </div>

                  <!-- Severity breakdown (Docker Hub style) -->
                  <div class="flex items-center gap-3 flex-wrap">
                    <div class="flex items-center gap-1.5">
                      <span class="text-xs font-medium text-gray-600 dark:text-gray-400">Total:</span>
                      <span class="text-sm font-semibold text-gray-900 dark:text-white">{{
                        report.data.totalCVEs
                      }}</span>
                    </div>
                    @if (report.data.critical > 0) {
                      <div class="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-red-100 dark:bg-red-900/30">
                        <span class="w-2 h-2 rounded-full bg-red-600 dark:bg-red-500"></span>
                        <span class="text-xs font-medium text-red-800 dark:text-red-300"
                          >Critical: {{ report.data.critical }}</span
                        >
                      </div>
                    }
                    @if (report.data.high > 0) {
                      <div class="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-orange-100 dark:bg-orange-900/30">
                        <span class="w-2 h-2 rounded-full bg-orange-600 dark:bg-orange-500"></span>
                        <span class="text-xs font-medium text-orange-800 dark:text-orange-300"
                          >High: {{ report.data.high }}</span
                        >
                      </div>
                    }
                    @if (report.data.medium > 0) {
                      <div class="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-yellow-100 dark:bg-yellow-900/30">
                        <span class="w-2 h-2 rounded-full bg-yellow-600 dark:bg-yellow-500"></span>
                        <span class="text-xs font-medium text-yellow-800 dark:text-yellow-300"
                          >Medium: {{ report.data.medium }}</span
                        >
                      </div>
                    }
                    @if (report.data.low > 0) {
                      <div class="flex items-center gap-1.5 px-2 py-1 rounded-lg bg-gray-100 dark:bg-gray-700">
                        <span class="w-2 h-2 rounded-full bg-gray-600 dark:bg-gray-400"></span>
                        <span class="text-xs font-medium text-gray-800 dark:text-gray-300"
                          >Low: {{ report.data.low }}</span
                        >
                      </div>
                    }
                  </div>
                </div>

                <!-- Expandable CVE list -->
                <div class="px-5 py-3">
                  <button
                    (click)="toggleVersion(report.resource.version)"
                    class="w-full flex items-center justify-between text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors">
                    <span
                      >{{ expandedVersions().has(report.resource.version) ? 'Hide' : 'Show' }}
                      {{ report.data.totalCVEs }} vulnerabilities</span
                    >
                    <fa-icon
                      [icon]="expandedVersions().has(report.resource.version) ? faChevronUp : faChevronDown"
                      size="sm"></fa-icon>
                  </button>
                </div>

                @if (expandedVersions().has(report.resource.version)) {
                  <div class="border-t border-gray-200 dark:border-gray-700">
                    <div class="overflow-x-auto">
                      <table class="w-full text-sm text-left text-gray-500 dark:text-gray-400">
                        <thead class="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                          <tr>
                            <th scope="col" class="px-5 py-3">CVE ID</th>
                            <th scope="col" class="px-5 py-3">Severity</th>
                            <th scope="col" class="px-5 py-3">Component</th>
                            <th scope="col" class="px-5 py-3">Current Version</th>
                            <th scope="col" class="px-5 py-3">Fixed In</th>
                            <th scope="col" class="px-5 py-3">Published</th>
                          </tr>
                        </thead>
                        <tbody>
                          @for (cve of report.data.cves; track cve.id) {
                            <tr class="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50">
                              <td class="px-5 py-3">
                                <a
                                  href="https://nvd.nist.gov/vuln/detail/{{ cve.id }}"
                                  target="_blank"
                                  class="font-mono text-xs text-primary-600 dark:text-primary-400 hover:underline">
                                  {{ cve.id }}
                                </a>
                              </td>
                              <td class="px-5 py-3">
                                <span
                                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                                  [ngClass]="{
                                    'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300':
                                      cve.severity === 'critical',
                                    'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300':
                                      cve.severity === 'high',
                                    'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300':
                                      cve.severity === 'medium',
                                    'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300':
                                      cve.severity === 'low',
                                  }">
                                  {{ cve.severity | titlecase }}
                                </span>
                              </td>
                              <td class="px-5 py-3">
                                <div class="font-mono text-xs text-gray-900 dark:text-white">{{ cve.component }}</div>
                              </td>
                              <td class="px-5 py-3">
                                <div class="font-mono text-xs text-gray-900 dark:text-white">{{ cve.version }}</div>
                              </td>
                              <td class="px-5 py-3">
                                @if (cve.fixedVersion) {
                                  <div class="font-mono text-xs text-green-600 dark:text-green-400">
                                    {{ cve.fixedVersion }}
                                  </div>
                                } @else {
                                  <span class="text-xs text-gray-400 dark:text-gray-500 italic">No fix</span>
                                }
                              </td>
                              <td class="px-5 py-3">
                                <div class="text-xs text-gray-500 dark:text-gray-400">
                                  {{ formatDate(cve.publishedDate) }}
                                </div>
                              </td>
                            </tr>
                            <tr class="border-b dark:border-gray-700">
                              <td colspan="6" class="px-5 py-2 bg-gray-50 dark:bg-gray-800/50">
                                <p class="text-xs text-gray-600 dark:text-gray-400">{{ cve.description }}</p>
                              </td>
                            </tr>
                          }
                        </tbody>
                      </table>
                    </div>
                  </div>
                }
              </div>
            }
          </div>
        } @else {
          <!-- No CVE reports available -->
          <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 p-8">
            <div class="text-center">
              <fa-icon [icon]="faShieldHalved" class="text-4xl text-gray-300 dark:text-gray-600 mb-3"></fa-icon>
              <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-2">No security reports available</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                CVE reports will appear here when your vendor uploads them for specific versions.
              </p>
            </div>
          </div>
        }

        <!-- SBOM section -->
        @if (sbomResources().length > 0) {
          <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 mt-6">
            <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Software Bill of Materials
              </h2>
            </div>
            <div class="px-5 py-4">
              <!-- MOCK DATA — replace with ApplicationsService.getResources() filtered for SBOM types -->
              <div class="grid grid-cols-1 gap-3">
                @for (resource of sbomResources(); track resource.name) {
                  <div
                    class="flex items-center justify-between p-4 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                    <div class="flex items-center gap-3">
                      <fa-icon [icon]="faFileShield" class="text-gray-400 dark:text-gray-500"></fa-icon>
                      <div>
                        <div class="text-sm font-medium text-gray-900 dark:text-white">{{ resource.name }}</div>
                        <div class="text-xs text-gray-500 dark:text-gray-400">Version {{ resource.version }}</div>
                      </div>
                    </div>
                    <button
                      class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/20 rounded-lg transition-colors">
                      <fa-icon [icon]="faDownload" size="sm"></fa-icon>
                      Download
                    </button>
                  </div>
                }
              </div>
            </div>
          </div>
        }
      </div>
    </div>
  `,
})
export class SecurityPageComponent {
  protected readonly faShieldHalved = faShieldHalved;
  protected readonly faFileShield = faFileShield;
  protected readonly faDownload = faDownload;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronUp = faChevronUp;

  protected readonly expandedVersions = signal<Set<string>>(new Set());

  // MOCK DATA — replace with ApplicationsService.getResources() filtered for SBOM types
  protected readonly sbomResources = signal<MockResource[]>(
    MOCK_RESOURCES.filter((r) => r.type === 'sbom-spdx' || r.type === 'sbom-cyclonedx')
  );

  // MOCK DATA — replace with ApplicationsService.getResources() filtered for CVE reports
  protected readonly cveReports = computed(() => {
    const reports = MOCK_RESOURCES.filter((r) => r.type === 'cve-report' && r.cveData);
    return reports.map((resource) => ({
      resource,
      data: resource.cveData!,
    }));
  });

  protected toggleVersion(version: string): void {
    this.expandedVersions.update((versions) => {
      const newSet = new Set(versions);
      if (newSet.has(version)) {
        newSet.delete(version);
      } else {
        newSet.add(version);
      }
      return newSet;
    });
  }

  protected formatDate(dateString: string): string {
    return dayjs(dateString).format('MMM D, YYYY');
  }
}
