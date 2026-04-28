import {Platform} from '@angular/cdk/platform';
import {CdkStep, CdkStepper, CdkStepperPrevious} from '@angular/cdk/stepper';
import {HttpErrorResponse} from '@angular/common/http';
import {AfterViewInit, Component, inject, OnDestroy, OnInit, signal, viewChild} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faClipboard} from '@fortawesome/free-regular-svg-icons';
import {
  faArrowRight,
  faBox,
  faBoxesStacked,
  faCheck,
  faClipboardCheck,
  faLightbulb,
  faPalette,
  faWarning,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, lastValueFrom, Subject, switchMap, takeUntil, tap} from 'rxjs';
import {GITHUB_URL, WEBSITE_URL} from '../../../constants';
import {getFormDisplayedError} from '../../../util/errors';
import {ClipComponent} from '../../components/clip.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ApplicationsService} from '../../services/applications.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {ToastService} from '../../services/toast.service';
import {TutorialsService} from '../../services/tutorials.service';
import {TutorialProgress} from '../../types/tutorials';
import {TutorialStepperComponent} from '../stepper/tutorial-stepper.component';
import {getExistingTask, getLastExistingTask} from '../utils';

const tutorialId = 'agents';
const welcomeStep = 'welcome';
const welcomeTaskStart = 'start';
const deployStep = 'deploy';
const deployStepTaskDeploy = 'deploy';
const deployStepTaskVerify = 'verify';
const deployStepTaskDockerPs = 'docker-ps';
const deployStepTaskOpen = 'open';
const releaseStep = 'release';
const releaseStepTaskFork = 'fork';
const releaseStepTaskRelease = 'release';

@Component({
  selector: 'app-agents-tutorial',
  imports: [
    ReactiveFormsModule,
    CdkStep,
    TutorialStepperComponent,
    FaIconComponent,
    CdkStepperPrevious,
    ClipComponent,
    AutotrimDirective,
    RouterLink,
  ],
  templateUrl: './agents-tutorial.component.html',
})
export class AgentsTutorialComponent implements OnInit, AfterViewInit, OnDestroy {
  loading = signal(true);
  private readonly destroyed$ = new Subject<void>();
  protected readonly githubUrl = GITHUB_URL;
  protected readonly websiteUrl = WEBSITE_URL;
  protected readonly faBox = faBox;
  protected readonly faPalette = faPalette;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faLightbulb = faLightbulb;
  private readonly stepper = viewChild.required<CdkStepper>('stepper');
  private readonly router = inject(Router);
  protected readonly toast = inject(ToastService);
  protected readonly applicationsService = inject(ApplicationsService);
  protected readonly tutorialsService = inject(TutorialsService);
  private readonly deploymentTargetService = inject(DeploymentTargetsService);
  protected progress?: TutorialProgress;
  protected readonly welcomeFormGroup = new FormGroup({});
  protected readonly deployFormGroup = new FormGroup({
    deployDone: new FormControl<boolean>(false, Validators.requiredTrue),
    verifyDone: new FormControl<boolean>(false, Validators.requiredTrue),
    dockerPsDone: new FormControl<boolean>(false, Validators.requiredTrue),
    openDone: new FormControl<boolean>(false, Validators.requiredTrue),
  });
  protected readonly releaseFormGroup = new FormGroup({
    forkDone: new FormControl<boolean>(false, Validators.requiredTrue),
    forkedRepo: new FormControl<string>('', {nonNullable: true}),
    releaseDone: new FormControl<boolean>(false, Validators.requiredTrue),
  });
  connectCommand?: string;
  targetId?: string;
  targetSecret?: string;
  commandCopied = false;
  protected readonly route = inject(ActivatedRoute);
  private platform: Platform = inject(Platform);
  protected showMacOsWarning = this.platform.IOS || this.platform.SAFARI || /Mac/i.test(navigator.userAgent);

  ngOnInit() {
    this.registerTaskToggle(this.deployFormGroup.controls.deployDone, deployStep, deployStepTaskDeploy);
    this.registerTaskToggle(this.deployFormGroup.controls.verifyDone, deployStep, deployStepTaskVerify);
    this.registerTaskToggle(this.deployFormGroup.controls.dockerPsDone, deployStep, deployStepTaskDockerPs);
    this.registerTaskToggle(this.deployFormGroup.controls.openDone, deployStep, deployStepTaskOpen);
    this.registerTaskToggle(this.releaseFormGroup.controls.forkDone, releaseStep, releaseStepTaskFork);
  }

  private registerTaskToggle(ctrl: FormControl<boolean | null>, stepId: string, taskId: string) {
    ctrl.valueChanges
      .pipe(
        takeUntil(this.destroyed$),
        switchMap((done) =>
          this.tutorialsService.saveDoneIfNotYetDone(this.progress, done ?? false, tutorialId, stepId, taskId)
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
          await this.continueFromDeploy();
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
    } finally {
      this.loading.set(false);
    }
  }

  protected async continueFromWelcome() {
    if (!this.progress) {
      this.loading.set(true);
      try {
        await this.saveWelcomeStep();
        this.stepper().next();
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
          return;
        }
      } finally {
        this.loading.set(false);
      }
    } else {
      this.stepper().next();
    }

    this.prepareDeployStep();
  }

  private async saveWelcomeStep() {
    this.progress = await lastValueFrom(
      this.tutorialsService.save(tutorialId, {
        stepId: welcomeStep,
        taskId: welcomeTaskStart,
      })
    );
    this.applicationsService.refresh();
  }

  private prepareDeployStep() {
    const deployed = getExistingTask(this.progress, deployStep, deployStepTaskDeploy);
    const verified = getExistingTask(this.progress, deployStep, deployStepTaskVerify);
    const dockerPs = getExistingTask(this.progress, deployStep, deployStepTaskDockerPs);
    const opened = getExistingTask(this.progress, deployStep, deployStepTaskOpen);
    this.deployFormGroup.patchValue({
      deployDone: !!deployed,
      verifyDone: !!verified,
      dockerPsDone: !!dockerPs,
      openDone: !!opened,
    });
  }

  protected async requestAccess() {
    const startTask = getLastExistingTask(this.progress, welcomeStep, welcomeTaskStart);
    if (startTask?.value && 'deploymentTargetId' in startTask.value) {
      try {
        this.loading.set(true);
        const resp = await firstValueFrom(
          this.deploymentTargetService.requestAccess(startTask.value['deploymentTargetId'])
        );
        this.connectCommand = resp.connectCommand;
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (e instanceof HttpErrorResponse && e.status === 404) {
          // old deployment might have been deleted already
          await this.saveWelcomeStep();
          await this.requestAccess();
        } else if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.loading.set(false);
      }
    }
  }

  async copyConnectCommand() {
    if (this.connectCommand) {
      await navigator.clipboard.writeText(this.connectCommand);
    }
    this.commandCopied = true;
    setTimeout(() => (this.commandCopied = false), 2000);
  }

  protected async continueFromDeploy() {
    this.deployFormGroup.markAllAsTouched();
    if (this.deployFormGroup.valid) {
      this.prepareReleaseStep();
      this.stepper().next();
    }
  }

  private prepareReleaseStep() {
    const fork = getExistingTask(this.progress, releaseStep, releaseStepTaskFork);
    const release = getLastExistingTask(this.progress, releaseStep, releaseStepTaskRelease);
    this.releaseFormGroup.patchValue({
      forkDone: !!fork,
      forkedRepo: release?.value,
      releaseDone: !!release,
    });
  }

  protected async completeAndExit() {
    this.releaseFormGroup.markAllAsTouched();
    if (this.releaseFormGroup.valid && this.releaseFormGroup.dirty) {
      this.loading.set(true);
      this.progress = await lastValueFrom(
        this.tutorialsService.save(tutorialId, {
          stepId: releaseStep,
          taskId: releaseStepTaskRelease,
          value: this.releaseFormGroup.value.forkedRepo,
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
  protected readonly faCheck = faCheck;

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faWarning = faWarning;
}
