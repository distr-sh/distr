import {NgClass} from '@angular/common';
import {Component, computed, input, viewChild} from '@angular/core';
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
      class="text-gray-900 dark:text-gray-400 m-0.5 hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-600 dark:hover:bg-gray-700 px-2 inline-flex gap-1 items-center justify-center bg-white border-gray-200 border"
      [ngClass]="small() ? ['text-xs', 'py-0.5', 'rounded-sm'] : ['py-1', 'rounded-lg']">
      <code>{{ shortUuid() }}</code>
      <fa-icon [icon]="clipIcon()" />
    </button>
  `,
  imports: [FaIconComponent, NgClass, ClipDirective],
})
export class UuidComponent {
  public readonly uuid = input.required<string>();
  public readonly small = input(false);
  protected readonly shortUuid = computed(() => this.uuid().slice(0, 8));
  private readonly clip = viewChild.required(ClipDirective);
  protected readonly clipIcon = computed(() => (this.clip().copied() ? faClipboardCheck : faClipboard));
}
