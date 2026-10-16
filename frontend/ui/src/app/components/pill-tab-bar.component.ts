import {NgTemplateOutlet} from '@angular/common';
import {Component, contentChild, input, model, TemplateRef} from '@angular/core';
import {TabItem} from './tab-bar.component';

/** The compact tab bar that switches between the sections of a form or a panel. */
@Component({
  selector: 'app-pill-tab-bar',
  imports: [NgTemplateOutlet],
  template: `
    @for (tab of tabs(); track tab.id) {
      <button
        type="button"
        role="tab"
        class="pill-tab"
        [id]="'tab-' + tab.id"
        [attr.aria-controls]="'tabpanel-' + tab.id"
        [attr.aria-selected]="tab.id === active()"
        [disabled]="!!tab.disabled"
        (click)="active.set(tab.id)">
        {{ tab.label }}
        @if (tabSuffix(); as tabSuffix) {
          <ng-container *ngTemplateOutlet="tabSuffix; context: {$implicit: tab}" />
        }
      </button>
    }
  `,
  host: {role: 'tablist'},
  styleUrl: './pill-tab-bar.component.scss',
})
export class PillTabBarComponent<T extends string> {
  public readonly tabs = input.required<readonly TabItem<T>[]>();
  public readonly active = model.required<T>();

  /** Rendered after the label of each tab, with the tab as context. */
  protected readonly tabSuffix = contentChild<TemplateRef<unknown>>('tabSuffix');
}
