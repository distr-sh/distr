import {DatePipe, NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faPen, faUpRightFromSquare} from '@fortawesome/free-solid-svg-icons';
import {catchError, firstValueFrom, map, of, startWith, Subject, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {
  ActivityTimelineComponent,
  ActivityTimelineEntry,
} from '../components/activity-timeline/activity-timeline.component';
import {InnerMarkdownDirective} from '../directives/inner-markdown.directive';
import {AdvisoriesService} from '../services/advisories.service';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {AdvisoryDetail, AdvisoryImpact, AdvisoryStatus} from '../types/advisory';
import {
  affectedBadgeClass,
  affectedLabel,
  allowedStatusTransitions,
  applicationVersionLabel,
  artifactVersionLabel,
  eventLabel,
  impactStateBadgeClass,
  impactStateLabel,
  severityBadgeClass,
  statusActionLabel,
  statusBadgeClass,
  statusChangeConfirmation,
  statusLabel,
} from './advisory-display';
import {AdvisoryFormComponent, AdvisoryFormDraft} from './advisory-form.component';
import {AdvisoryResolveDialogComponent, AdvisoryResolveResult} from './advisory-resolve-dialog.component';

/**
 * Impact is loaded separately from the advisory itself, so a failed query must stay
 * distinguishable from a query that found nobody affected.
 */
type ImpactState = {state: 'loading'} | {state: 'loaded'; impact: AdvisoryImpact} | {state: 'failed'};

@Component({
  selector: 'app-advisory-detail',
  templateUrl: './advisory-detail.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DatePipe,
    NgClass,
    RouterLink,
    FaIconComponent,
    InnerMarkdownDirective,
    AdvisoryFormComponent,
    ActivityTimelineComponent,
  ],
})
export class AdvisoryDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly advisoriesService = inject(AdvisoriesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  protected readonly auth = inject(AuthService);

  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faPen = faPen;
  protected readonly faUpRightFromSquare = faUpRightFromSquare;

  protected readonly severityBadgeClass = severityBadgeClass;
  protected readonly statusBadgeClass = statusBadgeClass;
  protected readonly statusActionLabel = statusActionLabel;
  protected readonly impactStateBadgeClass = impactStateBadgeClass;
  protected readonly impactStateLabel = impactStateLabel;
  protected readonly applicationVersionLabel = applicationVersionLabel;
  protected readonly artifactVersionLabel = artifactVersionLabel;
  protected readonly statusLabel = statusLabel;
  protected readonly affectedLabel = affectedLabel;
  protected readonly affectedBadgeClass = affectedBadgeClass;

  /** Everyone outside the vendor organization is shown their own exposure, not the status. */
  protected readonly showsAffectedState = this.auth.isCustomer() || this.auth.isPartner();

  protected readonly backRoute = this.auth.isCustomer() ? '/security' : '/advisories';
  protected readonly canEdit = this.auth.isVendor() && this.auth.hasAnyRole('read_write', 'admin');

  private readonly timeline = viewChild(ActivityTimelineComponent);

  protected readonly advisory = signal<AdvisoryDetail | undefined>(undefined);
  protected readonly updatingStatus = signal(false);
  protected readonly submittingComment = signal(false);
  protected readonly editDialogOpen = signal(false);
  protected readonly editDraft = signal<AdvisoryFormDraft | undefined>(undefined);

  protected readonly timelineEntries = computed<ActivityTimelineEntry[]>(() =>
    (this.advisory()?.events ?? []).map((event) => ({
      id: event.id,
      createdAt: event.createdAt,
      userName: event.userName,
      userImageUrl: event.userImageUrl,
      action: eventLabel(event.type),
      body: event.message,
    }))
  );

  protected readonly affectedApplicationVersions = computed(() =>
    (this.advisory()?.applicationVersions ?? []).filter((v) => v.relation === 'affected')
  );
  protected readonly fixedApplicationVersions = computed(() =>
    (this.advisory()?.applicationVersions ?? []).filter((v) => v.relation === 'fixed')
  );
  protected readonly affectedArtifactVersions = computed(() =>
    (this.advisory()?.artifactVersions ?? []).filter((v) => v.relation === 'affected')
  );
  protected readonly fixedArtifactVersions = computed(() =>
    (this.advisory()?.artifactVersions ?? []).filter((v) => v.relation === 'fixed')
  );

  protected readonly nextStatuses = computed<AdvisoryStatus[]>(() => {
    const advisory = this.advisory();
    return advisory ? allowedStatusTransitions(advisory.status) : [];
  });

  private readonly refresh$ = new Subject<void>();
  /** Impact only changes when the versions or status change, not when a comment is added. */
  private readonly refreshImpact$ = new Subject<void>();
  private dialogRef: DialogRef | null = null;

  protected readonly impact = toSignal(
    this.route.paramMap.pipe(
      switchMap((params) => {
        const id = params.get('advisoryId')!;
        // switchMap discards an in-flight request when a newer one starts, so a slow
        // response for a previous advisory can never overwrite the current one.
        return this.refreshImpact$.pipe(
          startWith(0),
          switchMap(() =>
            this.advisoriesService.getImpact(id).pipe(
              map((impact): ImpactState => ({state: 'loaded', impact})),
              catchError(() => of<ImpactState>({state: 'failed'})),
              startWith<ImpactState>({state: 'loading'})
            )
          )
        );
      })
    ),
    {initialValue: {state: 'loading'} as ImpactState}
  );

  protected readonly impactResult = computed(() => {
    const impact = this.impact();
    return impact.state === 'loaded' ? impact.impact : undefined;
  });
  protected readonly impactFailed = computed(() => this.impact().state === 'failed');

  /** Drives the customer-facing verdict: whether anything of theirs still runs an affected version. */
  protected readonly stillAffectedCount = computed(
    () => this.impactResult()?.deployments.filter((deployment) => deployment.state === 'affected').length ?? 0
  );

  constructor() {
    this.route.paramMap
      .pipe(
        switchMap((params) => {
          const id = params.get('advisoryId')!;
          return this.refresh$.pipe(
            startWith(0),
            switchMap(() =>
              this.advisoriesService.get(id).pipe(
                // Handled here rather than in the subscriber so that one failed load does not
                // tear down the subscription and leave the page unable to reload.
                catchError((e) => {
                  const message = getFormDisplayedError(e);
                  if (message) {
                    this.toast.error(message);
                  }
                  return of(undefined);
                })
              )
            ),
            // Drops the previously shown advisory as soon as the route id changes, so a slow
            // or failing load can never leave the wrong record under the new URL.
            startWith(undefined)
          );
        }),
        takeUntilDestroyed()
      )
      .subscribe((detail) => this.advisory.set(detail));
  }

  protected openEditDialog(templateRef: TemplateRef<unknown>): void {
    this.closeEditDialog();
    this.editDialogOpen.set(true);
    this.dialogRef = this.overlay.showModal(templateRef);
    this.dialogRef
      .result()
      .pipe(take(1))
      .subscribe(() => {
        this.dialogRef = null;
        this.editDialogOpen.set(false);
      });
  }

  protected closeEditDialog(): void {
    this.dialogRef?.dismiss();
    this.dialogRef = null;
    this.editDialogOpen.set(false);
  }

  protected cancelEditDialog(): void {
    this.editDraft.set(undefined);
    this.closeEditDialog();
  }

  protected onSaved(detail: AdvisoryDetail): void {
    this.editDraft.set(undefined);
    this.closeEditDialog();
    this.advisory.set(detail);
    this.refresh$.next();
    this.refreshImpact$.next();
  }

  protected async changeStatus(status: AdvisoryStatus): Promise<void> {
    const advisory = this.advisory();
    if (!advisory) {
      return;
    }
    let comment: string | undefined;
    if (status === 'resolved') {
      const result = await firstValueFrom(
        this.overlay.showModal<AdvisoryResolveResult>(AdvisoryResolveDialogComponent).result()
      );
      if (!result) {
        return;
      }
      comment = result.comment;
    } else {
      const confirmation = statusChangeConfirmation(status);
      if (confirmation && !(await firstValueFrom(this.overlay.confirm(confirmation)))) {
        return;
      }
    }

    this.updatingStatus.set(true);
    try {
      const updated = await firstValueFrom(this.advisoriesService.updateStatus(advisory.id, {status}));
      this.advisory.set(updated);
      // Posted after the status change so that a failing comment cannot leave the advisory
      // unresolved; the timeline then reads as the transition followed by its explanation.
      if (comment) {
        await firstValueFrom(this.advisoriesService.createComment(advisory.id, {content: comment}));
      }
      this.toast.success('Status updated');
      this.refresh$.next();
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.updatingStatus.set(false);
    }
  }

  protected async submitComment(content: string): Promise<void> {
    const advisory = this.advisory();
    if (!advisory) {
      return;
    }
    this.submittingComment.set(true);
    try {
      await firstValueFrom(this.advisoriesService.createComment(advisory.id, {content}));
      this.timeline()?.reset();
      this.refresh$.next();
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.submittingComment.set(false);
    }
  }
}
