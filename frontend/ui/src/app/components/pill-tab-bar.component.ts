import {Component, input, model} from '@angular/core';
import {TabItem} from './tab-bar.component';

/** The compact tab bar that switches between the sections of a form or a panel. */
@Component({
  selector: 'app-pill-tab-bar',
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
      </button>
    }
  `,
  host: {role: 'tablist'},
  styleUrl: './pill-tab-bar.component.scss',
})
export class PillTabBarComponent<T extends string> {
  public readonly tabs = input.required<readonly TabItem<T>[]>();
  public readonly active = model.required<T>();
}
