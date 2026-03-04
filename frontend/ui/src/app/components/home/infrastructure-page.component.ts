// Customer Portal Infrastructure page — shows agent status + infrastructure configuration questionnaire.
// Questionnaire allows customers to submit deployment configuration (cloud provider, region, networking, SAML, etc.)
// and track provisioning status through stages: Submitted → Scheduled → Provisioning → Ready.
// MOCK DATA: The questionnaire form and provisioning status are entirely mocked for UI demonstration.
import {DatePipe, TitleCasePipe} from '@angular/common';
import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faCheck,
  faCircle,
  faClipboardList,
  faClock,
  faRocket,
  faServer,
  faSpinner,
  faTerminal,
} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {isStale} from '../../../util/model';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';

type ProvisioningStage = 'not_submitted' | 'submitted' | 'scheduled' | 'provisioning' | 'ready';

@Component({
  selector: 'app-infrastructure-page',
  imports: [FaIconComponent, DatePipe, TitleCasePipe, ReactiveFormsModule],
  template: `
    <div class="bg-gray-50 dark:bg-gray-900 min-h-screen">
      <div class="mx-auto max-w-screen-xl px-4 py-6 sm:px-6 lg:px-8">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Infrastructure</h1>

        @if (firstTarget(); as target) {
          <!-- Agent connection status -->
          <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 mb-6">
            <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Agent Status
              </h2>
            </div>
            <div class="px-5 py-6">
              <div class="flex items-center gap-6">
                <!-- Large status indicator -->
                <div class="flex-shrink-0">
                  <div
                    class="size-16 rounded-full flex items-center justify-center"
                    [class.bg-lime-100]="agentStatus() === 'live'"
                    [class.dark:bg-lime-900/30]="agentStatus() === 'live'"
                    [class.bg-yellow-100]="agentStatus() === 'stale'"
                    [class.dark:bg-yellow-900/30]="agentStatus() === 'stale'"
                    [class.bg-gray-100]="agentStatus() === 'disconnected'"
                    [class.dark:bg-gray-700]="agentStatus() === 'disconnected'">
                    <div
                      class="size-6 rounded-full"
                      [class.bg-lime-500]="agentStatus() === 'live'"
                      [class.animate-pulse]="agentStatus() === 'live'"
                      [class.bg-yellow-400]="agentStatus() === 'stale'"
                      [class.bg-gray-400]="agentStatus() === 'disconnected'"></div>
                  </div>
                </div>

                <div class="flex-1 grid grid-cols-1 gap-3">
                  <div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">Status</div>
                    <div
                      class="text-lg font-semibold"
                      [class.text-lime-600]="agentStatus() === 'live'"
                      [class.dark:text-lime-400]="agentStatus() === 'live'"
                      [class.text-yellow-600]="agentStatus() === 'stale'"
                      [class.dark:text-yellow-400]="agentStatus() === 'stale'"
                      [class.text-gray-500]="agentStatus() === 'disconnected'"
                      [class.dark:text-gray-400]="agentStatus() === 'disconnected'">
                      @switch (agentStatus()) {
                        @case ('live') {
                          Connected
                        }
                        @case ('stale') {
                          Stale
                        }
                        @case ('disconnected') {
                          Disconnected
                        }
                      }
                    </div>
                  </div>
                  <div class="flex gap-6">
                    <div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">Agent Name</div>
                      <div class="text-sm font-medium text-gray-900 dark:text-white flex items-center gap-1.5">
                        <fa-icon [icon]="faServer" class="text-gray-400" size="sm"></fa-icon>
                        {{ target.name }}
                      </div>
                    </div>
                    <div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">Type</div>
                      <div class="text-sm font-medium text-gray-900 dark:text-white capitalize">{{ target.type }}</div>
                    </div>
                    @if (target.currentStatus?.createdAt; as heartbeatTime) {
                      <div>
                        <div class="text-xs text-gray-500 dark:text-gray-400">Last Heartbeat</div>
                        <div
                          class="text-sm font-medium text-gray-900 dark:text-white"
                          [title]="heartbeatTime | date: 'medium'">
                          {{ heartbeatAgo() }}
                        </div>
                      </div>
                    }
                    @if (target.agentVersion?.name; as version) {
                      <div>
                        <div class="text-xs text-gray-500 dark:text-gray-400">Agent Version</div>
                        <div class="text-sm font-mono font-medium text-gray-900 dark:text-white">{{ version }}</div>
                      </div>
                    }
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Active Deployments -->
          @if (target.deployments.length > 0) {
            <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 mb-6">
              <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
                <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Active Deployments
                </h2>
              </div>
              <div class="divide-y divide-gray-200 dark:divide-gray-700">
                @for (deployment of target.deployments; track deployment.id) {
                  <div class="px-5 py-4">
                    <div class="flex items-center justify-between mb-2">
                      <div class="flex items-center gap-2">
                        <span class="text-sm font-medium text-gray-900 dark:text-white">{{
                          deployment.application.name
                        }}</span>
                        <span
                          class="font-mono text-xs bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded text-gray-700 dark:text-gray-300"
                          >{{ deployment.applicationVersionName }}</span
                        >
                      </div>
                      @if (deployment.latestStatus) {
                        <span
                          class="inline-flex items-center gap-1 text-xs font-medium"
                          [class.text-lime-600]="deployment.latestStatus.type === 'healthy'"
                          [class.dark:text-lime-400]="deployment.latestStatus.type === 'healthy'"
                          [class.text-red-600]="deployment.latestStatus.type === 'error'"
                          [class.dark:text-red-400]="deployment.latestStatus.type === 'error'"
                          [class.text-blue-600]="deployment.latestStatus.type === 'progressing'"
                          [class.dark:text-blue-400]="deployment.latestStatus.type === 'progressing'"
                          [class.text-yellow-600]="deployment.latestStatus.type === 'running'"
                          [class.dark:text-yellow-400]="deployment.latestStatus.type === 'running'">
                          <fa-icon [icon]="faCircle" size="2xs"></fa-icon>
                          {{ deployment.latestStatus.type | titlecase }}
                        </span>
                      }
                    </div>
                    @if (deployment.releaseName) {
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        Release: <span class="font-mono">{{ deployment.releaseName }}</span>
                      </div>
                    }
                  </div>
                }
              </div>
            </div>
          }
        } @else {
          <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 p-12 mb-6">
            <div class="flex flex-col items-center justify-center text-center text-gray-500 dark:text-gray-400">
              <fa-icon [icon]="faTerminal" class="text-4xl text-gray-300 dark:text-gray-600 mb-3"></fa-icon>
              <p class="text-sm font-medium">No agent connected.</p>
              <p class="text-xs mt-1">Agent status will appear here once connected.</p>
            </div>
          </div>
        }

        <!-- Infrastructure Configuration Questionnaire -->
        <!-- MOCK DATA: Entire questionnaire is for UI demonstration -->
        <div class="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
          <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Infrastructure Configuration
                </h2>
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Provide deployment configuration details for your dedicated instance
                </p>
              </div>
              <fa-icon [icon]="faClipboardList" class="text-gray-400 dark:text-gray-500"></fa-icon>
            </div>
          </div>

          <div class="px-5 py-6">
            @if (provisioningStage() === 'not_submitted') {
              <!-- Configuration Form -->
              <form [formGroup]="configForm" (ngSubmit)="submitConfig()" class="space-y-6">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <!-- Cloud Provider -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Cloud Provider <span class="text-red-500">*</span>
                    </label>
                    <select
                      formControlName="cloudProvider"
                      class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-400">
                      <option value="">Select provider</option>
                      <option value="aws">Amazon Web Services (AWS)</option>
                      <option value="azure">Microsoft Azure</option>
                      <option value="gcp">Google Cloud Platform (GCP)</option>
                    </select>
                  </div>

                  <!-- Region -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Region <span class="text-red-500">*</span>
                    </label>
                    <select
                      formControlName="region"
                      class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white">
                      <option value="">Select region</option>
                      <option value="us-east-1">US East (N. Virginia)</option>
                      <option value="us-west-2">US West (Oregon)</option>
                      <option value="eu-central-1">EU (Frankfurt)</option>
                      <option value="ap-southeast-1">Asia Pacific (Singapore)</option>
                    </select>
                  </div>

                  <!-- Networking Preference -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Networking <span class="text-red-500">*</span>
                    </label>
                    <select
                      formControlName="networking"
                      class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white">
                      <option value="">Select networking</option>
                      <option value="privatelink">PrivateLink (Private connectivity)</option>
                      <option value="public">Public endpoints</option>
                      <option value="vpn">VPN connection</option>
                    </select>
                  </div>

                  <!-- Model Provider -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Model Provider <span class="text-red-500">*</span>
                    </label>
                    <select
                      formControlName="modelProvider"
                      class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white">
                      <option value="">Select provider</option>
                      <option value="openai">OpenAI</option>
                      <option value="anthropic">Anthropic</option>
                      <option value="bedrock">AWS Bedrock</option>
                      <option value="azure-openai">Azure OpenAI</option>
                    </select>
                  </div>

                  <!-- Subdomain Preference -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Subdomain Preference
                    </label>
                    <input
                      type="text"
                      formControlName="subdomain"
                      placeholder="my-company"
                      class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-400" />
                    <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      Your instance will be available at: subdomain.your-vendor.com
                    </p>
                  </div>
                </div>

                <!-- SAML/SSO Metadata -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    SAML/SSO Metadata (XML)
                  </label>
                  <textarea
                    formControlName="samlMetadata"
                    rows="6"
                    placeholder="Paste your SAML IdP metadata XML here..."
                    class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 font-mono focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-400"></textarea>
                </div>

                <!-- IP Allowlists -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"> IP Allowlist </label>
                  <textarea
                    formControlName="ipAllowlist"
                    rows="4"
                    placeholder="10.0.0.0/8&#10;192.168.1.0/24&#10;203.0.113.0/24"
                    class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 font-mono focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder:text-gray-400"></textarea>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">One CIDR range per line</p>
                </div>

                <!-- Submit Button -->
                <div class="flex justify-end pt-4 border-t border-gray-200 dark:border-gray-700">
                  <button
                    type="submit"
                    [disabled]="!configForm.valid"
                    class="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 focus:ring-4 focus:ring-primary-300 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed dark:bg-primary-500 dark:hover:bg-primary-600 dark:focus:ring-primary-800">
                    <fa-icon [icon]="faRocket"></fa-icon>
                    Submit Configuration
                  </button>
                </div>
              </form>
            } @else {
              <!-- Provisioning Status Stepper -->
              <div class="space-y-8">
                <!-- Progress Steps -->
                <div class="flex items-center justify-between">
                  @for (stage of provisioningStages; track stage.key; let idx = $index; let isLast = $last) {
                    <div class="flex items-center" [class.flex-1]="!isLast">
                      <div class="flex flex-col items-center gap-2">
                        <div
                          class="flex items-center justify-center size-12 rounded-full border-2 transition-all"
                          [class.bg-lime-500]="isStageCompleted(stage.key)"
                          [class.border-lime-500]="isStageCompleted(stage.key)"
                          [class.text-white]="isStageCompleted(stage.key)"
                          [class.bg-primary-500]="isStageActive(stage.key)"
                          [class.border-primary-500]="isStageActive(stage.key)"
                          [class.text-white]="isStageActive(stage.key)"
                          [class.border-gray-300]="!isStageCompleted(stage.key) && !isStageActive(stage.key)"
                          [class.text-gray-400]="!isStageCompleted(stage.key) && !isStageActive(stage.key)"
                          [class.dark:border-gray-600]="!isStageCompleted(stage.key) && !isStageActive(stage.key)"
                          [class.dark:text-gray-500]="!isStageCompleted(stage.key) && !isStageActive(stage.key)">
                          @if (isStageCompleted(stage.key)) {
                            <fa-icon [icon]="faCheck" size="lg"></fa-icon>
                          } @else if (isStageActive(stage.key)) {
                            <fa-icon [icon]="stage.icon" size="lg" class="animate-pulse"></fa-icon>
                          } @else {
                            <fa-icon [icon]="stage.icon"></fa-icon>
                          }
                        </div>
                        <div class="text-center">
                          <div
                            class="text-sm font-medium"
                            [class.text-lime-600]="isStageCompleted(stage.key)"
                            [class.dark:text-lime-400]="isStageCompleted(stage.key)"
                            [class.text-primary-600]="isStageActive(stage.key)"
                            [class.dark:text-primary-400]="isStageActive(stage.key)"
                            [class.text-gray-500]="!isStageCompleted(stage.key) && !isStageActive(stage.key)"
                            [class.dark:text-gray-400]="!isStageCompleted(stage.key) && !isStageActive(stage.key)">
                            {{ stage.label }}
                          </div>
                          @if (stage.timestamp) {
                            <div class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{{ stage.timestamp }}</div>
                          }
                        </div>
                      </div>
                      @if (!isLast) {
                        <div
                          class="flex-1 h-0.5 mx-4"
                          [class.bg-lime-500]="isStageCompleted(provisioningStages[idx + 1].key)"
                          [class.bg-gray-300]="!isStageCompleted(provisioningStages[idx + 1].key)"
                          [class.dark:bg-gray-600]="!isStageCompleted(provisioningStages[idx + 1].key)"></div>
                      }
                    </div>
                  }
                </div>

                <!-- Status Message -->
                <div
                  class="rounded-lg bg-primary-50 dark:bg-primary-900/20 border border-primary-200 dark:border-primary-800 p-4">
                  <div class="flex items-start gap-3">
                    <fa-icon
                      [icon]="currentStageInfo().icon"
                      class="text-primary-600 dark:text-primary-400 mt-0.5"
                      size="lg"></fa-icon>
                    <div>
                      <h3 class="text-sm font-semibold text-primary-800 dark:text-primary-200">
                        {{ currentStageInfo().label }}
                      </h3>
                      <p class="text-sm text-primary-700 dark:text-primary-300 mt-1">
                        {{ currentStageInfo().description }}
                      </p>
                      @if (provisioningStage() !== 'ready') {
                        <p class="text-xs text-primary-600 dark:text-primary-400 mt-2">
                          Estimated time: {{ currentStageInfo().estimatedTime }}
                        </p>
                      }
                    </div>
                  </div>
                </div>

                <!-- Configuration Summary -->
                <div class="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
                  <h4 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Submitted Configuration</h4>
                  <div class="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <span class="text-gray-500 dark:text-gray-400">Cloud Provider:</span>
                      <span class="text-gray-900 dark:text-white ml-2 font-medium">{{
                        submittedConfig().cloudProvider | titlecase
                      }}</span>
                    </div>
                    <div>
                      <span class="text-gray-500 dark:text-gray-400">Region:</span>
                      <span class="text-gray-900 dark:text-white ml-2 font-mono font-medium">{{
                        submittedConfig().region
                      }}</span>
                    </div>
                    <div>
                      <span class="text-gray-500 dark:text-gray-400">Networking:</span>
                      <span class="text-gray-900 dark:text-white ml-2 font-medium">{{
                        submittedConfig().networking | titlecase
                      }}</span>
                    </div>
                    <div>
                      <span class="text-gray-500 dark:text-gray-400">Model Provider:</span>
                      <span class="text-gray-900 dark:text-white ml-2 font-medium">{{
                        submittedConfig().modelProvider | titlecase
                      }}</span>
                    </div>
                  </div>
                </div>
              </div>
            }
          </div>
        </div>
      </div>
    </div>
  `,
})
export class InfrastructurePageComponent {
  private readonly deploymentTargets = inject(DeploymentTargetsService);

  protected readonly faServer = faServer;
  protected readonly faTerminal = faTerminal;
  protected readonly faCircle = faCircle;
  protected readonly faClipboardList = faClipboardList;
  protected readonly faRocket = faRocket;
  protected readonly faCheck = faCheck;
  protected readonly faClock = faClock;
  protected readonly faSpinner = faSpinner;

  protected readonly deploymentTargetList = toSignal(this.deploymentTargets.list(), {initialValue: []});

  protected readonly firstTarget = computed(() => {
    const targets = this.deploymentTargetList();
    return targets.length > 0 ? targets[0] : undefined;
  });

  protected readonly agentStatus = computed<'live' | 'stale' | 'disconnected'>(() => {
    const target = this.firstTarget();
    if (!target?.currentStatus) return 'disconnected';
    return isStale(target.currentStatus) ? 'stale' : 'live';
  });

  protected readonly heartbeatAgo = computed(() => {
    const target = this.firstTarget();
    if (!target?.currentStatus?.createdAt) return '';
    return dayjs(target.currentStatus.createdAt).fromNow();
  });

  // MOCK DATA: Infrastructure questionnaire
  protected readonly provisioningStage = signal<ProvisioningStage>('not_submitted');
  protected readonly submittedConfig = signal<any>({});

  protected readonly configForm = new FormGroup({
    cloudProvider: new FormControl('', Validators.required),
    region: new FormControl('', Validators.required),
    networking: new FormControl('', Validators.required),
    modelProvider: new FormControl('', Validators.required),
    subdomain: new FormControl(''),
    samlMetadata: new FormControl(''),
    ipAllowlist: new FormControl(''),
  });

  protected readonly provisioningStages = [
    {key: 'submitted', label: 'Submitted', icon: faCheck, timestamp: '2 min ago'},
    {key: 'scheduled', label: 'Scheduled', icon: faClock, timestamp: '1 min ago'},
    {key: 'provisioning', label: 'Provisioning', icon: faSpinner, timestamp: 'In progress'},
    {key: 'ready', label: 'Ready', icon: faRocket, timestamp: null},
  ];

  protected submitConfig() {
    if (this.configForm.valid) {
      // MOCK DATA: Simulate form submission
      this.submittedConfig.set(this.configForm.value);
      this.provisioningStage.set('submitted');

      // MOCK: Simulate progression through stages
      setTimeout(() => this.provisioningStage.set('scheduled'), 2000);
      setTimeout(() => this.provisioningStage.set('provisioning'), 4000);
      setTimeout(() => this.provisioningStage.set('ready'), 8000);
    }
  }

  protected isStageCompleted(stage: string): boolean {
    const stages: ProvisioningStage[] = ['submitted', 'scheduled', 'provisioning', 'ready'];
    const current = stages.indexOf(this.provisioningStage());
    const target = stages.indexOf(stage as ProvisioningStage);
    return current > target;
  }

  protected isStageActive(stage: string): boolean {
    return this.provisioningStage() === stage;
  }

  protected currentStageInfo() {
    const stage = this.provisioningStage();
    const info = {
      not_submitted: {
        label: 'Not Submitted',
        description: 'Complete the form above to submit your infrastructure configuration.',
        estimatedTime: '',
        icon: faClipboardList,
      },
      submitted: {
        label: 'Configuration Submitted',
        description: 'Your infrastructure configuration has been received and is being validated.',
        estimatedTime: '2-5 minutes',
        icon: faCheck,
      },
      scheduled: {
        label: 'Provisioning Scheduled',
        description: 'Your dedicated instance has been queued for provisioning. Our team will begin setup shortly.',
        estimatedTime: '10-30 minutes',
        icon: faClock,
      },
      provisioning: {
        label: 'Provisioning In Progress',
        description:
          'Your infrastructure is being provisioned. This includes setting up cloud resources, networking, and deploying the application.',
        estimatedTime: '1-2 hours',
        icon: faSpinner,
      },
      ready: {
        label: 'Instance Ready',
        description: 'Your dedicated instance has been successfully provisioned and is ready to use!',
        estimatedTime: '',
        icon: faRocket,
      },
    };
    return info[stage] || info.not_submitted;
  }
}
