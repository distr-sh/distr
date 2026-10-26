import {Component, computed, inject, input, output} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faListCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {AuthService} from '../../services/auth.service';
import {CustomerHomeChecklist} from '../../types/customer-home';
import {CustomerHomeChecklistItem, CustomerHomeChecklistItemComponent} from './customer-home-checklist-item.component';
import {CustomerHomePrimarySection} from './customer-home-deployments.component';

@Component({
  selector: 'app-customer-home-checklist',
  templateUrl: './customer-home-checklist.component.html',
  imports: [RouterLink, FaIconComponent, CardHeadingComponent, CustomerHomeChecklistItemComponent],
})
export class CustomerHomeChecklistComponent {
  private readonly canInvite = inject(AuthService).hasRole('admin');

  public readonly checklist = input.required<CustomerHomeChecklist>();
  public readonly section = input.required<CustomerHomePrimarySection>();
  public readonly dismissed = output<void>();

  protected readonly faListCheck = faListCheck;
  protected readonly faXmark = faXmark;

  protected readonly items = computed<CustomerHomeChecklistItem[]>(() => {
    const checklist = this.checklist();
    const items: CustomerHomeChecklistItem[] = [
      {
        label: 'Complete your profile',
        hint: 'Upload a profile picture or enable multi-factor authentication.',
        done: checklist.profileCompleted,
        link: '/settings/profile/general',
      },
      {
        label: 'Invite team members',
        hint: this.canInvite ? 'Give your colleagues access.' : 'Ask an admin to give your colleagues access.',
        done: checklist.teamMembersInvited,
        link: this.canInvite ? '/users' : undefined,
      },
    ];
    if (this.section() === 'deployments') {
      items.push(
        {
          label: 'Create a deployment',
          hint: 'Install an application in your environment.',
          done: checklist.deploymentCreated,
          link: '/deployments',
        },
        {
          label: 'Create a secret',
          hint: 'Store credentials your deployments use.',
          done: checklist.secretCreated,
          link: '/secrets',
        }
      );
    }
    if (this.section() === 'artifacts') {
      items.push(
        {
          label: 'Create a personal access token',
          hint: 'Authenticate against the registry.',
          done: checklist.accessTokenCreated,
          link: '/settings/access-tokens',
        },
        {
          label: 'Download an artifact',
          hint: 'Pull your first artifact from the registry.',
          done: checklist.artifactPulled,
          link: '/artifacts',
        }
      );
    }
    if (checklist.updateNotificationsConfigured !== undefined) {
      items.push({
        label: 'Set up update notifications',
        hint: 'Get notified about new versions.',
        done: checklist.updateNotificationsConfigured,
        link: '/notifications/updates',
      });
    }
    return items.sort((a, b) => Number(b.done) - Number(a.done));
  });

  protected readonly doneCount = computed(() => this.items().filter((item) => item.done).length);
}
