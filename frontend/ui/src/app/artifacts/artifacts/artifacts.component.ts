import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faLightbulb,
  faMagnifyingGlass,
  faPlus,
  faTrash,
  faUserCircle,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, lastValueFrom, map, startWith} from 'rxjs';
import {fromPromise} from 'rxjs/internal/observable/innerFrom';
import {getRemoteEnvironment} from '../../../env/remote';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireCustomerDirective, RequireVendorDirective} from '../../directives/required-role.directive';
import {ArtifactsService, ArtifactUpstreamAuth, UpstreamAuthType} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsCache} from '../../services/customer-organizations.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {
  ArtifactsDownloadCountComponent,
  ArtifactsDownloadedByComponent,
  ArtifactTypeBadgesComponent,
} from '../components';

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
    ArtifactTypeBadgesComponent,
    AutotrimDirective,
    RequireVendorDirective,
    RequireCustomerDirective,
    SecureImagePipe,
    OverlayModule,
    SpinnerComponent,
  ],
  templateUrl: './artifacts.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  providers: [CustomerOrganizationsCache],
})
export class ArtifactsComponent {
  private readonly artifactsService = inject(ArtifactsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBox = faBox;
  protected readonly faTrash = faTrash;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faUserCircle = faUserCircle;

  protected readonly filterForm = new FormGroup({
    search: this.fb.control(''),
  });

  protected readonly createForm = new FormGroup({
    name: this.fb.control('', Validators.required),
    upstreamUrl: this.fb.control('', Validators.required),
    upstreamAuthType: this.fb.control<UpstreamAuthType | 'none'>('none', Validators.required),
    upstreamUsername: this.fb.control('', Validators.required),
    upstreamPassword: this.fb.control('', Validators.required),
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
  private readonly organizationBrandingService = inject(OrganizationBrandingService);
  protected readonly registrySlug$ = this.organizationService.get().pipe(map((org) => org.slug));
  protected readonly registryHost$ = combineLatest([
    fromPromise(getRemoteEnvironment()),
    this.organizationBrandingService.registryDomain(),
  ]).pipe(map(([env, registryDomain]) => registryDomain ?? env.registryHost));

  protected readonly auth = inject(AuthService);
  protected readonly hasNoSubscription = this.organizationService.hasNoSubscription;

  constructor() {
    this.createForm.controls.upstreamAuthType.valueChanges
      .pipe(startWith(this.createForm.controls.upstreamAuthType.value), takeUntilDestroyed())
      .subscribe((t) => {
        if (t === 'none') {
          this.createForm.controls.upstreamUsername.disable();
          this.createForm.controls.upstreamPassword.disable();
        } else {
          this.createForm.controls.upstreamUsername.enable();
          this.createForm.controls.upstreamPassword.enable();
        }
      });
  }

  openCreateModal(templateRef: TemplateRef<unknown>) {
    this.hideCreateModal();
    this.createModalRef = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  hideCreateModal() {
    this.createModalRef?.close();
    this.createForm.reset({upstreamAuthType: 'none'});
  }

  async createArtifact() {
    this.createForm.markAllAsTouched();
    if (!this.createForm.valid) {
      return;
    }
    this.createFormLoading = true;
    try {
      const {name, upstreamUrl, upstreamAuthType, upstreamUsername, upstreamPassword} = this.createForm.value;
      let upstreamAuth: ArtifactUpstreamAuth | undefined;
      if (upstreamAuthType && upstreamAuthType !== 'none') {
        upstreamAuth = {
          type: upstreamAuthType,
          username: upstreamUsername || undefined,
          password: upstreamPassword || undefined,
        };
      }
      const created = await lastValueFrom(
        this.artifactsService.createArtifact(name!, upstreamUrl || undefined, upstreamAuth)
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
