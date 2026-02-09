import {AsyncPipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDownload, faEllipsis, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {catchError, map, of, switchMap, zip} from 'rxjs';
import {SecureImagePipe} from '../../util/secureImage';
import {HasDownloads} from '../services/artifacts.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {UsersService} from '../services/users.service';

@Component({
  selector: 'app-artifacts-download-count',
  template: `
    <div class="inline-flex items-center text-sm text-gray-500 truncate dark:text-gray-400">
      <fa-icon class="me-1" [icon]="faDownload"></fa-icon>
      {{ source().downloadsTotal }}
    </div>
  `,
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
          <fa-icon [icon]="faUserCircle" size="xl" class="text-xl text-gray-400"></fa-icon>
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
          <fa-icon [icon]="faUserCircle" size="xl" class="text-xl text-gray-400"></fa-icon>
        }
      }
      @if ((source().downloadedByCount ?? 0) - (shownUsers?.length ?? 0) - (shownCustomers?.length ?? 0); as count) {
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
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);

  public readonly downloadedByUsers = toSignal(
    toObservable(this.source).pipe(
      switchMap((dl) => {
        const userObservables = (dl.downloadedByUsers ?? []).map((id) =>
          this.usersService.getUser(id).pipe(catchError((e) => of(undefined)))
        );
        return zip(...userObservables).pipe(map((it) => it.filter((u) => u !== undefined)));
      })
    )
  );

  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly downloadedByCustomerOrganizations = computed(() => {
    const orgs = this.customerOrganizations();
    return this.source()
      .downloadedByCustomerOrganizations?.map((id) => orgs?.find((o) => o.id === id))
      .filter((o) => o !== undefined);
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
        <fa-icon [icon]="faEllipsis"></fa-icon>
      </button>
    }
  `,
  imports: [FaIconComponent],
})
export class ArtifactsHashComponent {
  public readonly hash = input.required<string>();
  public readonly expandable = input<boolean>(true);
  protected readonly showFull = signal(false);
  protected readonly hashForDisplay = computed(() => (this.showFull() ? this.hash() : this.hash().substring(0, 17)));

  protected readonly faEllipsis = faEllipsis;
}
