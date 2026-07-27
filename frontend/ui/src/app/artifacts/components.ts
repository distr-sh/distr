import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDownload, faEllipsis, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {SecureImagePipe} from '../../util/secureImage';
import {ArtifactManifestType, HasDownloads} from '../services/artifacts.service';
import {CustomerOrganizationsCache} from '../services/customer-organizations.service';
import {UsersService} from '../services/users.service';

@Component({
  selector: 'app-artifact-type-badge',
  template: `
    @switch (type()) {
      @case ('container-image') {
        <span
          class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-md border bg-cyan-100 text-cyan-800 border-cyan-400 dark:bg-gray-700 dark:text-cyan-400">
          Container image
        </span>
      }
      @case ('helm-chart') {
        <span
          class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-md border bg-indigo-100 text-indigo-800 border-indigo-400 dark:bg-gray-700 dark:text-indigo-400">
          Helm chart
        </span>
      }
      @case ('zarf-package') {
        <span
          class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-md border bg-purple-100 text-purple-800 border-purple-400 dark:bg-gray-700 dark:text-purple-400">
          Zarf package
        </span>
      }
      @case ('generic') {
        <span
          class="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-md border bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-700 dark:text-gray-400">
          Generic OCI artifact
        </span>
      }
      <!-- signatures are companion artifacts and never shown as a badge -->
    }
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
})
export class ArtifactTypeBadgeComponent {
  public readonly type = input.required<ArtifactManifestType>();
}

@Component({
  selector: 'app-artifact-type-badges',
  template: `
    @for (type of types(); track type) {
      <app-artifact-type-badge [type]="type" />
    }
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'inline-flex flex-wrap items-center gap-1'},
  imports: [ArtifactTypeBadgeComponent],
})
export class ArtifactTypeBadgesComponent {
  public readonly types = input<ArtifactManifestType[] | undefined>([]);
}

@Component({
  selector: 'app-artifacts-download-count',
  template: `
    <div class="inline-flex items-center text-sm text-gray-500 truncate dark:text-gray-400">
      <fa-icon class="me-1" [icon]="faDownload" />
      {{ source().downloadsTotal }}
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent],
})
export class ArtifactsDownloadCountComponent {
  public readonly source = input.required<HasDownloads>();

  protected readonly faDownload = faDownload;
}

@Component({
  selector: 'app-artifacts-downloaded-by',
  template: `
    <div class="flex -space-x-3 hover:-space-x-1 rtl:space-x-reverse justify-end">
      @let shownUsers = downloadedByUsers();
      @for (user of shownUsers; track user.id) {
        @if (user.imageUrl; as imageUrl) {
          <img
            class="size-8 border-2 border-white rounded-full dark:border-gray-800 transition-all duration-100 ease-in-out"
            [attr.src]="imageUrl | secureImage | async"
            [title]="user.name ?? user.email" />
        } @else {
          <fa-icon [icon]="faUserCircle" size="xl" class="text-xl text-gray-400" [title]="user.name ?? user.email" />
        }
      }
      @let shownCustomers = downloadedByCustomerOrganizations();
      @for (customer of shownCustomers; track customer.id) {
        @if (customer.imageUrl; as imageUrl) {
          <img
            class="size-8 border-2 border-white rounded-full dark:border-gray-800 transition-all duration-100 ease-in-out"
            [attr.src]="imageUrl | secureImage | async"
            [title]="customer.name" />
        } @else {
          <fa-icon [icon]="faUserCircle" size="xl" class="text-xl text-gray-400" [title]="customer.name" />
        }
      }
      @if (count(); as count) {
        @if (count > 0) {
          <div
            class="flex items-center justify-center size-8 text-xs font-medium text-white bg-gray-500 dark:bg-gray-700 border-2 border-white rounded-full dark:border-gray-800">
            +{{ count }}
          </div>
        }
      }
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
})
export class ArtifactsDownloadedByComponent {
  public readonly source = input.required<HasDownloads>();

  private readonly usersService = inject(UsersService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsCache);

  private readonly users = toSignal(this.usersService.getUsers());
  protected readonly downloadedByUsers = computed(() => {
    const users = this.users();
    return this.source()
      .downloadedByUsers?.map((id) => users?.find((u) => u.id === id))
      .filter((u) => u !== undefined);
  });

  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly downloadedByCustomerOrganizations = computed(() => {
    const orgs = this.customerOrganizations();
    return this.source()
      .downloadedByCustomerOrganizations?.map((id) => orgs?.find((o) => o.id === id))
      .filter((o) => o !== undefined);
  });

  protected readonly count = computed(() => {
    return (
      (this.source().downloadedByUsersCount ?? 0) +
      (this.source().downloadedByCustomerOrganizationsCount ?? 0) -
      (this.downloadedByUsers()?.length ?? 0) -
      (this.downloadedByCustomerOrganizations()?.length ?? 0)
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
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent],
})
export class ArtifactsHashComponent {
  public readonly hash = input.required<string>();
  public readonly expandable = input<boolean>(true);
  protected readonly showFull = signal(false);
  protected readonly hashForDisplay = computed(() => (this.showFull() ? this.hash() : this.hash().substring(0, 17)));

  protected readonly faEllipsis = faEllipsis;
}
