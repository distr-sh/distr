import {ChangeDetectionStrategy, Component, computed} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {ClosableDialog} from '../components/confirm-dialog/closable-dialog';

/** What the dialog closed with. A dismiss yields null and resolves nothing. */
export interface AdvisoryResolveResult {
  /** Posted to the timeline after the status change, when the vendor wrote one. */
  comment?: string;
}

/**
 * Asks how to resolve an advisory. Resolving is the step that tells customers a fix has
 * shipped, and the reasoning behind it is usually worth recording, so the note is offered
 * here rather than left to a separate trip to the timeline. It stays optional: an advisory
 * whose description already says everything needs no further explanation.
 */
@Component({
  selector: 'app-advisory-resolve-dialog',
  templateUrl: './advisory-resolve-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, ReactiveFormsModule],
})
export class AdvisoryResolveDialogComponent extends ClosableDialog<AdvisoryResolveResult> {
  protected readonly faXmark = faXmark;

  protected readonly comment = new FormControl('', {nonNullable: true});
  private readonly commentValue = toSignal(this.comment.valueChanges, {initialValue: ''});
  protected readonly hasComment = computed(() => this.commentValue().trim().length > 0);

  protected resolve(): void {
    this.dialogRef.close({});
  }

  protected resolveWithComment(): void {
    const comment = this.comment.value.trim();
    if (!comment) {
      return;
    }
    this.dialogRef.close({comment});
  }
}
