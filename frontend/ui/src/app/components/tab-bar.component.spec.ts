import {Component, signal} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {TabBarComponent, TabItem} from './tab-bar.component';

@Component({
  imports: [TabBarComponent],
  template: `
    <app-tab-bar [tabs]="tabs" [(active)]="active" (tabClick)="clicked.push($event.id)">
      <ng-template #tabSuffix let-tab let-origin="origin">
        <span class="suffix" [attr.data-has-origin]="!!origin">{{ tab.id }}-suffix</span>
      </ng-template>
    </app-tab-bar>
  `,
})
class HostComponent {
  readonly tabs: TabItem<string>[] = [
    {id: 'general', label: 'General'},
    {id: 'email', label: 'Email', disabled: true},
  ];
  readonly active = signal('general');
  readonly clicked: string[] = [];
}

describe('TabBarComponent', () => {
  it('renders the projected suffix with the tab and its overlay origin', () => {
    const fixture = TestBed.createComponent(HostComponent);
    fixture.detectChanges();

    const suffixes = fixture.nativeElement.querySelectorAll('.suffix');
    expect(suffixes.length).toBe(2);
    expect(suffixes[0].textContent).toBe('general-suffix');
    expect(suffixes[0].getAttribute('data-has-origin')).toBe('true');
  });

  it('does not select a disabled tab but still reports the click', () => {
    const fixture = TestBed.createComponent(HostComponent);
    fixture.detectChanges();

    const buttons = fixture.nativeElement.querySelectorAll('button[role="tab"]');
    buttons[1].click();
    fixture.detectChanges();

    expect(fixture.componentInstance.active()).toBe('general');
    expect(fixture.componentInstance.clicked).toEqual(['email']);
    expect(buttons[1].getAttribute('aria-disabled')).toBe('true');
  });
});
