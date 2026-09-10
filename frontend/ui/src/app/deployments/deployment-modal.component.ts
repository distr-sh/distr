import {ChangeDetectionStrategy, Component, computed, effect, inject, input, output, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {DeploymentTarget, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faShip, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {fromBase64} from '../../util/encoding';
import {getFormDisplayedError} from '../../util/errors';
import {SpinnerComponent} from '../components/spinner/spinner.component';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {ToastService} from '../services/toast.service';
import {
  DeploymentFormComponent,
  DeploymentFormValue,
  mapToDeploymentRequest,
} from './deployment-form/deployment-form.component';

@Component({
  selector: 'app-deployment-modal',
  templateUrl: './deployment-modal.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [DeploymentFormComponent, FaIconComponent, ReactiveFormsModule, SpinnerComponent],
})
export class DeploymentModalComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deployment = input<DeploymentWithLatestRevision>();
  public readonly versionId = input<string>();
  public readonly closed = output();

  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly applications = inject(ApplicationsService);
  private readonly applicationEntitlements = inject(ApplicationEntitlementsService);
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly dataLoaded = signal(false);

  protected readonly customerManagedWarningVisible = computed(
    () => this.deploymentTarget().customerOrganization !== undefined && this.auth.isVendor()
  );

  protected readonly deployForm = new FormControl<DeploymentFormValue | undefined>(undefined, Validators.required);
  /**
   * This is required because ngSubmit only works on forms with a form group attached
   */
  protected readonly deployFormWrapper = new FormGroup({deployment: this.deployForm});
  protected readonly loading = signal(false);

  protected readonly faShip = faShip;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faXmark = faXmark;

  constructor() {
    this.refreshCachedLists();
    effect(() => {
      const deployment = this.deployment();
      this.deployForm.reset({
        deploymentId: deployment?.id,
        applicationId: deployment?.applicationId,
        applicationVersionId: this.versionId() ?? deployment?.applicationVersionId,
        applicationEntitlementId: deployment?.applicationEntitlementId,
        releaseName: deployment?.releaseName,
        valuesYaml: deployment?.valuesYaml ? fromBase64(deployment.valuesYaml) : undefined,
        swarmMode: deployment?.dockerType === 'swarm',
        envFileData: deployment?.envFileData ? fromBase64(deployment.envFileData) : undefined,
        helmOptions: deployment?.helmOptions,
      });
    });
  }

  private async refreshCachedLists() {
    try {
      const licensingEnabled = await firstValueFrom(this.featureFlags.isLicensingEnabled$);
      await Promise.all([
        this.applications.refresh(),
        ...(licensingEnabled ? [this.applicationEntitlements.refresh()] : []),
      ]);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      // Falling back to the cached list still lets the user deploy, which a modal stuck on a spinner would not.
      this.dataLoaded.set(true);
    }
  }

  protected async saveDeployment() {
    this.deployForm.markAllAsTouched();
    if (this.deployForm.valid) {
      this.loading.set(true);
      const deployment = mapToDeploymentRequest(this.deployForm.value!, this.deploymentTarget().id!);
      try {
        await firstValueFrom(this.deploymentTargets.deploy(deployment));
        this.toast.success('Deployment saved successfully');
        this.closed.emit();
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.loading.set(false);
      }
    }
  }
}
