import {AsyncPipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faDownload, faEllipsis, faGlobe, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {shortDigest} from '../../util/digest';
import {SecureImagePipe} from '../../util/secureImage';
import {HasDownloads} from '../services/artifacts.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {UsersService} from '../services/users.service';

@Component({
  selector: 'app-artifact-logo',
  template: `
    @if (imageUrl(); as imageUrl) {
      <img class="size-full rounded-sm object-contain" [attr.src]="imageUrl | secureImage | async" alt="" />
    } @else {
      <span class="@container flex size-full items-center justify-center text-gray-900 dark:text-gray-400">
        <fa-icon [icon]="faBox" class="text-[80cqi]" />
      </span>
    }
  `,
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
})
export class ArtifactLogoComponent {
  public readonly imageUrl = input<string>();

  protected readonly faBox = faBox;
}

@Component({
  selector: 'app-artifact-public-badge',
  host: {
    class: 'distr-badge',
    title: 'Anyone who knows the name can pull this artifact without credentials',
  },
  template: `
    <fa-icon [icon]="faGlobe" />
    Public
  `,
  imports: [FaIconComponent],
})
export class ArtifactPublicBadgeComponent {
  protected readonly faGlobe = faGlobe;
}

@Component({
  selector: 'app-artifacts-download-count',
  template: `
    <div class="inline-flex items-center truncate text-sm text-gray-500 dark:text-gray-400">
      <fa-icon class="me-1" [icon]="faDownload" />
      {{ source().downloadsTotal }}
    </div>
  `,
  imports: [FaIconComponent],
})
export class ArtifactsDownloadCountComponent {
  public readonly source = input.required<HasDownloads>();

  protected readonly faDownload = faDownload;
}

/**
 * An avatar costs an image request or a rendered icon, and this component is repeated for every artifact version
 * on the page, so only the first few downloaders are shown and the rest are summarized as a count.
 */
const maxShownAvatars = 5;

interface ShownDownloader {
  id: string;
  title: string;
  imageUrl?: string;
}

@Component({
  selector: 'app-artifacts-downloaded-by',
  template: `
    <div class="flex justify-end -space-x-3 hover:-space-x-1 rtl:space-x-reverse">
      @for (downloader of shownDownloaders(); track downloader.id) {
        @if (downloader.imageUrl; as imageUrl) {
          <img
            class="size-8 rounded-full border-2 border-white transition-all duration-100 ease-in-out dark:border-gray-800"
            [attr.src]="imageUrl | secureImage | async"
            [title]="downloader.title" />
        } @else {
          <fa-icon
            [icon]="faUserCircle"
            size="xl"
            class="text-xl text-gray-400 transition-all duration-100 ease-in-out"
            [title]="downloader.title" />
        }
      }
      @if (remainingCount(); as count) {
        @if (count > 0) {
          <div
            class="flex size-8 items-center justify-center rounded-full border-2 border-white bg-gray-500 text-xs font-medium text-white transition-all duration-100 ease-in-out dark:border-gray-800 dark:bg-gray-700">
            +{{ count }}
          </div>
        }
      }
    </div>
  `,
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
})
export class ArtifactsDownloadedByComponent {
  public readonly source = input.required<HasDownloads>();

  private readonly usersService = inject(UsersService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);

  private readonly users = toSignal(this.usersService.getUsers());
  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());

  protected readonly shownDownloaders = computed<ShownDownloader[]>(() => {
    const source = this.source();
    const users = this.users() ?? [];
    const orgs = this.customerOrganizations() ?? [];
    const shown: ShownDownloader[] = [];

    for (const id of source.downloadedByUsers ?? []) {
      if (shown.length === maxShownAvatars) return shown;
      const user = users.find((u) => u.id === id);
      if (user) {
        shown.push({id, title: user.name ?? user.email, imageUrl: user.imageUrl});
      }
    }
    for (const id of source.downloadedByCustomerOrganizations ?? []) {
      if (shown.length === maxShownAvatars) return shown;
      const org = orgs.find((o) => o.id === id);
      if (org) {
        shown.push({id, title: org.name, imageUrl: org.imageUrl});
      }
    }
    return shown;
  });

  protected readonly remainingCount = computed(() => {
    const source = this.source();
    return (
      (source.downloadedByUsersCount ?? 0) +
      (source.downloadedByCustomerOrganizationsCount ?? 0) -
      this.shownDownloaders().length
    );
  });

  protected readonly faUserCircle = faUserCircle;
}

@Component({
  selector: 'app-artifacts-hash',
  template: `
    <span class="font-mono">{{ hashForDisplay() }}</span>
    @if (expandable()) {
      <button
        type="button"
        class="ms-1 inline-flex h-3.5 items-center justify-center rounded-xs bg-gray-200 px-1 hover:bg-gray-100 dark:bg-gray-700 dark:hover:bg-gray-600"
        (click)="showFull.set(!showFull())">
        <fa-icon [icon]="faEllipsis" />
      </button>
    }
  `,
  imports: [FaIconComponent],
})
export class ArtifactsHashComponent {
  public readonly hash = input.required<string>();
  public readonly expandable = input<boolean>(true);
  protected readonly showFull = signal(false);
  protected readonly hashForDisplay = computed(() => (this.showFull() ? this.hash() : shortDigest(this.hash())));

  protected readonly faEllipsis = faEllipsis;
}

/** An artifact with its logo and, where one is meant, the digest and tags of a version. */
@Component({
  selector: 'app-artifact-preview',
  template: `
    <app-artifact-logo class="size-10 shrink-0" [imageUrl]="imageUrl()" />
    <div class="min-w-0">
      <div class="flex min-w-0 items-baseline gap-2" [title]="label()">
        <span class="truncate">{{ name() }}</span>
        @if (digest(); as digest) {
          <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
            <app-artifacts-hash [hash]="digest" [expandable]="false" />
          </span>
        }
      </div>
      @if (tags().length > 0) {
        <div class="mt-1 flex flex-row flex-wrap gap-1">
          @for (tag of tags(); track tag) {
            <span
              class="distr-badge inline-block max-w-40 truncate"
              [class.distr-badge-success]="tag === 'latest'"
              [title]="tag">
              {{ tag }}
            </span>
          }
        </div>
      }
    </div>
  `,
  host: {class: 'flex items-center gap-2'},
  imports: [ArtifactLogoComponent, ArtifactsHashComponent],
})
export class ArtifactPreviewComponent {
  public readonly name = input.required<string>();
  public readonly imageUrl = input<string>();
  public readonly digest = input<string>();
  public readonly tags = input<string[]>([]);

  protected readonly label = computed(() =>
    [this.name(), this.tags().join(', '), this.digest()].filter((part) => part).join(' ')
  );
}
