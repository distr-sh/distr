import {signal} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {provideRouter, Router} from '@angular/router';
import {faHome} from '@fortawesome/free-solid-svg-icons';
import {GoToShortcutsService} from './go-to-shortcuts.service';
import {NavigationSection} from './navigation';
import {NavigationService} from './navigation.service';

const sections: NavigationSection[] = [
  {
    id: 'main',
    label: 'Main',
    entries: [
      {kind: 'link', id: 'home', label: 'Home', icon: faHome, route: '/home', shortcut: 'h'},
      {kind: 'locked', id: 'security', label: 'Security', icon: faHome, plan: 'business'},
    ],
  },
];

function press(service: GoToShortcutsService, key: string, init: KeyboardEventInit = {}, target?: EventTarget) {
  const event = new KeyboardEvent('keydown', {key, cancelable: true, ...init});
  if (target) {
    Object.defineProperty(event, 'target', {value: target});
  }
  service.onKeydown(event);
  return event;
}

describe('GoToShortcutsService', () => {
  let service: GoToShortcutsService;
  let navigateByUrl: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        GoToShortcutsService,
        {provide: NavigationService, useValue: {sections: signal(sections)}},
      ],
    });
    service = TestBed.inject(GoToShortcutsService);
    navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
  });

  afterEach(() => vi.useRealTimers());

  it('navigates to the entry whose shortcut follows "g"', () => {
    press(service, 'g');
    const event = press(service, 'h');
    expect(navigateByUrl).toHaveBeenCalledWith('/home');
    expect(event.defaultPrevented).toBe(true);
  });

  it('keeps the sequence while "g" is held down', () => {
    press(service, 'g');
    press(service, 'g', {repeat: true});
    press(service, 'h');
    expect(navigateByUrl).toHaveBeenCalledWith('/home');
  });

  it('ignores the shortcut key once the sequence timed out', () => {
    vi.useFakeTimers();
    press(service, 'g');
    vi.advanceTimersByTime(2000);
    press(service, 'h');
    expect(navigateByUrl).not.toHaveBeenCalled();
  });

  it('ignores keys typed into a text field', () => {
    const input = document.createElement('input');
    press(service, 'g', {}, input);
    press(service, 'h', {}, input);
    expect(navigateByUrl).not.toHaveBeenCalled();
  });

  it('cancels the sequence on an unknown key', () => {
    press(service, 'g');
    press(service, 'x');
    press(service, 'h');
    expect(navigateByUrl).not.toHaveBeenCalled();
  });
});
