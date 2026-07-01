import {CdkStep, CdkStepper, CdkStepperPrevious} from '@angular/cdk/stepper';
import {ChangeDetectionStrategy, Component, inject, signal, viewChild} from '@angular/core';
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
import {UsersService} from '../../services/users.service';
import {TutorialStepperComponent} from '../stepper/tutorial-stepper.component';

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
export class UsersTutorialComponent {
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

  protected readonly welcomeFormGroup = new FormGroup({});
  protected readonly inviteFormGroup = new FormGroup({
    users: new FormArray([userFormGroup()]),
  });

  get users(): FormArray {
    return this.inviteFormGroup.controls.users;
  }

  protected continueFromWelcome() {
    this.stepper().next();
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

    this.loading.set(true);
    const invited: FormGroup[] = [];
    try {
      for (const group of pending) {
        const email = group.get('email')!.value as string;
        const name = (group.get('name')!.value as string) || undefined;
        const userRole = group.get('userRole')!.value as UserRole;
        await lastValueFrom(this.usersService.addUser({email, name, userRole}));
        invited.push(group);
      }

      this.toast.success('Your teammates have been invited. Good job!');
      this.navigateToOverviewPage();
    } catch (e) {
      // drop already-invited rows so a retry does not invite them twice
      for (const group of invited) {
        const idx = this.users.controls.indexOf(group);
        if (idx >= 0) {
          this.users.removeAt(idx);
        }
      }
      if (this.users.length === 0) {
        this.addUser();
      }
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }
  }

  protected navigateToOverviewPage() {
    this.router.navigate(['tutorials']);
  }
}
