import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, input, output} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faPaperPlane} from '@fortawesome/free-solid-svg-icons';
import {AvatarComponent} from '../avatar.component';

/**
 * A single entry in an activity timeline. Features map their own event or comment type onto
 * this shape, so the component stays independent of where the entries come from.
 */
export interface ActivityTimelineEntry {
  id: string;
  createdAt: string;
  /** Omitted for entries recorded without a user, e.g. automated events. */
  userName?: string;
  userImageUrl?: string;
  /** What the user did, phrased to follow the name, e.g. "commented". */
  action: string;
  body?: string;
}

@Component({
  selector: 'app-activity-timeline',
  templateUrl: './activity-timeline.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  // Without an explicit display the host stays inline, which silently drops the vertical
  // margins that the surrounding space-y and mb utilities put on it.
  host: {class: 'block'},
  imports: [DatePipe, ReactiveFormsModule, FaIconComponent, AvatarComponent],
})
export class ActivityTimelineComponent {
  public readonly entries = input.required<ActivityTimelineEntry[]>();
  public readonly heading = input('Timeline');
  public readonly icon = input<IconDefinition>();
  public readonly emptyText = input('Nothing here yet.');
  public readonly canComment = input(false);
  public readonly submitting = input(false);
  public readonly placeholder = input('Write a comment...');

  public readonly commentSubmitted = output<string>();

  protected readonly faPaperPlane = faPaperPlane;

  protected readonly commentForm = new FormGroup({
    content: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
  });

  protected submit(): void {
    this.commentForm.markAllAsTouched();
    // Validators.required accepts whitespace, which the API rejects.
    const content = this.commentForm.controls.content.value.trim();
    if (!content) {
      return;
    }
    this.commentSubmitted.emit(content);
  }

  /**
   * Called by the parent once the comment was accepted, so that a failed submission keeps
   * the text the user typed.
   */
  public reset(): void {
    this.commentForm.reset();
  }
}
