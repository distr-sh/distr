import {DatePipe, NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, viewChild} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faCheck,
  faChevronDown,
  faChevronRight,
  faComment,
  faDownload,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {downloadBlob} from '../../../util/blob';
import {getFormDisplayedError} from '../../../util/errors';
import {
  ActivityTimelineComponent,
  ActivityTimelineEntry,
} from '../../components/activity-timeline/activity-timeline.component';
import {ClipComponent} from '../../components/clip.component';
import {PageComponent} from '../../components/page.component';
import {AuthService} from '../../services/auth.service';
import {OverlayService} from '../../services/overlay.service';
import {SupportBundlesService, supportBundleZipFileName} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleDetail} from '../../types/support-bundle';
import {supportBundleStatusBadgeClass} from '../support-bundle-display';

@Component({
  selector: 'app-support-bundle-detail',
  templateUrl: './support-bundle-detail.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [DatePipe, NgClass, RouterLink, FaIconComponent, ClipComponent, ActivityTimelineComponent, PageComponent],
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
  protected readonly faDownload = faDownload;
  protected readonly faXmark = faXmark;
  protected readonly statusBadgeClass = supportBundleStatusBadgeClass;

  private readonly timeline = viewChild(ActivityTimelineComponent);

  protected readonly bundle = signal<SupportBundleDetail | undefined>(undefined);
  protected readonly expandedResources = signal(new Set<string>());
  protected readonly updatingStatus = signal(false);
  protected readonly submittingComment = signal(false);
  protected readonly downloading = signal(false);

  protected readonly timelineEntries = computed<ActivityTimelineEntry[]>(() =>
    (this.bundle()?.comments ?? []).map((comment) => ({
      id: comment.id,
      createdAt: comment.createdAt,
      userName: comment.userName,
      userImageUrl: comment.userImageUrl,
      action: 'commented',
      body: comment.content,
    }))
  );

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

  protected shortId(id: string): string {
    return id.substring(0, 8);
  }

  protected readonly backRoute = this.auth.isCustomer() ? '/support' : '/support-bundles';

  protected async downloadResources(): Promise<void> {
    const bundle = this.bundle();
    if (!bundle || this.downloading()) {
      return;
    }
    this.downloading.set(true);
    try {
      const blob = await firstValueFrom(this.supportBundlesService.downloadResources(bundle.id));
      downloadBlob(blob, supportBundleZipFileName(bundle));
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.downloading.set(false);
    }
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

  protected async submitComment(content: string): Promise<void> {
    const bundle = this.bundle();
    if (!bundle) {
      return;
    }
    this.submittingComment.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.createComment(bundle.id, {content}));
      this.timeline()?.reset();
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
