import {CdkStep, CdkStepper, CdkStepperPrevious} from '@angular/cdk/stepper';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject, OnDestroy, OnInit, signal, viewChild} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {CustomerOrganization, OrganizationBranding} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck} from '@fortawesome/free-regular-svg-icons';
import {
  faArrowRight,
  faB,
  faBox,
  faBoxesStacked,
  faCheck,
  faDownload,
  faLightbulb,
  faPalette,
  faRightToBracket,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, lastValueFrom, Observable, Subject} from 'rxjs';
import {WEBSITE_URL} from '../../../constants';
import {getFormDisplayedError} from '../../../util/errors';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {ToastService} from '../../services/toast.service';
import {TutorialsService} from '../../services/tutorials.service';
import {UsersService} from '../../services/users.service';
import {TutorialProgress} from '../../types/tutorials';
import {TutorialStepperComponent} from '../stepper/tutorial-stepper.component';
import {getExistingTask, getLastExistingTask} from '../utils';

const defaultBrandingDescription = `# Welcome

In this Customer Portal you can manage your deployments.
`;

const tutorialId = 'branding';
const welcomeStep = 'welcome';
const welcomeTaskStart = 'start';
const brandingStep = 'branding';
const brandingTaskSet = 'set';
const customerStep = 'customer';
const customerTaskCreateCustomer = 'create_customer';
const customerTaskInvite = 'invite';
const customerTaskLogin = 'login';

@Component({
  selector: 'app-branding-tutorial',
  imports: [
    ReactiveFormsModule,
    CdkStep,
    TutorialStepperComponent,
    FaIconComponent,
    CdkStepperPrevious,
    AutotrimDirective,
  ],
  templateUrl: './branding-tutorial.component.html',
})
export class BrandingTutorialComponent implements OnInit, OnDestroy {
  loading = signal(true);
  private readonly destroyed$ = new Subject<void>();
  protected readonly websiteUrl = WEBSITE_URL;
  protected readonly faBox = faBox;
  protected readonly faDownload = faDownload;
  protected readonly faPalette = faPalette;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faB = faB;
  protected readonly faLightbulb = faLightbulb;

  private readonly stepper = viewChild.required<CdkStepper>('stepper');

  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  protected readonly toast = inject(ToastService);
  protected readonly brandingService = inject(OrganizationBrandingService);
  protected readonly usersService = inject(UsersService);
  protected readonly tutorialsService = inject(TutorialsService);
  protected readonly customerOrgService = inject(CustomerOrganizationsService);

  protected progress?: TutorialProgress;
  private organizationBranding?: OrganizationBranding;
  protected readonly welcomeFormGroup = new FormGroup({});
  protected readonly brandingFormGroup = new FormGroup({
    titleDone: new FormControl<boolean>(false),
    title: new FormControl<string>('', {nonNullable: true, validators: Validators.required}),
    descriptionDone: new FormControl<boolean>(false),
    description: new FormControl<string>('', {nonNullable: true, validators: Validators.required}),
  });
  protected readonly inviteFormGroup = new FormGroup({
    customerName: new FormControl<string>('', {
      nonNullable: true,
      validators: [Validators.required],
    }),
    customerCreateDone: new FormControl<boolean>(false, Validators.requiredTrue),
    customerEmail: new FormControl<string>('', {
      nonNullable: true,
      validators: [Validators.required, Validators.email],
    }),
    inviteDone: new FormControl<boolean>(false, Validators.requiredTrue),
    customerConfirmed: new FormControl<boolean>(false, Validators.requiredTrue),
  });
  protected emailUsername?: string;
  protected emailDomain?: string;

  private customerOrganization?: CustomerOrganization;

  async ngOnInit() {
    try {
      this.progress = await lastValueFrom(this.tutorialsService.get(tutorialId));
      if (this.progress.createdAt) {
        if (!this.progress.completedAt) {
          await this.continueFromWelcome();
          await this.continueFromBranding();
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
    // prepare branding step
    try {
      this.organizationBranding = await lastValueFrom(this.brandingService.get());
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        // it's a valid use case for an organization to have no branding (hence 404 is not shown in toast)
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }

    this.brandingFormGroup.patchValue({
      title: this.organizationBranding?.title,
      titleDone: !!this.organizationBranding?.title,
      description: this.organizationBranding?.description || defaultBrandingDescription,
      descriptionDone: !!this.organizationBranding?.description,
    });

    if (!this.progress) {
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
    } else {
      this.stepper().next();
    }
  }

  protected async continueFromBranding() {
    this.brandingFormGroup.markAllAsTouched();
    if (this.brandingFormGroup.valid) {
      if (this.brandingFormGroup.dirty) {
        this.loading.set(true);
        const formVal = this.brandingFormGroup.getRawValue();
        const formData = new FormData();
        formData.set('title', formVal.title);
        formData.set('description', formVal.description);

        const id = this.organizationBranding?.id;
        let req: Observable<OrganizationBranding>;
        if (id) {
          req = this.brandingService.update(formData);
        } else {
          req = this.brandingService.create(formData);
        }

        try {
          this.organizationBranding = await lastValueFrom(req);
          this.brandingFormGroup.markAsPristine();
          this.progress = await lastValueFrom(
            this.tutorialsService.save(tutorialId, {
              stepId: brandingStep,
              taskId: brandingTaskSet,
            })
          );
          this.toast.success('Branding options have been updated');
        } catch (e) {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return;
        } finally {
          this.loading.set(false);
        }
      }

      this.brandingFormGroup.controls.titleDone.patchValue(true);
      this.brandingFormGroup.controls.descriptionDone.patchValue(true);
      this.prepareCustomerStep();
      this.stepper().next();
    }
  }

  private prepareCustomerStep() {
    const customerOrganization = getLastExistingTask(this.progress, customerStep, customerTaskCreateCustomer);
    if (customerOrganization?.value) {
      this.customerOrganization = customerOrganization.value;
      this.inviteFormGroup.controls.customerName.setValue(this.customerOrganization!.name);
      this.inviteFormGroup.controls.customerCreateDone.setValue(true);
    }

    // prepare the email form
    const email = getLastExistingTask(this.progress, customerStep, customerTaskInvite);
    if (email?.value && typeof email?.value === 'string') {
      this.inviteFormGroup.controls.customerEmail.patchValue(email.value);
      this.inviteFormGroup.controls.inviteDone.patchValue(true);
    }

    const login = getExistingTask(this.progress, customerStep, customerTaskLogin);
    if (login) {
      this.inviteFormGroup.controls.customerConfirmed.patchValue(true);
    }

    this.inviteFormGroup.markAsPristine();

    const claims = this.auth.getClaims();
    if (claims?.email) {
      const parts = claims.email.split('@');
      this.emailUsername = parts[0];
      this.emailDomain = parts[1];
    }
  }

  protected async createCustomerOrganization() {
    if (this.inviteFormGroup.controls.customerName.valid && this.inviteFormGroup.controls.customerName.dirty) {
      this.loading.set(true);
      try {
        this.customerOrganization = await firstValueFrom(
          this.customerOrgService.createCustomerOrganization({
            name: this.inviteFormGroup.value.customerName!,
          })
        );
        this.inviteFormGroup.controls.customerCreateDone.setValue(true);
        this.inviteFormGroup.markAsPristine();
        this.toast.success('Customer created');
        this.progress = await lastValueFrom(
          this.tutorialsService.save(tutorialId, {
            stepId: customerStep,
            taskId: customerTaskCreateCustomer,
            value: this.customerOrganization,
          })
        );
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.loading.set(false);
      }
    }
  }

  protected async sendInviteMail() {
    this.inviteFormGroup.markAllAsTouched();
    if (this.inviteFormGroup.controls.customerEmail.valid && this.inviteFormGroup.controls.customerEmail.dirty) {
      this.loading.set(true);
      try {
        const email = this.inviteFormGroup.value.customerEmail!;
        await lastValueFrom(
          this.usersService.addUser({
            email,
            customerOrganizationId: this.customerOrganization?.id,
            userRole: 'admin',
          })
        );
        this.inviteFormGroup.controls.inviteDone.patchValue(true);
        this.inviteFormGroup.markAsPristine();
        this.toast.success('Invite email has been sent');
        this.progress = await lastValueFrom(
          this.tutorialsService.save(tutorialId, {
            stepId: customerStep,
            taskId: customerTaskInvite,
            value: email,
          })
        );
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.loading.set(false);
      }
    }
  }

  protected async completeAndExit() {
    this.inviteFormGroup.markAllAsTouched();
    if (this.inviteFormGroup.valid && this.inviteFormGroup.dirty) {
      this.loading.set(true);
      this.progress = await lastValueFrom(
        this.tutorialsService.save(tutorialId, {
          stepId: customerStep,
          taskId: customerTaskLogin,
          markCompleted: true,
        })
      );
      this.stepper().selected!.completed = true;
      this.loading.set(false);
      this.toast.success('Congrats on finishing the tutorial! Good Job!');
      this.navigateToOverviewPage();
    } else if (this.progress?.completedAt) {
      this.navigateToOverviewPage();
    }
  }

  protected navigateToOverviewPage() {
    this.router.navigate(['tutorials']);
  }

  protected readonly faArrowRight = faArrowRight;
  protected readonly faRightToBracket = faRightToBracket;
  protected readonly faCheck = faCheck;
  protected readonly faCircleCheck = faCircleCheck;

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }
}
