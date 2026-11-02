import {NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input, viewChild} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard} from '@fortawesome/free-regular-svg-icons';
import {faClipboardCheck} from '@fortawesome/free-solid-svg-icons';
import {ClipDirective} from './clip.component';

@Component({
  selector: 'app-uuid',
  template: `
    <button
      [appClip]="uuid()"
      #clip="clip"
      (click)="clip.writeClip()"
      [title]="uuid()"
      type="button"
      class="flex items-center justify-center gap-1 border border-gray-200 bg-white px-2 text-gray-900 hover:bg-gray-100 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700"
      [ngClass]="small() ? ['text-xs', 'py-0.5', 'rounded-sm'] : ['py-1', 'rounded-lg']">
      <code>{{ shortUuid() }}</code>
      <fa-icon [icon]="clipIcon()" />
    </button>
  `,
  host: {class: 'inline-block'},
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, NgClass, ClipDirective],
})
export class UuidComponent {
  public readonly uuid = input.required<string>();
  public readonly small = input(false);
  protected readonly shortUuid = computed(() => this.uuid().slice(0, 8));
  private readonly clip = viewChild.required(ClipDirective);
  protected readonly clipIcon = computed(() => (this.clip().copied() ? faClipboardCheck : faClipboard));
}
