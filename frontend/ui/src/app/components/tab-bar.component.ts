import {CdkOverlayOrigin} from '@angular/cdk/overlay';
import {NgTemplateOutlet} from '@angular/common';
import {Component, contentChild, input, model, output, TemplateRef} from '@angular/core';

export interface TabItem<T extends string = string> {
  readonly id: T;
  readonly label: string;
  readonly disabled?: boolean;
}

@Component({
  selector: 'app-tab-bar',
  imports: [CdkOverlayOrigin, NgTemplateOutlet],
  template: `
    <div role="tablist" class="flex flex-wrap border-b border-gray-200 dark:border-gray-700">
      @for (tab of tabs(); track tab.id) {
        <button
          type="button"
          role="tab"
          class="distr-tab"
          cdkOverlayOrigin
          #origin="cdkOverlayOrigin"
          [id]="'tab-' + tab.id"
          [attr.aria-controls]="'tabpanel-' + tab.id"
          [attr.aria-selected]="tab.id === active()"
          [attr.aria-disabled]="!!tab.disabled"
          (click)="onClick(tab)">
          {{ tab.label }}
          @if (tabSuffix(); as tabSuffix) {
            <ng-container *ngTemplateOutlet="tabSuffix; context: {$implicit: tab, origin: origin}" />
          }
        </button>
      }
    </div>
  `,
})
export class TabBarComponent<T extends string> {
  public readonly tabs = input.required<readonly TabItem<T>[]>();
  public readonly active = model.required<T>();
  /** Emitted for every tab, including disabled ones, which are not selected but may react. */
  public readonly tabClick = output<TabItem<T>>();

  /** Rendered after the label of each tab, with the tab and the tab's overlay origin as context. */
  protected readonly tabSuffix = contentChild<TemplateRef<unknown>>('tabSuffix');

  protected onClick(tab: TabItem<T>) {
    if (!tab.disabled) {
      this.active.set(tab.id);
    }
    this.tabClick.emit(tab);
  }
}
