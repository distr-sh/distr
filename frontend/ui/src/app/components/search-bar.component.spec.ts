import {Component} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {FormControl} from '@angular/forms';
import {SearchBarComponent} from './search-bar.component';

@Component({
  imports: [SearchBarComponent],
  template: `<app-search-bar [control]="search" placeholder="Filter" />`,
})
class HostComponent {
  readonly search = new FormControl('needle');
}

describe('SearchBarComponent', () => {
  // Browsers clear a search input on Escape unless the event is canceled.
  it('gives up the focus on Escape without clearing the filter', () => {
    const fixture = TestBed.createComponent(HostComponent);
    fixture.detectChanges();
    const input: HTMLInputElement = fixture.nativeElement.querySelector('input');

    input.focus();
    const event = new KeyboardEvent('keydown', {key: 'Escape', bubbles: true, cancelable: true});
    input.dispatchEvent(event);

    expect(document.activeElement).not.toBe(input);
    expect(event.defaultPrevented).toBe(true);
  });
});
