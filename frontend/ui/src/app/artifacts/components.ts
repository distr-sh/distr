import {AsyncPipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDownload, faEllipsis, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {shortDigest} from '../../util/digest';
import {SecureImagePipe} from '../../util/secureImage';
import {HasDownloads} from '../services/artifacts.service';
import {CustomerOrganizationsCache} from '../services/customer-organizations.service';
import {UsersService} from '../services/users.service';

@Component({
  selector: 'app-artifacts-download-count',
  template: `
    <div class="inline-flex items-center text-sm text-gray-500 truncate dark:text-gray-400">
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
    <div class="flex -space-x-3 hover:-space-x-1 rtl:space-x-reverse justify-end">
      @for (downloader of shownDownloaders(); track downloader.id) {
        @if (downloader.imageUrl; as imageUrl) {
          <img
            class="size-8 border-2 border-white rounded-full dark:border-gray-800 transition-all duration-100 ease-in-out"
            [attr.src]="imageUrl | secureImage | async"
            [title]="downloader.title" />
        } @else {
          <fa-icon [icon]="faUserCircle" size="xl" class="text-xl text-gray-400" [title]="downloader.title" />
        }
      }
      @if (remainingCount(); as count) {
        @if (count > 0) {
          <div
            class="flex items-center justify-center size-8 text-xs font-medium text-white bg-gray-500 dark:bg-gray-700 border-2 border-white rounded-full dark:border-gray-800">
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
  private readonly customerOrganizationsService = inject(CustomerOrganizationsCache);

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
        class="inline-flex items-center justify-center h-3.5 ms-1 px-1 rounded-xs bg-gray-200 hover:bg-gray-100 dark:bg-gray-700 dark:hover:bg-gray-600"
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
