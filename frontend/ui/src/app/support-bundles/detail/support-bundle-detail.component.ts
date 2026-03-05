import {DatePipe, NgClass} from '@angular/common';
import {Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faCheck,
  faChevronDown,
  faChevronRight,
  faComment,
  faPaperPlane,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {ClipComponent} from '../../components/clip.component';
import {AuthService} from '../../services/auth.service';
import {OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleDetail} from '../../types/support-bundle';

@Component({
  selector: 'app-support-bundle-detail',
  templateUrl: './support-bundle-detail.component.html',
  imports: [DatePipe, NgClass, ReactiveFormsModule, RouterLink, FaIconComponent, ClipComponent],
})
export class SupportBundleDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  protected readonly auth = inject(AuthService);

  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronRight = faChevronRight;
  protected readonly faCheck = faCheck;
  protected readonly faComment = faComment;
  protected readonly faPaperPlane = faPaperPlane;
  protected readonly faXmark = faXmark;

  protected readonly bundle = signal<SupportBundleDetail | undefined>(undefined);
  protected readonly expandedResources = signal(new Set<string>());
  protected readonly updatingStatus = signal(false);
  protected readonly submittingComment = signal(false);

  protected readonly commentForm = new FormGroup({
    content: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
  });

  private readonly refresh$ = new Subject<void>();

  constructor() {
    this.route.paramMap
      .pipe(
        switchMap((params) => {
          const id = params.get('supportBundleId')!;
          return this.refresh$.pipe(
            startWith(0),
            switchMap(() => this.supportBundlesService.get(id))
          );
        }),
        takeUntilDestroyed()
      )
      .subscribe({
        next: (detail) => this.bundle.set(detail),
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  protected toggleResource(resourceId: string): void {
    this.expandedResources.update((set) => {
      const next = new Set(set);
      if (next.has(resourceId)) {
        next.delete(resourceId);
      } else {
        next.add(resourceId);
      }
      return next;
    });
  }

  protected isResourceExpanded(resourceId: string): boolean {
    return this.expandedResources().has(resourceId);
  }

  protected statusClass(status: SupportBundleDetail['status']): string {
    switch (status) {
      case 'initialized':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      case 'created':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
      case 'resolved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'canceled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
      default:
        return '';
    }
  }

  protected shortId(id: string): string {
    return id.substring(0, 8);
  }

  protected backRoute(): string {
    return this.auth.isVendor() ? '/support-bundles' : '/support';
  }

  protected userInitials(name: string): string {
    return name
      .split(' ')
      .map((part) => part.charAt(0))
      .join('')
      .toUpperCase()
      .substring(0, 2);
  }

  protected async markAsResolved(): Promise<void> {
    const bundle = this.bundle();
    if (!bundle) {
      return;
    }
    const confirmed = await firstValueFrom(
      this.overlay.confirm('Are you sure you want to mark this support bundle as resolved?')
    );
    if (!confirmed) {
      return;
    }
    this.updatingStatus.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.updateStatus(bundle.id, {status: 'resolved'}));
      this.toast.success('Support bundle marked as resolved');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.updatingStatus.set(false);
    }
  }

  protected async cancelBundle(): Promise<void> {
    const bundle = this.bundle();
    if (!bundle) {
      return;
    }
    const confirmed = await firstValueFrom(
      this.overlay.confirm('Are you sure you want to cancel this support bundle?')
    );
    if (!confirmed) {
      return;
    }
    this.updatingStatus.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.updateStatus(bundle.id, {status: 'canceled'}));
      this.toast.success('Support bundle canceled');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.updatingStatus.set(false);
    }
  }

  protected async submitComment(): Promise<void> {
    this.commentForm.markAllAsTouched();
    if (!this.commentForm.valid) {
      return;
    }
    const bundle = this.bundle();
    if (!bundle) {
      return;
    }
    this.submittingComment.set(true);
    try {
      await firstValueFrom(
        this.supportBundlesService.createComment(bundle.id, {
          content: this.commentForm.controls.content.value,
        })
      );
      this.commentForm.reset();
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.submittingComment.set(false);
    }
  }
}
