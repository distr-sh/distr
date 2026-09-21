import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe, NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, input, OnDestroy, TemplateRef} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {Application, DeploymentType} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faBoxArchive, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {lastValueFrom, Observable, Subject, takeUntil} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {filteredByFormControl} from '../../util/filter';
import {automaticUpdatesBadgeClass, SelectableVersioningStrategy} from '../../util/versions';
import {SearchBarComponent} from '../components/search-bar.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {
  PermissionsDirective,
  RequireRoleDirective,
  RequireVendorDirective,
} from '../directives/required-role.directive';
import {ApplicationsService} from '../services/applications.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {ApplicationLogoComponent} from './components';

@Component({
  selector: 'app-applications',
  imports: [
    AsyncPipe,
    DatePipe,
    ReactiveFormsModule,
    FaIconComponent,
    OverlayModule,
    AutotrimDirective,
    RequireVendorDirective,
    RequireRoleDirective,
    PermissionsDirective,
    RouterLink,
    ApplicationLogoComponent,
    SearchBarComponent,
    NgClass,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './applications.component.html',
})
export class ApplicationsComponent implements OnDestroy {
  readonly fullVersion = input<boolean>(false);
  private readonly router = inject(Router);
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faXmark = faXmark;
  protected readonly faBoxArchive = faBoxArchive;
  protected readonly faTrash = faTrash;

  private readonly destroyed$ = new Subject<void>();
  private readonly applications = inject(ApplicationsService);
  filterForm = new FormGroup({
    search: new FormControl(''),
  });
  applications$: Observable<Application[]> = filteredByFormControl(
    this.applications.list(),
    this.filterForm.controls.search,
    (it: Application, search: string) => !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
  ).pipe(takeUntil(this.destroyed$));
  createApplicationForm = new FormGroup({
    name: new FormControl('', Validators.required),
    type: new FormControl<DeploymentType>('docker', Validators.required),
    versioningStrategy: new FormControl<SelectableVersioningStrategy>('semver', Validators.required),
    allowAutomaticUpdates: new FormControl(false, {nonNullable: true}),
  });
  createApplicationFormLoading = false;
  protected readonly isAutoUpdatesEnabled = inject(FeatureFlagService).isAutoUpdatesEnabled;
  protected readonly automaticUpdatesBadgeClass = automaticUpdatesBadgeClass(true);

  private applicationCreateModalRef?: DialogRef;

  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  openCreateModal(templateRef: TemplateRef<unknown>) {
    this.hideCreateModal();
    this.applicationCreateModalRef = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  hideCreateModal() {
    this.applicationCreateModalRef?.close();
    this.createApplicationForm.reset({type: 'docker', versioningStrategy: 'semver', allowAutomaticUpdates: false});
  }

  async createApplication() {
    this.createApplicationForm.markAllAsTouched();
    if (this.createApplicationForm.valid) {
      this.createApplicationFormLoading = true;
      try {
        const formVal = this.createApplicationForm.value;
        const created = await lastValueFrom(
          this.applications.create({
            name: formVal.name!,
            type: formVal.type!,
            versioningStrategy: formVal.versioningStrategy!,
            allowAutomaticUpdates: formVal.allowAutomaticUpdates!,
          })
        );
        this.toast.success(`${formVal.name} created successfully`);
        this.hideCreateModal();
        await this.router.navigate(['/applications', created.id]);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.createApplicationFormLoading = false;
      }
    }
  }

  protected readonly faBox = faBox;
}
