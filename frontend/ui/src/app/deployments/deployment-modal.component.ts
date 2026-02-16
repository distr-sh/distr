import {Component, effect, inject, input, output, signal} from '@angular/core';
import {FormControl, FormsModule, ReactiveFormsModule, Validators} from '@angular/forms';
import {DeploymentTarget, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faShip} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {ToastService} from '../services/toast.service';
import {
  DeploymentFormComponent,
  DeploymentFormValue,
  mapToDeploymentRequest,
} from './deployment-form/deployment-form.component';

@Component({
  selector: 'app-deployment-modal',
  template: `<div class="z-50 w-256 max-w-full max-h-full overflow-x-hidden overflow-y-auto">
    <div class="relative w-full max-h-full">
      <!-- Modal content -->
      <div class="relative bg-white rounded-lg shadow-sm dark:bg-gray-700">
        <!-- Modal header -->
        <div
          class="flex items-center justify-between p-4 md:p-5 border-b border-gray-200 rounded-t dark:border-gray-600">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
            @if (deployment()?.id) {
              Update Deployment
            } @else {
              Deploy new application version
            }
          </h3>
          <button
            type="button"
            class="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm w-8 h-8 ms-auto inline-flex justify-center items-center dark:hover:bg-gray-600 dark:hover:text-white"
            (click)="closed.emit()">
            <svg class="w-3 h-3" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 14 14">
              <path
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="m1 1 6 6m0 0 6 6M7 7l6-6M7 7l-6 6" />
            </svg>
            <span class="sr-only">Close modal</span>
          </button>
        </div>
        <!-- Modal body -->
        <form class="p-4 md:p-5" (ngSubmit)="saveDeployment()">
          @if (deploymentTarget().customerOrganization !== undefined && auth.isVendor()) {
            <div
              class="flex items-center p-4 mb-4 text-yellow-800 rounded-lg bg-yellow-50 dark:bg-gray-800 dark:text-yellow-300"
              role="alert">
              <fa-icon [icon]="faCircleExclamation" />
              <span class="sr-only">Info</span>
              <div class="ms-3 text-sm font-medium">
                Warning: You are about to overwrite a customer-managed deployment. Ensure this is done in coordination
                with the customer.
              </div>
            </div>
          }
          <app-deployment-form
            [formControl]="deployForm"
            [deploymentType]="deploymentTarget().type"
            [customerOrganizationId]="deploymentTarget().customerOrganization?.id"
            [deploymentTargetName]="deploymentTarget().name"></app-deployment-form>
          <button
            type="submit"
            [disabled]="loading()"
            class="text-white inline-flex items-center bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800">
            <fa-icon [icon]="faShip" class="h-5 w-5 mr-2 -ml-0.5 dark:text-gray-400"></fa-icon>
            Deploy
          </button>
        </form>
      </div>
    </div>
  </div>`,
  imports: [DeploymentFormComponent, FaIconComponent, FormsModule, ReactiveFormsModule],
})
export class DeploymentModalComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deployment = input<DeploymentWithLatestRevision>();
  public readonly versionId = input<string>();
  public readonly closed = output();

  protected readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);

  protected readonly deployForm = new FormControl<DeploymentFormValue | undefined>(undefined, Validators.required);
  protected readonly loading = signal(false);

  protected readonly faShip = faShip;
  protected readonly faCircleExclamation = faCircleExclamation;

  constructor() {
    effect(() => {
      const deployment = this.deployment();
      this.deployForm.reset({
        deploymentId: deployment?.id,
        applicationId: deployment?.applicationId,
        applicationVersionId: this.versionId() ?? deployment?.applicationVersionId,
        applicationLicenseId: deployment?.applicationLicenseId,
        releaseName: deployment?.releaseName,
        valuesYaml: deployment?.valuesYaml ? atob(deployment.valuesYaml) : undefined,
        swarmMode: deployment?.dockerType === 'swarm',
        envFileData: deployment?.envFileData ? atob(deployment.envFileData) : undefined,
        helmOptions: deployment?.helmOptions,
      });
    });
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
