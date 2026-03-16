import {Component, inject, input, output, signal} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {DeploymentTarget, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  distinctUntilChanged,
  EMPTY,
  filter,
  firstValueFrom,
  map,
  startWith,
  Subject,
  switchMap,
  timer,
} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {DeploymentLogsTableComponent} from './deployment-logs-table.component';
import {DeploymentStatusTableComponent} from './deployment-status-table.component';

const resourceRefreshInterval = 15_000;

@Component({
  selector: 'app-deployment-status-modal',
  templateUrl: './deployment-status-modal.component.html',
  imports: [DeploymentLogsTableComponent, DeploymentStatusTableComponent, FaIconComponent],
})
export class DeploymentStatusModalComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
  public readonly closed = output<void>();

  protected readonly faXmark = faXmark;

  private readonly deploymentLogs = inject(DeploymentLogsService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);

  private readonly refreshResources$ = new Subject<void>();

  private readonly deploymentId$ = toObservable(this.deployment).pipe(
    map((d) => d.id),
    filter((id) => id !== undefined),
    distinctUntilChanged()
  );

  protected readonly resources = toSignal(
    combineLatest([this.deploymentId$, this.refreshResources$.pipe(startWith(undefined))]).pipe(
      switchMap(([id]) =>
        timer(0, resourceRefreshInterval).pipe(
          switchMap(() => this.deploymentLogs.getResources(id).pipe(catchError(() => EMPTY)))
        )
      )
    )
  );

  /**
   * `null` means agent status
   */
  protected readonly selectedResource = signal<string | null>(null);

  protected async deleteLogRecords(deploymentId: string, resource: string) {
    if (await firstValueFrom(this.overlay.confirm(`Permanently delete all log records for ${resource}?`))) {
      try {
        await firstValueFrom(this.deploymentLogs.delete(deploymentId, [resource]));
        this.toast.success(`Log records for resource ${resource} have been deleted.`);
        this.refreshResources$.next();
        if (this.selectedResource() === resource) {
          this.selectedResource.set(null);
        }
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      }
    }
  }

  protected hideModal() {
    this.closed.emit();
  }
}
