import {Component, computed, inject, linkedSignal, output, Signal, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {Router, UrlTree} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faGlasses, faShuffle} from '@fortawesome/free-solid-svg-icons';
import {map, Observable, of, switchMap} from 'rxjs';
import {organizationKind} from '../../../util/organization-kind';
import {refreshInBackground} from '../../../util/refresh';
import {
  DeploymentStatusFilter,
  deploymentStatusFilterSelectOptions,
} from '../../deployments/deployment-target-card/deployment-status-text.component';
import {deploymentTargetSearchTexts, matchesSearch} from '../../deployments/deployment-target-search';
import {ContextSwitcherService} from '../../navigation/context-switcher.service';
import {LEAVE_LENS_SHORTCUT} from '../../navigation/go-to-shortcuts.service';
import {NavigationEntry} from '../../navigation/navigation';
import {NavigationService} from '../../navigation/navigation.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactsService} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationService} from '../../services/organization.service';
import {CustomerLens} from '../../types/lens';
import {Organization} from '../../types/organization';
import {SearchBarComponent} from '../search-bar.component';

const PALETTE_DEPLOYMENT_STATUSES: DeploymentStatusFilter[] = ['running', 'error', 'stale', 'outdated'];
const PALETTE_DEPLOYMENT_STATUS_OPTIONS = deploymentStatusFilterSelectOptions.filter((option) =>
  PALETTE_DEPLOYMENT_STATUSES.includes(option.value)
);

type CommandAction =
  | {kind: 'route'; route: string | UrlTree}
  | {kind: 'href'; href: string}
  | {kind: 'organization'; organization: Organization}
  | {kind: 'lens'; lens: CustomerLens};

interface Command {
  id: string;
  label: string;
  detail?: string;
  /** What a search matches in place of the label and the detail. */
  searchTexts?: (string | undefined)[];
  icon: IconDefinition;
  shortcut?: string;
  action: CommandAction;
}

interface CommandGroup {
  id: string;
  label: string;
  commands: Command[];
}

interface IndexedCommand extends Command {
  index: number;
  elementId: string;
  /** Unique across groups, unlike the id: a customer is listed among the customers and the lenses. */
  key: string;
}

type NavigationLink = Extract<NavigationEntry, {kind: 'link'}>;

function linkedCommands<T>(
  link: NavigationLink | undefined,
  items: readonly T[],
  toCommand: (item: T, link: NavigationLink) => Command
): Command[] {
  return link ? items.map((item) => toCommand(item, link)) : [];
}

function selectionStep(event: KeyboardEvent): number {
  switch (event.key) {
    case 'ArrowDown':
      return 1;
    case 'ArrowUp':
      return -1;
    // The options are not focusable, so Tab would otherwise move the focus to the page behind the palette.
    case 'Tab':
      return event.shiftKey ? -1 : 1;
    default:
      return 0;
  }
}

@Component({
  selector: 'app-command-palette-modal',
  templateUrl: './command-palette-modal.component.html',
  host: {class: 'block origin-top'},
  imports: [FaIconComponent, SearchBarComponent],
})
export class CommandPaletteModalComponent {
  private readonly navigation = inject(NavigationService);
  private readonly switcher = inject(ContextSwitcherService);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly artifactsService = inject(ArtifactsService);
  private readonly organizationService = inject(OrganizationService);

  public readonly closed = output<void>();

  private readonly searchBar = viewChild.required(SearchBarComponent);

  protected readonly search = new FormControl<string>('', {nonNullable: true});
  private readonly searchTerm = toSignal(this.search.valueChanges.pipe(map((term) => term.trim().toLowerCase())), {
    initialValue: '',
  });

  // Only what the sidebar links to is listed, so the palette follows the same visibility rules.
  private readonly links = computed(
    () =>
      new Map(
        this.navigation
          .sections()
          .flatMap((section) => section.entries)
          .flatMap((entry): [string, NavigationLink][] => (entry.kind === 'link' ? [[entry.id, entry]] : []))
      )
  );
  private readonly applicationsLink = computed(() => this.links().get('applications'));
  private readonly customersLink = computed(() => this.links().get('customers'));
  private readonly deploymentsLink = computed(() => this.links().get('deployments'));
  private readonly artifactsLink = computed(() => this.links().get('registry') ?? this.links().get('artifacts'));

  private readonly applications = this.listWhileLinked(this.applicationsLink, () => this.applicationsService.list());
  private readonly customers = this.listWhileLinked(this.customersLink, () =>
    this.customerOrganizationsService.getCustomerOrganizations()
  );
  private readonly deploymentTargets = this.listWhileLinked(this.deploymentsLink, () =>
    this.deploymentTargetsService.list()
  );
  private readonly artifacts = this.listWhileLinked(this.artifactsLink, () => this.artifactsService.list());
  private readonly isCustomer = this.auth.isCustomer();
  private readonly organization = toSignal(this.organizationService.get());

  private readonly entityGroups = computed<CommandGroup[]>(() => [
    {
      id: 'applications',
      label: 'Applications',
      commands: linkedCommands(this.applicationsLink(), this.applications(), (application, link) => ({
        id: application.id!,
        label: application.name!,
        icon: link.icon,
        action: {kind: 'route', route: `${link.route}/${application.id}`},
      })),
    },
    {
      id: 'customers',
      label: 'Customers',
      commands: linkedCommands(this.customersLink(), this.customers(), (customer, link) => ({
        id: customer.id,
        label: customer.name,
        icon: link.icon,
        action: {kind: 'route', route: `${link.route}/${customer.id}`},
      })),
    },
    {
      id: 'deployment-status',
      label: 'Filter deployments',
      commands: linkedCommands(this.deploymentsLink(), PALETTE_DEPLOYMENT_STATUS_OPTIONS, (option, link) => ({
        id: option.value,
        label: option.label,
        icon: link.icon,
        action: {kind: 'route', route: this.router.createUrlTree([link.route], {queryParams: {status: option.value}})},
      })),
    },
    {
      id: 'deployment-targets',
      label: 'Deployments',
      commands: linkedCommands(this.deploymentsLink(), this.deploymentTargets(), (target, link) => ({
        id: target.id!,
        label: target.name,
        detail: this.isCustomer ? undefined : target.customerOrganization?.name,
        searchTexts: deploymentTargetSearchTexts(target, this.organization()?.name),
        icon: link.icon,
        action: {kind: 'route', route: this.router.createUrlTree([link.route], {queryParams: {search: target.name}})},
      })),
    },
    {
      id: 'artifacts',
      label: 'Artifacts',
      commands: linkedCommands(this.artifactsLink(), this.artifacts(), (artifact, link) => ({
        id: artifact.id,
        label: artifact.name,
        icon: link.icon,
        action: {kind: 'route', route: `${link.route}/${artifact.id}`},
      })),
    },
  ]);

  private readonly allGroups = computed<CommandGroup[]>(() => {
    const navigationGroups = this.navigation.sections().map((section) => ({
      id: section.id,
      label: section.label,
      commands: section.entries.flatMap((entry): Command[] => {
        switch (entry.kind) {
          case 'link':
            return [{...entry, action: {kind: 'route', route: entry.route}}];
          case 'external':
            return [{...entry, action: {kind: 'href', href: entry.href}}];
          case 'locked':
            return [];
        }
      }),
    }));
    const tree = this.switcher.switcherTree();
    const lensHome = this.switcher.lensHome();
    const leaveLens: Command[] = lensHome
      ? [
          {
            id: 'leave-lens',
            label: organizationKind(lensHome) === 'partner' ? 'Return to Partner Portal' : 'Return to Vendor Portal',
            detail: lensHome.name,
            icon: faArrowLeft,
            shortcut: LEAVE_LENS_SHORTCUT,
            action: {kind: 'organization', organization: lensHome},
          },
        ]
      : [];
    const organizations = this.switcher.isOrganizationListed()
      ? tree
          .filter((entry) => !entry.current && entry.org.id !== lensHome?.id)
          .map((entry) => ({
            id: entry.org.id!,
            label: entry.org.name,
            icon: faShuffle,
            action: {kind: 'organization' as const, organization: entry.org},
          }))
      : [];
    const lenses = tree
      .flatMap((entry) => entry.lenses)
      .filter((lens) => !lens.current)
      .map((lens) => ({
        id: lens.customerOrganizationId,
        label: lens.customerOrganizationName,
        icon: faGlasses,
        action: {kind: 'lens' as const, lens},
      }));
    return [
      ...this.entityGroups(),
      ...navigationGroups,
      {id: 'organizations', label: 'Switch organization', commands: [...leaveLens, ...organizations]},
      {id: 'lenses', label: 'Enter customer lens', commands: lenses},
    ];
  });

  protected readonly groups = computed(() => {
    const term = this.searchTerm();
    const shortcutTerm = term.replace(/\s/g, '');
    let index = 0;
    return this.allGroups()
      .map((group) => ({
        ...group,
        commands: group.commands
          .filter(
            (command) =>
              matchesSearch(command.searchTexts ?? [command.label, command.detail], term) ||
              (command.shortcut !== undefined && `g${command.shortcut}` === shortcutTerm)
          )
          .map((command): IndexedCommand => ({
            ...command,
            index,
            elementId: `command-palette-${index++}`,
            key: `${group.id}/${command.id}`,
          })),
      }))
      .filter((group) => group.commands.length > 0);
  });

  private readonly commands = computed(() => this.groups().flatMap((group) => group.commands));

  // The lists load and refresh while the palette is open. A selection the user moved stays on its command, the
  // first row stays first, so Enter right after opening runs the top command.
  protected readonly activeIndex = linkedSignal<{term: string; commands: IndexedCommand[]}, number>({
    source: () => ({term: this.searchTerm(), commands: this.commands()}),
    computation: ({term, commands}, previous) => {
      if (!previous || previous.value === 0 || previous.source.term !== term) {
        return 0;
      }
      const activeKey = previous.source.commands[previous.value]?.key;
      return Math.max(
        0,
        commands.findIndex((command) => command.key === activeKey)
      );
    },
  });

  protected readonly activeElementId = computed(() => this.commands()[this.activeIndex()]?.elementId);

  constructor() {
    refreshInBackground(
      ...(this.applicationsLink() ? [this.applicationsService.refresh()] : []),
      ...(this.customersLink() ? [this.customerOrganizationsService.refresh()] : []),
      ...(this.deploymentsLink() ? [this.deploymentTargetsService.refresh()] : []),
      ...(this.artifactsLink() ? [this.artifactsService.refresh()] : [])
    );
  }

  private listWhileLinked<T>(link: Signal<NavigationLink | undefined>, list: () => Observable<T[]>): Signal<T[]> {
    return toSignal(
      toObservable(computed(() => link() !== undefined)).pipe(switchMap((isLinked) => (isLinked ? list() : of([])))),
      {initialValue: []}
    );
  }

  protected onKeydown(event: KeyboardEvent) {
    const commands = this.commands();
    const step = selectionStep(event);
    if (step !== 0) {
      event.preventDefault();
      if (commands.length > 0) {
        const index = (this.activeIndex() + step + commands.length) % commands.length;
        this.activeIndex.set(index);
        document.getElementById(commands[index].elementId)?.scrollIntoView({block: 'nearest'});
      }
    } else if (event.key === 'Enter' && commands[this.activeIndex()]) {
      event.preventDefault();
      this.run(commands[this.activeIndex()]);
    } else if (event.key === 'Escape' && this.search.value) {
      // The overlay closes on every Escape that reaches the body, and the search bar blurs its input on Escape.
      event.stopPropagation();
      this.search.setValue('');
      this.searchBar().focus();
    }
  }

  protected async run(command: Command) {
    this.closed.emit();
    const action = command.action;
    switch (action.kind) {
      case 'route':
        await this.router.navigateByUrl(action.route);
        break;
      case 'href':
        window.open(action.href, '_blank', 'noopener,noreferrer');
        break;
      case 'organization':
        await this.switcher.switchOrganization(action.organization);
        break;
      case 'lens':
        await this.switcher.enterLens(action.lens);
        break;
    }
  }
}
