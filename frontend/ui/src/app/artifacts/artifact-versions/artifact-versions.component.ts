import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, computed, inject, resource, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faCheck,
  faEllipsisVertical,
  faExclamationTriangle,
  faFileSignature,
  faKey,
  faPen,
  faRotate,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  distinctUntilChanged,
  filter,
  firstValueFrom,
  lastValueFrom,
  map,
  NEVER,
  startWith,
  switchMap,
  tap,
} from 'rxjs';
import {RelativeDatePipe} from '../../../util/dates';
import {shortDigest} from '../../../util/digest';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {BytesPipe} from '../../../util/units';
import {ClipComponent} from '../../components/clip.component';
import {PageComponent} from '../../components/page.component';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {
  ArtifactsService,
  ArtifactUpstreamAuth,
  ArtifactWithTags,
  HasDownloads,
  TaggedArtifactVersion,
  UpstreamAuthType,
} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {CustomerOrganizationsCache} from '../../services/customer-organizations.service';
import {ImageUploadService} from '../../services/image-upload.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ArtifactsDownloadCountComponent, ArtifactsDownloadedByComponent, ArtifactsHashComponent} from '../components';

@Component({
  selector: 'app-artifact-tags',
  imports: [
    FaIconComponent,
    AsyncPipe,
    UuidComponent,
    RelativeDatePipe,
    ArtifactsDownloadCountComponent,
    ArtifactsDownloadedByComponent,
    ArtifactsHashComponent,
    ClipComponent,
    SpinnerComponent,
    BytesPipe,
    SecureImagePipe,
    RequireVendorDirective,
    OverlayModule,
    ReactiveFormsModule,
    AutotrimDirective,
    PageComponent,
  ],
  templateUrl: './artifact-versions.component.html',
  providers: [CustomerOrganizationsCache],
})
export class ArtifactVersionsComponent {
  protected readonly auth = inject(AuthService);
  private readonly artifacts = inject(ArtifactsService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly organization = inject(OrganizationService);
  private readonly contextService = inject(ContextService);
  private readonly overlay = inject(OverlayService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faBox = faBox;
  protected readonly faXmark = faXmark;
  protected readonly faTrash = faTrash;
  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faFileSignature = faFileSignature;
  protected readonly faRotate = faRotate;
  protected readonly faPen = faPen;
  protected readonly faKey = faKey;
  protected readonly faCheck = faCheck;
  protected readonly faExclamationTriangle = faExclamationTriangle;

  protected readonly syncing = signal(false);

  protected readonly showDropdown = signal(false);
  protected readonly signatureOverlayDigest = signal<string | void>(undefined);

  protected readonly upstreamURLForm = new FormGroup({
    upstreamUrl: this.fb.control('', Validators.required),
  });
  protected readonly upstreamURLFormLoading = signal(false);
  protected upstreamURLModalRef?: DialogRef;

  protected readonly upstreamAuthForm = new FormGroup({
    upstreamAuthType: this.fb.control<UpstreamAuthType | 'none'>('none', Validators.required),
    upstreamUsername: this.fb.control('', Validators.required),
    upstreamPassword: this.fb.control('', Validators.required),
  });
  protected readonly upstreamAuthFormLoading = signal(false);
  protected upstreamAuthModalRef?: DialogRef;

  constructor() {
    this.upstreamAuthForm.controls.upstreamAuthType.valueChanges
      .pipe(startWith(this.upstreamAuthForm.controls.upstreamAuthType.value), takeUntilDestroyed())
      .subscribe((authType) => {
        if (authType === 'none') {
          this.upstreamAuthForm.controls.upstreamUsername.disable();
          this.upstreamAuthForm.controls.upstreamPassword.disable();
        } else {
          this.upstreamAuthForm.controls.upstreamUsername.enable();
          this.upstreamAuthForm.controls.upstreamPassword.enable();
        }
      });
  }

  protected readonly artifact = toSignal(
    this.route.params.pipe(
      map((params) => params['id']?.trim()),
      distinctUntilChanged(),
      switchMap((id) => this.artifacts.getByIdAndCache(id)),
      map((artifact) => {
        if (artifact) {
          return {
            ...artifact,
            versions: (artifact.versions ?? []).map((v) => ({
              ...v,
              ...this.calcVersionDownloads(v),
            })),
          };
        }
        return undefined;
      })
    )
  );

  protected readonly filteredVersions = computed(() => {
    const versions = this.artifact()?.versions;
    if (!versions) {
      return [];
    }

    const signatureVersionsByTag = new Map<string, (typeof versions)[number]>();
    for (const version of versions) {
      if (version.inferredType === 'signature') {
        for (const tag of version.tags) {
          signatureVersionsByTag.set(tag.name, version);
        }
      }
    }

    return versions
      .filter((version) => version.inferredType !== 'signature')
      .map((version) => ({
        ...version,
        signatureVersion: signatureVersionsByTag.get(`sha256-${version.digest.substring(7)}`),
      }));
  });

  protected readonly org = resource({
    loader: () => firstValueFrom(this.organization.get()),
  });
  private readonly registryHost = resource({
    loader: () => firstValueFrom(this.contextService.getRegistryHost()),
  });

  protected readonly artifactUsage = computed(() => {
    const artifact = this.artifact();
    const version = artifact?.versions?.find((it) => it.inferredType !== 'signature' && it.tags.length > 0);
    if (!artifact || !version) {
      return undefined;
    }
    const org = this.org.value();
    const registryHost = this.registryHost.value();
    const url = `${registryHost ?? 'REGISTRY_DOMAIN'}/${org?.slug ?? 'ORG_SLUG'}/${artifact.name}`;
    switch (version.inferredType) {
      case 'helm-chart':
        return `helm install <release-name> oci://${url} --version ${version.tags[0].name}`;
      case 'container-image':
        return `docker pull ${url}:${version.tags[0].name}`;
      default:
        return `oras pull ${url}:${version.tags[0].name}`;
    }
  });

  protected calcVersionDownloads(version: TaggedArtifactVersion): HasDownloads {
    const downloadsTotal = version.tags.reduce(
      (sum, tag) => sum + (tag.downloads.downloadsTotal ?? 0),
      version.downloadsTotal ?? 0
    );
    const downloadedByUsers = Array.from(
      new Set<string>([
        ...(version.downloadedByUsers ?? []),
        ...version.tags.flatMap((t) => t.downloads.downloadedByUsers ?? []),
      ])
    );
    const downloadedByCustomerOrganizations = Array.from(
      new Set<string>([
        ...(version.downloadedByCustomerOrganizations ?? []),
        ...version.tags.flatMap((t) => t.downloads.downloadedByCustomerOrganizations ?? []),
      ])
    );
    return {
      downloadsTotal,
      downloadedByUsers,
      downloadedByUsersCount: downloadedByUsers.length,
      downloadedByCustomerOrganizations,
      downloadedByCustomerOrganizationsCount: downloadedByCustomerOrganizations.length,
    };
  }

  openUpstreamURLModal(artifact: ArtifactWithTags, templateRef: TemplateRef<unknown>) {
    this.upstreamURLForm.reset({upstreamUrl: artifact.upstreamUrl ?? ''});
    this.upstreamURLModalRef?.close();
    this.upstreamURLModalRef = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  async saveUpstreamURL(artifact: ArtifactWithTags) {
    this.upstreamURLForm.markAllAsTouched();
    if (this.upstreamURLForm.invalid) {
      return;
    }
    this.upstreamURLFormLoading.set(true);

    try {
      const {upstreamUrl} = this.upstreamURLForm.value;
      await lastValueFrom(this.artifacts.patchUpstreamURL(artifact.id, upstreamUrl || null));
      this.toast.success('Upstream URL updated');
      this.upstreamURLModalRef?.close();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) this.toast.error(msg);
    } finally {
      this.upstreamURLFormLoading.set(false);
    }
  }

  openUpstreamAuthModal(artifact: ArtifactWithTags, templateRef: TemplateRef<unknown>) {
    this.upstreamAuthForm.reset({
      upstreamAuthType: artifact.upstreamAuthType ?? 'none',
      upstreamUsername: '',
      upstreamPassword: '',
    });
    this.upstreamAuthModalRef?.close();
    this.upstreamAuthModalRef = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  async saveUpstreamAuth(artifact: ArtifactWithTags) {
    this.upstreamAuthForm.markAllAsTouched();
    if (this.upstreamAuthForm.invalid) {
      return;
    }
    this.upstreamAuthFormLoading.set(true);

    try {
      const {upstreamAuthType, upstreamUsername, upstreamPassword} = this.upstreamAuthForm.value;
      let auth: ArtifactUpstreamAuth | null = null;
      if (upstreamAuthType && upstreamAuthType !== 'none') {
        auth = {
          type: upstreamAuthType,
          username: upstreamUsername || undefined,
          password: upstreamPassword || undefined,
        };
      }
      await lastValueFrom(this.artifacts.patchUpstreamAuth(artifact.id, auth));
      this.toast.success('Upstream authentication updated');
      this.upstreamAuthModalRef?.close();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) this.toast.error(msg);
    } finally {
      this.upstreamAuthFormLoading.set(false);
    }
  }

  public async uploadImage(data: ArtifactWithTags) {
    const fileId = await firstValueFrom(this.imageUploadService.showDialog({imageUrl: data.imageUrl}));
    if (!fileId || data.imageUrl?.includes(fileId)) {
      return;
    }
    await firstValueFrom(this.artifacts.patchImage(data.id!, fileId));
  }

  public deleteArtifact(artifact: ArtifactWithTags): void {
    this.overlay
      .confirm(
        `This will permanently delete ${artifact.name} and all its versions. Users will no longer be able to download this artifact. Are you sure?`
      )
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.artifacts.deleteArtifact(artifact.id)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return NEVER;
        }),
        tap(() => this.toast.success('Artifact deleted successfully')),
        switchMap(() => this.router.navigate(['/artifacts']))
      )
      .subscribe();
  }

  public deleteArtifactTag(artifact: ArtifactWithTags, version: TaggedArtifactVersion, tagName: string): void {
    const digest = shortDigest(version.digest);
    this.overlay
      .confirm({
        message: {
          message: `This will untag "${tagName}" from ${artifact.name}. Are you sure?`,
          alert:
            version.tags.length > 1
              ? {
                  type: 'info',
                  message: `The digest ${digest} stays available under its other tags.`,
                }
              : {
                  type: 'warning',
                  message: `Since this is its only tag, the digest ${digest} may be deleted as well, unless  it is still referenced by a multi-arch index.`,
                },
        },
      })
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.artifacts.deleteArtifactTag(artifact, tagName)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return NEVER;
        }),
        tap(() => this.toast.success(`Tag "${tagName}" removed successfully`))
      )
      .subscribe();
  }

  public syncArtifact(artifact: ArtifactWithTags): void {
    if (this.syncing()) return;
    this.syncing.set(true);
    this.artifacts
      .syncArtifact(artifact.id)
      .pipe(
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return NEVER;
        }),
        tap(() => this.toast.success('Sync completed'))
      )
      .subscribe({complete: () => this.syncing.set(false)});
  }

  protected showSignatureOverlay(version: TaggedArtifactVersion) {
    this.signatureOverlayDigest.set(version.digest);
  }

  protected hideSignatureOverlay() {
    this.signatureOverlayDigest.set(undefined);
  }
}
