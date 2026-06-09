import {ChangeDetectionStrategy, Component, computed, inject, input} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {DeploymentTarget} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard} from '@fortawesome/free-regular-svg-icons';
import {faClipboardCheck} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, switchMap} from 'rxjs';
import {displayedInToast, getFormDisplayedError} from '../../../util/errors';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-connect-instructions',
  templateUrl: './connect-instructions.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent],
})
export class ConnectInstructionsComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();

  protected readonly modalConnectCommand = computed(() => this.deploymentTargetAccess()?.connectCommand);
  protected readonly modalTargetId = computed(() => this.deploymentTargetAccess()?.targetId);
  protected readonly modalTargetSecret = computed(() => this.deploymentTargetAccess()?.targetSecret);
  protected commandCopied = false;

  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly toast = inject(ToastService);

  private readonly deploymentTargetId = computed(() => this.deploymentTarget().id!);
  private readonly deploymentTargetAccess = toSignal(
    toObservable(this.deploymentTargetId).pipe(
      switchMap((id) =>
        this.deploymentTargets.requestAccess(id).pipe(
          catchError((e) => {
            if (!displayedInToast(e)) {
              this.toast.error(getFormDisplayedError(e) ?? e);
            }
            return EMPTY;
          })
        )
      )
    )
  );

  protected async copyConnectCommand() {
    if (this.modalConnectCommand) {
      await navigator.clipboard.writeText(this.modalConnectCommand()!);
    }
    this.commandCopied = true;
    setTimeout(() => (this.commandCopied = false), 2000);
  }

  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;
}
