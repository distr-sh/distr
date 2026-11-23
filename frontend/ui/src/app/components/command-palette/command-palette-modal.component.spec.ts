import {signal} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {provideRouter, Router, UrlTree} from '@angular/router';
import {Application, DeploymentTarget} from '@distr-sh/distr-sdk';
import {faBox, faBoxesStacked, faCube, faHome} from '@fortawesome/free-solid-svg-icons';
import {BehaviorSubject, Observable, of} from 'rxjs';
import {ContextSwitcherService, SwitcherOrganization} from '../../navigation/context-switcher.service';
import {NavigationSection} from '../../navigation/navigation';
import {NavigationService} from '../../navigation/navigation.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactsService, ArtifactWithTags} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationService} from '../../services/organization.service';
import {OrganizationWithUserRole} from '../../types/organization';
import {CommandPaletteModalComponent} from './command-palette-modal.component';

const sections: NavigationSection[] = [
  {
    id: 'distribution',
    label: 'Distribution',
    entries: [
      {kind: 'link', id: 'home', label: 'Home', icon: faHome, route: '/home', hasIndicator: false, shortcut: 'h'},
      {
        kind: 'link',
        id: 'applications',
        label: 'Applications',
        icon: faCube,
        route: '/applications',
        hasIndicator: false,
      },
      {kind: 'locked', id: 'security', label: 'Security', icon: faBox, plan: 'business'},
      {kind: 'link', id: 'registry', label: 'Registry', icon: faBox, route: '/registry', hasIndicator: false},
      {
        kind: 'link',
        id: 'deployments',
        label: 'Deployments',
        icon: faBoxesStacked,
        route: '/deployments',
        hasIndicator: false,
      },
    ],
  },
];

const applications = [{id: 'app-1', name: 'Shop'}] as Application[];
const artifacts = [{id: 'artifact-1', name: 'billing-image'}] as ArtifactWithTags[];
const deploymentTargets = [
  {
    id: 'target-1',
    name: 'edge-prod',
    customerOrganization: {id: 'customer-1', name: 'Globex'},
    deployments: [{application: {id: 'app-1', name: 'Shop'}}],
  },
] as DeploymentTarget[];

const org = {id: 'org-1', name: 'Acme'} as OrganizationWithUserRole;
const switcherTree: SwitcherOrganization[] = [
  {
    org,
    current: true,
    lenses: [
      {customerOrganizationId: 'customer-1', customerOrganizationName: 'Globex', current: false},
      {customerOrganizationId: 'customer-2', customerOrganizationName: 'Initech', current: true},
    ],
  },
];

describe('CommandPaletteModalComponent', () => {
  const enterLens = vi.fn(async () => undefined);

  // jsdom does not implement scrolling
  beforeAll(() => (Element.prototype.scrollIntoView = () => undefined));

  function render(
    lensHome?: OrganizationWithUserRole,
    navigationSections = signal(sections),
    applications$: Observable<Application[]> = of(applications)
  ) {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        {provide: NavigationService, useValue: {sections: navigationSections}},
        {
          provide: ContextSwitcherService,
          useValue: {
            switcherTree: signal(switcherTree),
            isOrganizationListed: signal(true),
            lensHome: signal(lensHome),
            enterLens,
          },
        },
        {provide: AuthService, useValue: {isCustomer: () => false}},
        {provide: OrganizationService, useValue: {get: () => of(org)}},
        {provide: ApplicationsService, useValue: {list: () => applications$, refresh: async () => applications}},
        {provide: ArtifactsService, useValue: {list: () => of(artifacts), refresh: async () => artifacts}},
        {
          provide: DeploymentTargetsService,
          useValue: {list: () => of(deploymentTargets), refresh: async () => deploymentTargets},
        },
        {
          provide: CustomerOrganizationsService,
          useValue: {getCustomerOrganizations: () => of([]), refresh: async () => []},
        },
      ],
    });
    const fixture = TestBed.createComponent(CommandPaletteModalComponent);
    fixture.detectChanges();
    return fixture;
  }

  function press(fixture: ReturnType<typeof render>, key: string, init: KeyboardEventInit = {}) {
    const event = new KeyboardEvent('keydown', {key, bubbles: true, cancelable: true, ...init});
    fixture.nativeElement.querySelector('input').dispatchEvent(event);
    fixture.detectChanges();
    return event;
  }

  function search(fixture: ReturnType<typeof render>, term: string) {
    const input: HTMLInputElement = fixture.nativeElement.querySelector('input');
    input.value = term;
    input.dispatchEvent(new Event('input'));
    fixture.detectChanges();
  }

  function selected(fixture: ReturnType<typeof render>): string {
    return fixture.nativeElement.querySelector('[aria-selected="true"] span.grow').textContent.trim();
  }

  function labels(fixture: ReturnType<typeof render>): string[] {
    return Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('[role="option"] span.grow')).map((label) =>
      label.textContent!.trim()
    );
  }

  it('lists the reachable pages and the lenses that are not entered', () => {
    const fixture = render();
    expect(labels(fixture)).toEqual([
      'Shop',
      'Running',
      'Stale',
      'Error',
      'Outdated',
      'edge-prod',
      'billing-image',
      'Home',
      'Applications',
      'Registry',
      'Deployments',
      'Globex',
    ]);
  });

  it('offers to return to the vendor portal in place of switching to it from a lens', () => {
    const fixture = render(org);
    search(fixture, 'g p');
    expect(labels(fixture)).toEqual(['Return to Vendor Portal']);
    search(fixture, 'acme');
    expect(labels(fixture)).toEqual(['Return to Vendor Portal']);
  });

  it('runs the command the arrow keys moved to, across groups', async () => {
    const fixture = render();
    const navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);

    press(fixture, 'ArrowUp');
    press(fixture, 'Enter');
    await fixture.whenStable();

    expect(enterLens).toHaveBeenCalledWith(switcherTree[0].lenses[0]);
    expect(navigateByUrl).not.toHaveBeenCalled();
  });

  it('clears the search on the first Escape and lets the second one reach the overlay', () => {
    const fixture = render();
    const input: HTMLInputElement = fixture.nativeElement.querySelector('input');
    const reachedBody = vi.fn();
    document.body.addEventListener('keydown', reachedBody);
    search(fixture, 'reg');
    input.focus();

    press(fixture, 'Escape');
    expect(input.value).toBe('');
    expect(document.activeElement).toBe(input);
    expect(reachedBody).not.toHaveBeenCalled();

    press(fixture, 'Escape');
    expect(reachedBody).toHaveBeenCalled();
    document.body.removeEventListener('keydown', reachedBody);
  });

  it('lists the entities of a page that becomes reachable while the palette is open', () => {
    const navigationSections = signal<NavigationSection[]>([]);
    const fixture = render(undefined, navigationSections);
    expect(labels(fixture)).not.toContain('edge-prod');

    navigationSections.set(sections);
    fixture.detectChanges();
    expect(labels(fixture)).toContain('edge-prod');
  });

  it('keeps the first row selected while lists load above it, and a moved selection on its command', () => {
    const applications$ = new BehaviorSubject<Application[]>([]);
    const fixture = render(undefined, signal(sections), applications$);

    applications$.next(applications);
    fixture.detectChanges();
    expect(selected(fixture)).toBe('Shop');

    press(fixture, 'ArrowDown');
    applications$.next([...applications, {id: 'app-2', name: 'Blog'} as Application]);
    fixture.detectChanges();
    expect(selected(fixture)).toBe('Running');
  });

  it('moves the selection with Tab instead of the focus', () => {
    const fixture = render();

    expect(press(fixture, 'Tab').defaultPrevented).toBe(true);
    expect(selected(fixture)).toBe('Running');
    expect(press(fixture, 'Tab', {shiftKey: true}).defaultPrevented).toBe(true);
    expect(selected(fixture)).toBe('Shop');
  });

  it('finds a page by its go-to keys', () => {
    const fixture = render();
    search(fixture, 'g h');
    expect(labels(fixture)).toEqual(['Home']);
  });

  it('starts over at the first match when the search changes', () => {
    const fixture = render();
    const navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);

    press(fixture, 'ArrowDown');
    search(fixture, 'reg');
    press(fixture, 'Enter');

    expect(labels(fixture)).toEqual(['Registry']);
    expect(navigateByUrl).toHaveBeenCalledWith('/registry');
  });

  it('opens a matching application at its detail page and finds the targets it is deployed on', () => {
    const fixture = render();
    const navigateByUrl = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);

    search(fixture, 'shop');
    press(fixture, 'Enter');

    expect(labels(fixture)).toEqual(['Shop', 'edge-prod']);
    expect(navigateByUrl).toHaveBeenCalledWith('/applications/app-1');
  });

  it('filters the deployments page by a status', () => {
    const fixture = render();
    const router = TestBed.inject(Router);
    const navigateByUrl = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);

    search(fixture, 'error');
    press(fixture, 'Enter');

    const url = navigateByUrl.mock.calls[0][0] as UrlTree;
    expect(router.serializeUrl(url)).toBe('/deployments?status=error');
  });

  it('filters the deployments page by a matching deployment target', () => {
    const fixture = render();
    const router = TestBed.inject(Router);
    const navigateByUrl = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);

    search(fixture, 'prod');
    press(fixture, 'Enter');

    const url = navigateByUrl.mock.calls[0][0] as UrlTree;
    expect(router.serializeUrl(url)).toBe('/deployments?search=edge-prod');
  });
});
