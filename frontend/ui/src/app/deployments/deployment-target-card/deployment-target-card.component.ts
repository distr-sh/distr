import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {DatePipe, NgOptimizedImage} from '@angular/common';
import {Component, computed, inject, resource, signal, TemplateRef, viewChild, WritableSignal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ReactiveFormsModule} from '@angular/forms';
import {ApplicationVersion, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBinoculars,
  faEllipsisVertical,
  faHeartPulse,
  faPlus,
  faRotate,
  faTrash,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons';
import {EMPTY, filter, firstValueFrom, switchMap} from 'rxjs';
import {SemVer} from 'semver';
import {maxBy} from '../../../util/arrays';
import {isArchived} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {StatusDotComponent} from '../../components/status-dot';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AgentVersionService} from '../../services/agent-version.service';
import {ApplicationsService} from '../../services/applications.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicensesService} from '../../services/licenses.service';
import {DeploymentModalComponent} from '../deployment-modal.component';
import {DeploymentStatusModalComponent} from '../deployment-status-modal/deployment-status-modal.component';
import {DeploymentTargetStatusModalComponent} from '../deployment-target-status-modal/deployment-target-status-modal.component';
import {DeploymentAppNameComponent} from './deployment-app-name.component';
import {DeploymentStatusTextComponent} from './deployment-status-text.component';
import {DeploymentTargetCardBaseComponent} from './deployment-target-card-base.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-card',
  templateUrl: './deployment-target-card.component.html',
  imports: [
    NgOptimizedImage,
    StatusDotComponent,
    UuidComponent,
    DatePipe,
    FaIconComponent,
    OverlayModule,
    ConnectInstructionsComponent,
    ReactiveFormsModule,
    DeploymentModalComponent,
    DeploymentTargetMetricsComponent,
    DeploymentStatusModalComponent,
    TextFieldModule,
    DeploymentTargetStatusModalComponent,
    DeploymentAppNameComponent,
    DeploymentStatusTextComponent,
    AutotrimDirective,
  ],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
})
export class DeploymentTargetCardComponent extends DeploymentTargetCardBaseComponent {
  private readonly agentVersionsSvc = inject(AgentVersionService);
  private readonly licensesService = inject(LicensesService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly deleteDeploymentProgressModal = viewChild.required<TemplateRef<unknown>>(
    'deleteDeploymentProgressModal'
  );

  protected readonly faBinoculars = faBinoculars;
  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faHeartPulse = faHeartPulse;
  protected readonly faPlus = faPlus;
  protected readonly faRotate = faRotate;
  protected readonly faTrash = faTrash;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  protected readonly showDeploymentTargetDropdown = signal(false);
  protected readonly showDeploymentDropdownForId = signal<string | undefined>(undefined);
  protected readonly metricsOpened = signal(false);

  protected readonly agentVersions = resource({
    loader: () => firstValueFrom(this.agentVersionsSvc.list()),
  });

  protected readonly isUndeploySupported = this.isAgentVersionAtLeast('1.3.0');
  protected readonly isMultiDeploymentSupported = this.isAgentVersionAtLeast('1.6.0');
  protected readonly isLoggingSupported = this.isAgentVersionAtLeast('1.9.0');
  protected readonly isForceRestartSupported = this.isAgentVersionAtLeast('1.12.0');

  protected readonly agentUpdateAvailable = computed(() => {
    const agentVersions = this.agentVersions.value() ?? [];
    return (
      agentVersions.length > 0 &&
      this.deploymentTarget().agentVersion?.id !== agentVersions[agentVersions.length - 1].id
    );
  });

  private readonly licenses = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(switchMap((enabled) => (enabled ? this.licensesService.list() : EMPTY))),
    {initialValue: []}
  );

  private readonly applications = toSignal(this.applicationsService.list(), {initialValue: []});

  protected readonly deploymentIdsWithUpdate = computed(() => {
    const deploymentTarget = this.deploymentTarget();
    const applications = this.applications();
    const licenses = this.licenses();

    return new Set(
      deploymentTarget.deployments
        .map((deployment) => {
          const applicationVersions =
            (deployment.applicationLicenseId
              ? licenses.find((license) => license.id === deployment.applicationLicenseId)?.application?.versions
              : undefined) ?? applications.find((app) => app.id === deployment.applicationId)?.versions;

          const maxVersion = this.findMaxVersion(applicationVersions?.filter((version) => !isArchived(version)) ?? []);

          return maxVersion && deployment.applicationVersionId !== maxVersion.id ? deployment.id : undefined;
        })
        .filter((id) => id !== undefined)
    );
  });

  protected readonly agentUpdatePending = computed(
    () =>
      this.deploymentTarget().currentStatus !== undefined &&
      this.deploymentTarget().agentVersion?.id !== this.deploymentTarget().reportedAgentVersionId
  );

  protected openStatusModal(deployment: DeploymentWithLatestRevision) {
    if (deployment?.id) {
      this.selectedDeployment.set(deployment);
      this.showModal(this.deploymentStatusModal());
    }
  }

  protected openDeploymentTargetStatusModal() {
    this.showModal(this.deploymentTargetStatusModal());
  }

  protected setLogsEnabled(deployment: DeploymentWithLatestRevision, logsEnabled: boolean) {
    if (deployment.id) {
      this.deploymentTargets.patchDeployment(deployment.id, {logsEnabled}).subscribe({
        next: () => this.toast.success('Deployment has been updated.'),
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
    }
  }

  protected forceRestart(deployment: DeploymentWithLatestRevision) {
    if (deployment.id) {
      this.overlay
        .confirm({
          message: {
            message: 'Are you sure you want to force restart this deployment?',
            alert: {
              type: 'warning',
              message: 'Depending on the deployment, this may cause downtime.',
            },
          },
        })
        .pipe(
          filter((result) => result === true),
          switchMap(() =>
            this.deploymentTargets.deploy({
              deploymentId: deployment.id,
              deploymentTargetId: deployment.deploymentTargetId,
              applicationVersionId: deployment.applicationVersionId,
              applicationLicenseId: deployment.applicationLicenseId,
              releaseName: deployment.releaseName,
              dockerType: deployment.dockerType,
              valuesYaml: deployment.valuesYaml,
              envFileData: deployment.envFileData,
              forceRestart: true,
            })
          )
        )
        .subscribe({
          next: () => this.toast.success('Deployment has been restarted.'),
          error: (e) => {
            const msg = getFormDisplayedError(e);
            if (msg) {
              this.toast.error(msg);
            }
          },
        });
    }
  }

  protected deleteDeploymentTarget() {
    const dt = this.deploymentTarget();
    const alert =
      dt.customerOrganization !== undefined && this.auth.isVendor()
        ? ({type: 'warning', message: this.customerManagedWarning} as const)
        : undefined;
    this.overlay
      .confirm({
        customTemplate: this.deleteConfirmModal(),
        requiredConfirmInputText: 'DELETE',
        message: {
          alert,
          message: '',
        },
      })
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.deploymentTargets.delete(dt))
      )
      .subscribe({
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  protected async deleteDeployment(d: DeploymentWithLatestRevision, confirmTemplate: TemplateRef<any>) {
    const dt = this.deploymentTarget();
    const alert =
      dt.customerOrganization !== undefined && this.auth.isVendor()
        ? ({type: 'warning', message: this.customerManagedWarning} as const)
        : undefined;
    if (d.id) {
      if (
        await firstValueFrom(
          this.overlay.confirm({
            customTemplate: confirmTemplate,
            requiredConfirmInputText: 'UNDEPLOY',
            message: {
              alert,
              message: '',
            },
          })
        )
      ) {
        const modalRef = this.overlay.showModal(this.deleteDeploymentProgressModal(), {
          positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
          backdropStyleOnly: true,
        });

        try {
          await firstValueFrom(this.deploymentTargets.undeploy(d.id));
        } catch (e) {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        } finally {
          modalRef?.dismiss();
        }
      }
    }
  }

  public async updateDeploymentTargetAgent(): Promise<void> {
    try {
      const dt = this.deploymentTarget();
      const agentVersions = this.agentVersions.value();
      if (agentVersions?.length) {
        const targetVersion = agentVersions[agentVersions.length - 1];
        if (
          await firstValueFrom(
            this.overlay.confirm(`Update ${dt.name} agent from ${dt.agentVersion?.name} to ${targetVersion.name}?`)
          )
        ) {
          dt.agentVersion = targetVersion;
          await firstValueFrom(this.deploymentTargets.update(dt));
        }
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected toggle(signal: WritableSignal<boolean>) {
    signal.update((val) => !val);
  }

  private isAgentVersionAtLeast(version: string, allowSnapshot = true) {
    return computed(() => {
      if (!this.deploymentTarget().reportedAgentVersionId) {
        console.warn('reported agent version id is empty');
        return true;
      }
      const reported = this.agentVersions
        .value()
        ?.find((it) => it.id === this.deploymentTarget().reportedAgentVersionId);
      if (!reported) {
        console.warn('agent version with id not found', this.deploymentTarget().reportedAgentVersionId);
        return false;
      }
      try {
        return (allowSnapshot && reported.name === 'snapshot') || new SemVer(reported.name).compare(version) >= 0;
      } catch (e) {
        console.warn(e);
        return allowSnapshot && reported.name === 'snapshot';
      }
    });
  }

  private findMaxVersion(versions: ApplicationVersion[]): ApplicationVersion | undefined {
    try {
      return maxBy(
        versions,
        (version) => new SemVer(version.name),
        (a, b) => a.compare(b) > 0
      );
    } catch (e) {
      console.warn('semver compare failed, falling back to creation date', e);
      return maxBy(versions, (version) => new Date(version.createdAt!));
    }
  }
}
