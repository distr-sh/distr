import {CdkStep, CdkStepper, CdkStepperPrevious} from '@angular/cdk/stepper';
import {HttpErrorResponse} from '@angular/common/http';
import {AfterViewInit, Component, inject, OnDestroy, OnInit, signal, ViewChild} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {AccessTokenWithKey} from '@distr-sh/distr-sdk';
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
import {combineLatest, firstValueFrom, lastValueFrom, Subject, switchMap, takeUntil, tap} from 'rxjs';
import {fromPromise} from 'rxjs/internal/observable/innerFrom';
import {getRemoteEnvironment} from '../../../env/remote';
import {getFormDisplayedError} from '../../../util/errors';
import {slugMaxLength, slugPattern} from '../../../util/slug';
import {ClipComponent} from '../../components/clip.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AccessTokensService} from '../../services/access-tokens.service';
import {OrganizationService} from '../../services/organization.service';
import {ToastService} from '../../services/toast.service';
import {TutorialsService} from '../../services/tutorials.service';
import {Organization} from '../../types/organization';
import {TutorialProgress} from '../../types/tutorials';
import {TutorialStepperComponent} from '../stepper/tutorial-stepper.component';
import {getExistingTask} from '../utils';

const tutorialId = 'registry';
const welcomeStep = 'welcome';
const welcomeTaskStart = 'start';
const prepareStep = 'prepare';
const prepareStepTaskSetSlug = 'set-slug';
const loginStep = 'login';
const loginStepTaskCreateToken = 'create-token';
const loginStepTaskLogin = 'login';
const usageStep = 'usage';
const usageStepTaskPull = 'pull';
const usageStepTaskTag = 'tag';
const usageStepTaskPush = 'push';
const usageStepTaskExplore = 'explore';

@Component({
  selector: 'app-registry-tutorial',
  imports: [
    ReactiveFormsModule,
    CdkStep,
    TutorialStepperComponent,
    FaIconComponent,
    CdkStepperPrevious,
    AutotrimDirective,
    ClipComponent,
    RouterLink,
  ],
  templateUrl: './registry-tutorial.component.html',
})
export class RegistryTutorialComponent implements OnInit, AfterViewInit, OnDestroy {
  loading = signal(true);
  private readonly destroyed$ = new Subject<void>();
  protected readonly faBox = faBox;
  protected readonly faDownload = faDownload;
  protected readonly faPalette = faPalette;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faB = faB;
  protected readonly faLightbulb = faLightbulb;
  @ViewChild('stepper', {static: false}) private stepper?: CdkStepper;
  private readonly router = inject(Router);
  protected readonly toast = inject(ToastService);
  protected readonly organizationService = inject(OrganizationService);
  protected readonly tutorialsService = inject(TutorialsService);
  protected readonly tokenService = inject(AccessTokensService);
  protected createdToken?: AccessTokenWithKey;
  protected progress?: TutorialProgress;
  private organization?: Organization;
  protected readonly welcomeFormGroup = new FormGroup({});
  protected readonly prepareFormGroup = new FormGroup({
    slugDone: new FormControl<boolean>(false),
    slug: new FormControl<string>('', {
      nonNullable: true,
      validators: [Validators.required, Validators.pattern(slugPattern), Validators.maxLength(slugMaxLength)],
    }),
  });
  protected readonly loginFormGroup = new FormGroup({
    tokenDone: new FormControl<boolean>(false, Validators.requiredTrue),
    loginDone: new FormControl<boolean>(false, Validators.requiredTrue),
  });
  protected readonly usageFormGroup = new FormGroup({
    pullDone: new FormControl<boolean>(false, Validators.requiredTrue),
    tagDone: new FormControl<boolean>(false, Validators.requiredTrue),
    pushDone: new FormControl<boolean>(false, Validators.requiredTrue),
    exploreDone: new FormControl<boolean>(false, Validators.requiredTrue),
  });
  protected slug?: string;
  protected host?: string;
  protected helloDistrTag = '0.1.10';
  protected readonly route = inject(ActivatedRoute);

  ngOnInit() {
    this.organizationService
      .get()
      .pipe(takeUntil(this.destroyed$))
      .subscribe((o) => (this.slug = o.slug));
    combineLatest([fromPromise(getRemoteEnvironment()), this.organizationService.get()])
      .pipe(takeUntil(this.destroyed$))
      .subscribe(([env, org]) => (this.host = org.registryDomain ?? env.registryHost));

    this.usageFormGroup.controls.pullDone.valueChanges
      .pipe(
        takeUntil(this.destroyed$),
        switchMap((done) =>
          this.tutorialsService.saveDoneIfNotYetDone(
            this.progress,
            done ?? false,
            tutorialId,
            usageStep,
            usageStepTaskPull
          )
        ),
        tap((updated) => (this.progress = updated))
      )
      .subscribe();
    this.usageFormGroup.controls.tagDone.valueChanges
      .pipe(
        takeUntil(this.destroyed$),
        switchMap((done) =>
          this.tutorialsService.saveDoneIfNotYetDone(
            this.progress,
            done ?? false,
            tutorialId,
            usageStep,
            usageStepTaskTag
          )
        ),
        tap((updated) => (this.progress = updated))
      )
      .subscribe();
    this.usageFormGroup.controls.pushDone.valueChanges
      .pipe(
        takeUntil(this.destroyed$),
        switchMap((done) =>
          this.tutorialsService.saveDoneIfNotYetDone(
            this.progress,
            done ?? false,
            tutorialId,
            usageStep,
            usageStepTaskPush
          )
        ),
        tap((updated) => (this.progress = updated))
      )
      .subscribe();
  }

  async ngAfterViewInit() {
    try {
      this.progress = await lastValueFrom(this.tutorialsService.get(tutorialId));
      if (this.progress.createdAt) {
        if (!this.progress.completedAt) {
          await this.continueFromWelcome();
          await this.continueFromPrepare();
          await this.continueFromLogin();
        } else {
          this.stepper?.steps.forEach((s) => (s.completed = true));
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
    try {
      this.loading.set(true);
      this.organization = await firstValueFrom(this.organizationService.get());
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      return;
    } finally {
      this.loading.set(false);
    }

    this.prepareFormGroup.patchValue({
      slug: this.organization?.slug,
      slugDone: !!this.organization?.slug,
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
        this.stepper?.next();
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.loading.set(false);
      }
    } else {
      this.stepper?.next();
    }
  }

  protected async continueFromPrepare() {
    this.prepareFormGroup.markAllAsTouched();
    if (this.prepareFormGroup.valid) {
      if (this.prepareFormGroup.dirty) {
        this.loading.set(true);
        const formVal = this.prepareFormGroup.getRawValue();
        try {
          this.organization = await lastValueFrom(
            this.organizationService.update({
              ...this.organization!,
              artifactVersionMutable: this.organization?.features.includes('artifact_version_mutable') ?? false,
              prePostScriptsEnabled: this.organization?.features.includes('pre_post_scripts') ?? false,
              slug: formVal.slug!,
            })
          );
          this.prepareFormGroup.markAsPristine();
          this.progress = await lastValueFrom(
            this.tutorialsService.save(tutorialId, {
              stepId: prepareStep,
              taskId: prepareStepTaskSetSlug,
            })
          );
          this.toast.success('Organization has been updated');
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

      this.prepareFormGroup.controls.slugDone.patchValue(true);
      this.prepareLoginStep();
      this.stepper?.next();
    }
  }

  private prepareLoginStep() {
    const token = getExistingTask(this.progress, loginStep, loginStepTaskCreateToken);
    const login = getExistingTask(this.progress, loginStep, loginStepTaskLogin);

    this.loginFormGroup.patchValue({
      tokenDone: !!token,
      loginDone: !!login,
    });
  }

  protected async createToken() {
    this.loading.set(true);
    try {
      this.createdToken = await firstValueFrom(
        this.tokenService.create({
          label: 'tutorial-token',
          // TODO expiry?
        })
      );
      this.loginFormGroup.controls.tokenDone.patchValue(true);
      this.toast.success('Personal Access Token created successfully');
      this.progress = await lastValueFrom(
        this.tutorialsService.save(tutorialId, {
          stepId: loginStep,
          taskId: loginStepTaskCreateToken,
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

  protected async continueFromLogin() {
    this.loginFormGroup.markAllAsTouched();
    if (this.loginFormGroup.valid) {
      if (this.loginFormGroup.dirty) {
        this.loading.set(true);
        try {
          this.loginFormGroup.markAsPristine();
          this.progress = await lastValueFrom(
            this.tutorialsService.save(tutorialId, {
              stepId: loginStep,
              taskId: loginStepTaskLogin,
            })
          );
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

      this.prepareUsageStep();
      this.stepper?.next();
    }
  }

  private prepareUsageStep() {
    const pull = getExistingTask(this.progress, usageStep, usageStepTaskPull);
    const tag = getExistingTask(this.progress, usageStep, usageStepTaskTag);
    const push = getExistingTask(this.progress, usageStep, usageStepTaskPush);
    const explore = getExistingTask(this.progress, usageStep, usageStepTaskExplore);

    this.usageFormGroup.patchValue({
      pullDone: !!pull,
      tagDone: !!tag,
      pushDone: !!push,
      exploreDone: !!explore,
    });
    this.usageFormGroup.markAsPristine();
  }

  protected async completeAndExit() {
    this.usageFormGroup.markAllAsTouched();
    if (this.usageFormGroup.valid && this.usageFormGroup.dirty) {
      this.loading.set(true);
      this.progress = await lastValueFrom(
        this.tutorialsService.save(tutorialId, {
          stepId: usageStep,
          taskId: usageStepTaskExplore,
          markCompleted: true,
        })
      );
      this.stepper!.selected!.completed = true;
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
