import {NgTemplateOutlet} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircle} from '@fortawesome/free-regular-svg-icons';
import {faCircleCheck, faListCheck} from '@fortawesome/free-solid-svg-icons';
import {CustomerHomeChecklist} from '../../types/customer-home';
import {CustomerHomePrimarySection} from './customer-home-deployments.component';

interface ChecklistItem {
  label: string;
  hint: string;
  done: boolean;
  link?: string;
}

@Component({
  selector: 'app-customer-home-checklist',
  templateUrl: './customer-home-checklist.component.html',
  imports: [NgTemplateOutlet, RouterLink, FaIconComponent],
})
export class CustomerHomeChecklistComponent {
  public readonly checklist = input.required<CustomerHomeChecklist>();
  public readonly section = input.required<CustomerHomePrimarySection>();
  public readonly canInvite = input.required<boolean>();

  protected readonly faCircle = faCircle;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faListCheck = faListCheck;

  protected readonly items = computed<ChecklistItem[]>(() => {
    const checklist = this.checklist();
    const items: ChecklistItem[] = [
      {
        label: 'Complete your profile',
        hint: 'Upload a profile picture or enable multi-factor authentication.',
        done: checklist.profileCompleted,
        link: '/settings/profile/general',
      },
      {
        label: 'Invite team members',
        hint: 'Give your colleagues access.',
        done: checklist.teamMembersInvited,
        link: this.canInvite() ? '/users' : undefined,
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
    return items.sort((a, b) => Number(b.done) - Number(a.done));
  });

  protected readonly doneCount = computed(() => this.items().filter((item) => item.done).length);
}
