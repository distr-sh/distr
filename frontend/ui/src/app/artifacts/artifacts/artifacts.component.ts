import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, inject, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faLightbulb,
  faMagnifyingGlass,
  faPlus,
  faSpinner,
  faTrash,
  faUserCircle,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, lastValueFrom, map, startWith} from 'rxjs';
import {fromPromise} from 'rxjs/internal/observable/innerFrom';
import {getRemoteEnvironment} from '../../../env/remote';
import {SecureImagePipe} from '../../../util/secureImage';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireCustomerDirective, RequireVendorDirective} from '../../directives/required-role.directive';
import {ArtifactsService, ArtifactWithTags} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsCache} from '../../services/customer-organizations.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ArtifactsDownloadCountComponent, ArtifactsDownloadedByComponent} from '../components';
import {getFormDisplayedError} from '../../../util/errors';

@Component({
  selector: 'app-artifacts',
  imports: [
    ReactiveFormsModule,
    AsyncPipe,
    FaIconComponent,
    UuidComponent,
    RouterLink,
    ArtifactsDownloadCountComponent,
    ArtifactsDownloadedByComponent,
    AutotrimDirective,
    RequireVendorDirective,
    RequireCustomerDirective,
    SecureImagePipe,
    OverlayModule,
  ],
  templateUrl: './artifacts.component.html',
  providers: [CustomerOrganizationsCache],
})
export class ArtifactsComponent {
  private readonly artifactsService = inject(ArtifactsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBox = faBox;
  protected readonly faTrash = faTrash;
  protected readonly faSpinner = faSpinner;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faUserCircle = faUserCircle;

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  protected readonly createForm = new FormGroup({
    name: new FormControl('', Validators.required),
    upstreamUrl: new FormControl(''),
  });
  protected createFormLoading = false;
  private createModalRef?: DialogRef;

  private readonly artifacts$ = this.artifactsService.list().pipe(takeUntilDestroyed());
  protected readonly hasNoArtifact = toSignal(this.artifacts$.pipe(map((artifacts) => artifacts.length === 0)));

  protected readonly filteredArtifacts = toSignal(
    combineLatest([this.artifacts$, this.filterForm.valueChanges.pipe(startWith(this.filterForm.value))]).pipe(
      map(([artifacts, formValue]) =>
        artifacts.filter((it) => !formValue.search || it.name.toLowerCase().includes(formValue.search.toLowerCase()))
      )
    )
  );

  private readonly organizationService = inject(OrganizationService);
  protected readonly registrySlug$ = this.organizationService.get().pipe(map((org) => org.slug));
  protected readonly registryHost$ = combineLatest([
    fromPromise(getRemoteEnvironment()),
    this.organizationService.get(),
  ]).pipe(map(([env, org]) => org.registryDomain ?? env.registryHost));

  protected readonly auth = inject(AuthService);
  protected readonly hasNoSubscription = this.organizationService.hasNoSubscription;

  openCreateModal(templateRef: TemplateRef<unknown>) {
    this.hideCreateModal();
    this.createModalRef = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  hideCreateModal() {
    this.createModalRef?.close();
    this.createForm.reset();
  }

  async createArtifact() {
    this.createForm.markAllAsTouched();
    if (!this.createForm.valid) {
      return;
    }
    this.createFormLoading = true;
    try {
      const {name, upstreamUrl} = this.createForm.value;
      const created = await lastValueFrom(
        this.artifactsService.createArtifact(name!, upstreamUrl || undefined)
      );
      this.toast.success(`${name} created successfully`);
      this.hideCreateModal();
      await this.router.navigate(['/artifacts', created.id]);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.createFormLoading = false;
    }
  }
}
