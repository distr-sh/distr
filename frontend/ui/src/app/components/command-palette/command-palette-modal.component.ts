import {Component, computed, inject, linkedSignal, output} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {Router, UrlTree} from '@angular/router';
import {Application, CustomerOrganizationWithUsage, DeploymentTarget} from '@distr-sh/distr-sdk';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faGlasses, faShuffle} from '@fortawesome/free-solid-svg-icons';
import {map, of} from 'rxjs';
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
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationService} from '../../services/organization.service';
import {CustomerLens} from '../../types/lens';
import {Organization} from '../../types/organization';
import {SearchBarComponent} from '../search-bar.component';

const SEARCH_RESULT_LIMIT = 5;

const PALETTE_DEPLOYMENT_STATUSES: DeploymentStatusFilter[] = ['running', 'error', 'stale', 'outdated'];

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
  /** Shows the group only while the user searches, with at most SEARCH_RESULT_LIMIT matches. */
  isSearchOnly?: boolean;
}

interface IndexedCommand extends Command {
  index: number;
  elementId: string;
}

type NavigationLink = Extract<NavigationEntry, {kind: 'link'}>;

@Component({
  selector: 'app-command-palette-modal',
  templateUrl: './command-palette-modal.component.html',
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
  private readonly organizationService = inject(OrganizationService);

  public readonly closed = output<void>();

  protected readonly search = new FormControl<string>('', {nonNullable: true});
  private readonly searchTerm = toSignal(this.search.valueChanges.pipe(map((term) => term.trim().toLowerCase())), {
    initialValue: '',
  });

  // Only what the sidebar links to is searched, so the palette follows the same visibility rules.
  private readonly links = new Map(
    this.navigation
      .sections()
      .flatMap((section) => section.entries)
      .flatMap((entry): [string, NavigationLink][] => (entry.kind === 'link' ? [[entry.id, entry]] : []))
  );
  private readonly applicationsLink = this.links.get('applications');
  private readonly customersLink = this.links.get('customers');
  private readonly deploymentsLink = this.links.get('deployments');

  private readonly applications = toSignal(
    this.applicationsLink ? this.applicationsService.list() : of<Application[]>([]),
    {initialValue: []}
  );
  private readonly customers = toSignal(
    this.customersLink
      ? this.customerOrganizationsService.getCustomerOrganizations()
      : of<CustomerOrganizationWithUsage[]>([]),
    {initialValue: []}
  );
  private readonly deploymentTargets = toSignal(
    this.deploymentsLink ? this.deploymentTargetsService.list() : of<DeploymentTarget[]>([]),
    {initialValue: []}
  );
  private readonly isCustomer = this.auth.isCustomer();
  private readonly organization = toSignal(this.organizationService.get());

  private readonly deploymentStatusGroup: CommandGroup = {
    id: 'deployment-status',
    label: 'Filter deployments',
    commands: this.deploymentsLink
      ? deploymentStatusFilterSelectOptions
          .filter((option) => PALETTE_DEPLOYMENT_STATUSES.includes(option.value))
          .map((option) => ({
            id: option.value,
            label: option.label,
            icon: this.deploymentsLink!.icon,
            action: {
              kind: 'route',
              route: this.router.createUrlTree(['/deployments'], {queryParams: {status: option.value}}),
            },
          }))
      : [],
  };

  private readonly searchGroups = computed<CommandGroup[]>(() => [
    {
      id: 'applications',
      label: 'Applications',
      isSearchOnly: true,
      commands: this.applications().map((application) => ({
        id: application.id!,
        label: application.name!,
        icon: this.applicationsLink!.icon,
        action: {kind: 'route', route: `/applications/${application.id}`},
      })),
    },
    {
      id: 'customers',
      label: 'Customers',
      isSearchOnly: true,
      commands: this.customers().map((customer) => ({
        id: customer.id,
        label: customer.name,
        icon: this.customersLink!.icon,
        action: {kind: 'route', route: `/customers/${customer.id}`},
      })),
    },
    {
      id: 'deployment-targets',
      label: 'Deployment targets',
      isSearchOnly: true,
      commands: this.deploymentTargets().map((target) => ({
        id: target.id!,
        label: target.name,
        detail: this.isCustomer ? undefined : target.customerOrganization?.name,
        searchTexts: deploymentTargetSearchTexts(target, this.organization()?.name),
        icon: this.deploymentsLink!.icon,
        action: {
          kind: 'route',
          route: this.router.createUrlTree(['/deployments'], {queryParams: {search: target.name}}),
        },
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
      ...navigationGroups,
      this.deploymentStatusGroup,
      ...this.searchGroups(),
      {id: 'organizations', label: 'Switch organization', commands: [...leaveLens, ...organizations]},
      {id: 'lenses', label: 'Enter customer lens', commands: lenses},
    ];
  });

  protected readonly groups = computed(() => {
    const term = this.searchTerm();
    const shortcutTerm = term.replace(/\s/g, '');
    let index = 0;
    return this.allGroups()
      .filter((group) => term || !group.isSearchOnly)
      .map((group) => {
        const matches = group.commands.filter(
          (command) =>
            matchesSearch(command.searchTexts ?? [command.label, command.detail], term) ||
            (command.shortcut !== undefined && `g${command.shortcut}` === shortcutTerm)
        );
        return {
          ...group,
          commands: (group.isSearchOnly ? matches.slice(0, SEARCH_RESULT_LIMIT) : matches).map(
            (command): IndexedCommand => ({...command, index, elementId: `command-palette-${index++}`})
          ),
        };
      })
      .filter((group) => group.commands.length > 0);
  });

  private readonly commands = computed(() => this.groups().flatMap((group) => group.commands));

  protected readonly activeIndex = linkedSignal({source: this.commands, computation: () => 0});

  constructor() {
    refreshInBackground(
      ...(this.applicationsLink ? [this.applicationsService.refresh()] : []),
      ...(this.customersLink ? [this.customerOrganizationsService.refresh()] : []),
      ...(this.deploymentsLink ? [this.deploymentTargetsService.refresh()] : [])
    );
  }

  protected onKeydown(event: KeyboardEvent) {
    const commands = this.commands();
    if ((event.key === 'ArrowDown' || event.key === 'ArrowUp') && commands.length > 0) {
      event.preventDefault();
      const step = event.key === 'ArrowDown' ? 1 : -1;
      const index = (this.activeIndex() + step + commands.length) % commands.length;
      this.activeIndex.set(index);
      document.getElementById(commands[index].elementId)?.scrollIntoView({block: 'nearest'});
    } else if (event.key === 'Enter' && commands[this.activeIndex()]) {
      event.preventDefault();
      this.run(commands[this.activeIndex()]);
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
