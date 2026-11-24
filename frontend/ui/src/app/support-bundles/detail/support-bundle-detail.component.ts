import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, viewChild} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faComment, faDownload, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, firstValueFrom, map, of, startWith, Subject, switchMap} from 'rxjs';
import {downloadBlob} from '../../../util/blob';
import {getFormDisplayedError, isNotFound, toastDisplayedError} from '../../../util/errors';
import {organizationKind} from '../../../util/organization-kind';
import {
  ActivityTimelineComponent,
  ActivityTimelineEntry,
} from '../../components/activity-timeline/activity-timeline.component';
import {AlertComponent} from '../../components/alert/alert.component';
import {ClipComponent} from '../../components/clip.component';
import {DeploymentTypeBadgeComponent} from '../../components/deployment-type-badge.component';
import {PageComponent} from '../../components/page.component';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {OverlayService} from '../../services/overlay.service';
import {SupportBundlesService, supportBundleZipFileName} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundle, SupportBundleComment, SupportBundleDetail} from '../../types/support-bundle';
import {supportBundleDeleteConfirm} from '../support-bundle-display';
import {SupportBundleStatusBadgeComponent} from '../support-bundle-status-badge.component';
import {buildResourceTree} from './resource-tree';
import {SupportBundleResourceFolderComponent} from './support-bundle-resource-folder.component';

interface Author {
  name: string;
  imageUrl?: string;
}

interface Actor {
  name?: string;
  imageUrl?: string;
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
  deleted?: boolean;
}

function createdBy(bundle: SupportBundle): Actor {
  return {
    name: bundle.createdByUserName,
    customerOrganizationId: bundle.createdByCustomerOrganizationId,
    partnerOrganizationId: bundle.createdByPartnerOrganizationId,
    deleted: bundle.createdByDeleted,
  };
}

function statusChangedBy(bundle: SupportBundle): Actor {
  return {
    name: bundle.statusChangedByUserName,
    customerOrganizationId: bundle.statusChangedByCustomerOrganizationId,
    partnerOrganizationId: bundle.statusChangedByPartnerOrganizationId,
    deleted: bundle.statusChangedByDeleted,
  };
}

function commentedBy(comment: SupportBundleComment): Actor {
  return {
    name: comment.userName,
    imageUrl: comment.userImageUrl,
    customerOrganizationId: comment.userCustomerOrganizationId,
    partnerOrganizationId: comment.userPartnerOrganizationId,
    deleted: comment.userDeleted,
  };
}

@Component({
  selector: 'app-support-bundle-detail',
  templateUrl: './support-bundle-detail.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DatePipe,
    FaIconComponent,
    AlertComponent,
    ClipComponent,
    ActivityTimelineComponent,
    PageComponent,
    SupportBundleStatusBadgeComponent,
    SupportBundleResourceFolderComponent,
    DeploymentTypeBadgeComponent,
  ],
})
export class SupportBundleDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly organizationService = inject(OrganizationService);
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly context = inject(ContextService);
  protected readonly auth = inject(AuthService);

  protected readonly faCheck = faCheck;
  protected readonly faComment = faComment;
  protected readonly faDownload = faDownload;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  private readonly timeline = viewChild(ActivityTimelineComponent);

  protected readonly bundle = signal<SupportBundleDetail | undefined>(undefined);
  protected readonly justInitialized = toSignal(
    this.route.queryParamMap.pipe(map((params) => params.get('initialized') === 'true')),
    {initialValue: false}
  );
  protected readonly updatingStatus = signal(false);
  protected readonly submittingComment = signal(false);
  protected readonly downloading = signal(false);
  protected readonly deleting = signal(false);

  protected readonly title = computed(() => {
    const bundle = this.bundle();
    return bundle?.title || `Support Request ${bundle?.id.substring(0, 8)}`;
  });

  protected readonly resourceTree = computed(() => buildResourceTree(this.bundle()?.resources ?? []));

  /** Only a customer or partner can be denied a user's identity. */
  private readonly vendor = toSignal<Author | undefined>(
    this.auth.isVendor()
      ? of(undefined)
      : combineLatest([
          this.organizationService.get(),
          this.organizationBranding.get().pipe(
            catchError((e) => {
              if (!isNotFound(e)) {
                toastDisplayedError(this.toast, e);
              }
              return of(undefined);
            })
          ),
        ]).pipe(map(([org, branding]) => ({name: org.name, imageUrl: branding?.logoImageId})))
  );
  private readonly partner = toSignal(this.context.getPartnerOrganization());

  protected readonly createdByName = computed(() => {
    const bundle = this.bundle();
    return bundle && this.author(createdBy(bundle))?.name;
  });

  protected readonly statusChangedByName = computed(() => {
    const bundle = this.bundle();
    return bundle && this.author(statusChangedBy(bundle))?.name;
  });

  protected readonly timelineEntries = computed<ActivityTimelineEntry[]>(() =>
    (this.bundle()?.comments ?? []).map((comment) => {
      const commentAuthor = this.author(commentedBy(comment));
      return {
        id: comment.id,
        createdAt: comment.createdAt,
        userName: commentAuthor?.name,
        userImageUrl: commentAuthor?.imageUrl,
        action: 'commented',
        body: comment.content,
      };
    })
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
      this.overlay.confirm('Are you sure you want to mark this support request as resolved?')
    );
    if (!confirmed) {
      return;
    }
    this.updatingStatus.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.updateStatus(bundle.id, {status: 'resolved'}));
      this.toast.success('Support request marked as resolved');
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
      this.overlay.confirm('Are you sure you want to cancel this support request?')
    );
    if (!confirmed) {
      return;
    }
    this.updatingStatus.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.updateStatus(bundle.id, {status: 'canceled'}));
      this.toast.success('Support request canceled');
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

  protected async deleteBundle(): Promise<void> {
    const bundle = this.bundle();
    if (!bundle || this.deleting()) {
      return;
    }
    const confirmed = await firstValueFrom(this.overlay.confirm(supportBundleDeleteConfirm));
    if (!confirmed) {
      return;
    }
    this.deleting.set(true);
    try {
      await firstValueFrom(this.supportBundlesService.delete(bundle.id));
      this.toast.success('Support request deleted');
      await this.router.navigate(['/support'], {replaceUrl: true});
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.deleting.set(false);
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

  /**
   * Who did something, as far as the viewer may know. A vendor or partner member the API withholds stands in as their
   * organization: the partner if the viewer's customer organization is assigned to it, the vendor otherwise. A
   * deleted user stays anonymous.
   */
  private author(actor: Actor): Author | undefined {
    if (actor.name) {
      return {name: actor.name, imageUrl: actor.imageUrl};
    }
    if (actor.deleted) {
      return undefined;
    }
    switch (organizationKind(actor)) {
      case 'vendor':
        return this.vendor();
      case 'partner': {
        const partner = this.partner();
        return partner && actor.partnerOrganizationId === partner.id ? {name: partner.name} : this.vendor();
      }
      default:
        return undefined;
    }
  }
}
