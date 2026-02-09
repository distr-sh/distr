import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, inject, resource, signal} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faEllipsisVertical, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, distinctUntilChanged, filter, firstValueFrom, map, NEVER, switchMap, tap} from 'rxjs';
import {getRemoteEnvironment} from '../../../env/remote';
import {RelativeDatePipe} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {BytesPipe} from '../../../util/units';
import {dropdownAnimation} from '../../animations/dropdown';
import {ClipComponent} from '../../components/clip.component';
import {UuidComponent} from '../../components/uuid';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {
  ArtifactsService,
  ArtifactWithTags,
  HasDownloads,
  TaggedArtifactVersion,
} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {ImageUploadService} from '../../services/image-upload.service';
import {OrganizationService} from '../../services/organization.service';
import {OverlayService} from '../../services/overlay.service';
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
    BytesPipe,
    SecureImagePipe,
    RequireVendorDirective,
    OverlayModule,
  ],
  animations: [dropdownAnimation],
  templateUrl: './artifact-versions.component.html',
})
export class ArtifactVersionsComponent {
  protected readonly auth = inject(AuthService);
  private readonly artifacts = inject(ArtifactsService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly organization = inject(OrganizationService);
  private readonly overlay = inject(OverlayService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly toast = inject(ToastService);

  protected readonly faBox = faBox;
  protected readonly faXmark = faXmark;
  protected readonly faTrash = faTrash;
  protected readonly faEllipsisVertical = faEllipsisVertical;

  protected readonly showDropdown = signal(false);

  protected readonly artifact$ = this.route.params.pipe(
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
  );

  protected readonly org = resource({
    loader: () => firstValueFrom(this.organization.get()),
  });
  private readonly remoteEnv = resource({
    loader: () => getRemoteEnvironment(),
  });

  public getArtifactUsage(artifact: ArtifactWithTags): string | undefined {
    if (!artifact.versions?.length) {
      // this should not actually happen
      return undefined;
    }
    const org = this.org.value();
    const env = this.remoteEnv.value();
    let url = `${org?.registryDomain ?? env?.registryHost ?? 'REGISTRY_DOMAIN'}/${org?.slug ?? 'ORG_SLUG'}/${artifact.name}`;
    const version = artifact.versions.find((it) => it.tags && it.tags.length > 0);
    if (!version) return;
    switch (version.inferredType) {
      case 'helm-chart':
        return `helm install <release-name> oci://${url} --version ${version.tags[0].name}`;
      case 'container-image':
        return `docker pull ${url}:${version.tags[0].name}`;
      default:
        return `oras pull ${url}:${version.tags[0].name}`;
    }
  }

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
      downloadedByCustomerOrganizations,
      downloadedByCount: downloadedByUsers.length + downloadedByCustomerOrganizations.length,
    };
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
    this.overlay
      .confirm(
        `This will untag "${tagName}" from ${artifact.name}. The artifact version SHA (${version.digest.substring(0, 12)}) will remain in the database. Are you sure?`
      )
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
}
