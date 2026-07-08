import {CdkStep, CdkStepper, CdkStepperPrevious} from '@angular/cdk/stepper';
import {HttpErrorResponse} from '@angular/common/http';
import {ChangeDetectionStrategy, Component, inject, OnInit, signal, viewChild} from '@angular/core';
import {FormArray, FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck} from '@fortawesome/free-regular-svg-icons';
import {faArrowRight, faCheck, faLightbulb, faPlus, faUserGroup, faXmark} from '@fortawesome/free-solid-svg-icons';
import {lastValueFrom} from 'rxjs';
import {WEBSITE_URL} from '../../../constants';
import {getFormDisplayedError} from '../../../util/errors';
import {UserRoleSelectComponent} from '../../components/user-role-select.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {PlaceholderDirective} from '../../directives/placeholder.directive';
import {ToastService} from '../../services/toast.service';
import {TutorialsService} from '../../services/tutorials.service';
import {UsersService} from '../../services/users.service';
import {TutorialProgress} from '../../types/tutorials';
import {TutorialStepperComponent} from '../stepper/tutorial-stepper.component';

const tutorialId = 'users';
const welcomeStep = 'welcome';
const welcomeTaskStart = 'start';
const inviteStep = 'invite';
const inviteTaskSend = 'send';

function userFormGroup() {
  return new FormGroup({
    name: new FormControl<string>('', {nonNullable: true}),
    email: new FormControl<string>('', {nonNullable: true, validators: [Validators.required, Validators.email]}),
    userRole: new FormControl<UserRole>('read_write', {nonNullable: true, validators: [Validators.required]}),
  });
}

@Component({
  selector: 'app-users-tutorial',
  imports: [
    ReactiveFormsModule,
    CdkStep,
    TutorialStepperComponent,
    FaIconComponent,
    CdkStepperPrevious,
    AutotrimDirective,
    PlaceholderDirective,
    UserRoleSelectComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './users-tutorial.component.html',
})
export class UsersTutorialComponent implements OnInit {
  loading = signal(false);
  protected readonly rbacDocsUrl = `${WEBSITE_URL}/docs/platform/rbac/`;
  protected readonly faArrowRight = faArrowRight;
  protected readonly faCheck = faCheck;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faUserGroup = faUserGroup;

  private readonly stepper = viewChild.required<CdkStepper>('stepper');

  private readonly router = inject(Router);
  protected readonly toast = inject(ToastService);
  protected readonly usersService = inject(UsersService);
  protected readonly tutorialsService = inject(TutorialsService);

  protected progress?: TutorialProgress;
  protected readonly welcomeFormGroup = new FormGroup({});
  protected readonly inviteFormGroup = new FormGroup({
    users: new FormArray([userFormGroup()]),
  });

  get users(): FormArray {
    return this.inviteFormGroup.controls.users;
  }

  async ngOnInit() {
    try {
      this.progress = await lastValueFrom(this.tutorialsService.get(tutorialId));
      if (this.progress.createdAt) {
        if (!this.progress.completedAt) {
          this.stepper().next();
        } else {
          this.stepper().steps.forEach((s) => (s.completed = true));
        }
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        // it's a valid use case for a tutorial progress not to exist yet
        this.toast.error(msg);
      }
    }
  }

  protected async continueFromWelcome() {
    if (this.progress) {
      this.stepper().next();
      return;
    }
    this.loading.set(true);
    try {
      this.progress = await lastValueFrom(
        this.tutorialsService.save(tutorialId, {
          stepId: welcomeStep,
          taskId: welcomeTaskStart,
        })
      );
      this.stepper().next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }
  }

  protected addUser() {
    this.users.push(userFormGroup());
  }

  protected removeUser(index: number) {
    if (this.users.length > 1) {
      this.users.removeAt(index);
    } else {
      this.users.setControl(index, userFormGroup());
    }
  }

  protected async inviteAndComplete() {
    this.inviteFormGroup.markAllAsTouched();

    const pending = (this.users.controls as FormGroup[]).filter(
      (c) => ((c.get('email')?.value as string) ?? '').length > 0
    );

    if (pending.some((c) => c.invalid)) {
      return;
    }
    if (pending.length === 0) {
      this.toast.error('Please add at least one teammate to invite.');
      return;
    }

    const seenEmails = new Set<string>();
    const toInvite = pending.filter((group) => {
      const email = (group.get('email')!.value as string).trim().toLowerCase();
      if (seenEmails.has(email)) {
        return false;
      }
      seenEmails.add(email);
      return true;
    });

    this.loading.set(true);
    const invitedEmails: string[] = [];
    const failedEmails: string[] = [];
    let firstError: unknown;
    try {
      for (const group of toInvite) {
        const email = group.get('email')!.value as string;
        const name = (group.get('name')!.value as string) || undefined;
        const userRole = group.get('userRole')!.value as UserRole;
        try {
          await lastValueFrom(this.usersService.addUser({email, name, userRole}));
          invitedEmails.push(email);
          const idx = this.users.controls.indexOf(group);
          if (idx >= 0) {
            this.users.removeAt(idx);
          }
        } catch (e) {
          failedEmails.push(email);
          firstError ??= e;
        }
      }

      if (invitedEmails.length === 0) {
        const msg = getFormDisplayedError(firstError);
        if (msg) {
          this.toast.error(msg);
        }
        return;
      }

      try {
        this.progress = await lastValueFrom(
          this.tutorialsService.save(tutorialId, {
            stepId: inviteStep,
            taskId: inviteTaskSend,
            value: invitedEmails,
            markCompleted: true,
          })
        );
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      }

      if (failedEmails.length > 0) {
        this.toast.error(`Some invites could not be sent: ${failedEmails.join(', ')}`);
      } else {
        this.toast.success('Your teammates have been invited. Good job!');
      }
      this.navigateToOverviewPage();
    } finally {
      if (this.users.length === 0) {
        this.addUser();
      }
      this.loading.set(false);
    }
  }

  protected navigateToOverviewPage() {
    this.router.navigate(['tutorials']);
  }
}
