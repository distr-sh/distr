import {AsyncPipe, JsonPipe} from '@angular/common';
import {
  AfterViewInit,
  Component,
  computed,
  forwardRef,
  inject,
  Injector,
  input,
  OnDestroy,
  OnInit,
} from '@angular/core';
import {toObservable} from '@angular/core/rxjs-interop';
import {
  ControlValueAccessor,
  FormBuilder,
  NG_VALUE_ACCESSOR,
  NgControl,
  ReactiveFormsModule,
  TouchedChangeEvent,
  Validators,
} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {DeploymentRequest, DeploymentType, HelmOptions} from '@distr-sh/distr-sdk';
import {
  BehaviorSubject,
  catchError,
  combineLatest,
  debounceTime,
  distinctUntilChanged,
  filter,
  map,
  NEVER,
  of,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  takeUntil,
} from 'rxjs';
import {isArchived} from '../../../util/dates';
import {DURATION_REGEX, HELM_RELEASE_NAME_MAX_LENGTH, HELM_RELEASE_NAME_REGEX} from '../../../util/validation';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicensesService} from '../../services/licenses.service';

export type DeploymentFormValue = Partial<{
  deploymentId: string;
  applicationId: string;
  applicationVersionId: string;
  applicationLicenseId: string;
  valuesYaml: string;
  releaseName: string;
  envFileData: string;
  swarmMode: boolean;
  logsEnabled: boolean;
  forceRestart: boolean;
  ignoreRevisionSkew: boolean;
  helmOptions: Partial<HelmOptions>;
}>;

export function mapToDeploymentRequest(value: DeploymentFormValue, deploymentTargetId: string): DeploymentRequest {
  return {
    deploymentTargetId: deploymentTargetId,
    applicationVersionId: value.applicationVersionId!,
    applicationLicenseId: value.applicationLicenseId || undefined,
    deploymentId: value.deploymentId || undefined,
    releaseName: value.releaseName || undefined,
    valuesYaml: value.valuesYaml ? btoa(value.valuesYaml) : undefined,
    dockerType: value.swarmMode ? 'swarm' : 'compose',
    envFileData: value.envFileData ? btoa(value.envFileData) : undefined,
    logsEnabled: value.logsEnabled ?? false,
    forceRestart: value.forceRestart ?? false,
    ignoreRevisionSkew: value.ignoreRevisionSkew ?? false,
    helmOptions: value.helmOptions as HelmOptions | undefined,
  };
}

type DeploymentFormValueCallback = (v: DeploymentFormValue | undefined) => void;

@Component({
  selector: 'app-deployment-form',
  imports: [ReactiveFormsModule, AsyncPipe, EditorComponent, AutotrimDirective, RouterLink, JsonPipe],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => DeploymentFormComponent),
      multi: true,
    },
  ],
  templateUrl: './deployment-form.component.html',
})
export class DeploymentFormComponent implements OnInit, AfterViewInit, OnDestroy, ControlValueAccessor {
  disableApplicationSelect = input(false);
  deploymentType = input<DeploymentType>('docker');
  customerOrganizationId = input<string>();
  deploymentTargetName = input<string>('default');

  protected readonly secretsUrl = computed(() => {
    const customerOrgId = this.customerOrganizationId();
    return customerOrgId ? `/customers/${customerOrgId}/secrets` : '/secrets';
  });

  protected readonly featureFlags = inject(FeatureFlagService);
  protected readonly auth = inject(AuthService);
  private readonly applications = inject(ApplicationsService);
  private readonly licenses = inject(LicensesService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly injector = inject(Injector);

  protected readonly deployForm = this.fb.group({
    deploymentId: this.fb.control<string | undefined>(undefined),
    applicationId: this.fb.control('', Validators.required),
    applicationVersionId: this.fb.control('', Validators.required),
    applicationLicenseId: this.fb.control('', Validators.required),
    releaseName: this.fb.control('', [
      Validators.required,
      Validators.maxLength(HELM_RELEASE_NAME_MAX_LENGTH),
      Validators.pattern(HELM_RELEASE_NAME_REGEX),
    ]),
    valuesYaml: this.fb.control(''),
    envFileData: this.fb.control(''),
    swarmMode: this.fb.control(false),
    logsEnabled: this.fb.control(true),
    forceRestart: this.fb.control(false),
    ignoreRevisionSkew: this.fb.control(false),
    helmOptionsEnabled: this.fb.control(false),
    helmOptions: this.fb.group({
      timeout: this.fb.control('15m', [Validators.required, Validators.pattern(DURATION_REGEX)]),
      waitStrategy: this.fb.control('watcher', [Validators.required]),
      rollbackOnFailure: this.fb.control(true),
      cleanupOnFailure: this.fb.control(true),
    }),
  });
  protected readonly composeFile = this.fb.control({disabled: true, value: ''});

  private readonly deploymentId$ = this.deployForm.controls.deploymentId.valueChanges.pipe(
    startWith(this.deployForm.controls.deploymentId.value),
    distinctUntilChanged(),
    shareReplay(1)
  );

  private readonly applicationId$ = this.deployForm.controls.applicationId.valueChanges.pipe(
    startWith(this.deployForm.controls.applicationId.value),
    distinctUntilChanged(),
    shareReplay(1)
  );

  private readonly applicationVersionId$ = this.deployForm.controls.applicationVersionId.valueChanges.pipe(
    startWith(this.deployForm.controls.applicationVersionId.value),
    distinctUntilChanged(),
    shareReplay(1)
  );

  private readonly applicationLicenseId$ = this.deployForm.controls.applicationLicenseId.valueChanges.pipe(
    startWith(this.deployForm.controls.applicationLicenseId.value),
    distinctUntilChanged(),
    shareReplay(1)
  );

  private readonly deploymentType$ = toObservable(this.deploymentType);
  private readonly customerOrganizationId$ = toObservable(this.customerOrganizationId);

  protected readonly allLicenses$ = this.featureFlags.isLicensingEnabled$.pipe(
    switchMap((enabled) => (enabled ? this.licenses.list() : of([])))
  );

  protected readonly licenses$ = combineLatest([
    this.applicationId$,
    this.featureFlags.isLicensingEnabled$,
    this.customerOrganizationId$,
  ]).pipe(
    switchMap(([applicationId, isLicensingEnabled, customerOrgId]) =>
      isLicensingEnabled && applicationId && (this.auth.isCustomer() || customerOrgId)
        ? this.licenses
            .list(applicationId)
            .pipe(
              map((licenses) =>
                this.auth.isVendor() ? licenses.filter((l) => l.customerOrganizationId === customerOrgId) : licenses
              )
            )
        : of([])
    ),
    distinctUntilChanged(),
    shareReplay(1)
  );

  protected readonly licenseUpdateRequired$ = new BehaviorSubject<boolean>(false);

  /**
   * The license control is VISIBLE for users editing a customer managed deployment.
   */
  protected readonly licenseControlVisible$ = combineLatest([this.allLicenses$, this.customerOrganizationId$]).pipe(
    map(([licenses, customerOrgId]) => (this.auth.isCustomer() || !!customerOrgId) && licenses.length > 0),
    distinctUntilChanged(),
    shareReplay(1)
  );

  /**
   * The license control is ENABLED when deploying to a customer managed target and there is no deployment yet,
   * or the deployment was created without an initial license, and license management was later activated.
   * A vendor might be required to choose a license for a customer managed deployment target with no previous
   * deployment but they may only choose a license owned by the same customer.
   */
  private readonly licenseControlEnabled$ = combineLatest([
    this.licenseControlVisible$,
    this.deploymentId$,
    this.licenseUpdateRequired$,
  ]).pipe(
    map(([isVisible, deploymentId, licenseUpdateRequired]) => isVisible && (!deploymentId || licenseUpdateRequired)),
    distinctUntilChanged(),
    shareReplay(1)
  );

  protected readonly swarmModeVisible$ = toObservable(computed(() => this.deploymentType() === 'docker'));

  protected readonly applications$ = combineLatest([
    this.applications.list(),
    this.deploymentType$,
    this.customerOrganizationId$,
    this.allLicenses$,
  ]).pipe(
    map(([applications, applicationType, customerOrganizationId, licenses]) =>
      applications.filter(
        (application) =>
          application.type === applicationType &&
          (!customerOrganizationId ||
            licenses.length === 0 ||
            licenses.some(
              (license) =>
                license.applicationId === application.id && license.customerOrganizationId === customerOrganizationId
            ))
      )
    )
  );

  private selectedApplication$ = combineLatest([this.applicationId$, this.applications$]).pipe(
    map(([applicationId, applications]) => applications.find((application) => application.id === applicationId)),
    distinctUntilChanged((a, b) => a?.id === b?.id),
    shareReplay(1)
  );

  private readonly selectedLicense$ = combineLatest([this.applicationLicenseId$, this.licenses$]).pipe(
    map(([licenseId, licenses]) => licenses.find((license) => license.id === licenseId))
  );

  protected availableApplicationVersions$ = combineLatest([
    this.licenseControlVisible$,
    this.selectedLicense$,
    this.selectedApplication$,
    this.applicationVersionId$,
  ]).pipe(
    map(([shouldShowLicense, license, application, selectedApplicationVersionId]) => {
      let versions;

      if (shouldShowLicense) {
        // if the license has no version associations, assume that the application has all available versions
        versions = license?.versions?.length ? license.versions : (application?.versions ?? []);
      } else {
        versions = application?.versions ?? [];
      }

      return versions.filter((av) => {
        if (av.id === selectedApplicationVersionId) {
          return true;
        }
        return !isArchived(av);
      });
    }),
    shareReplay(1)
  );

  private readonly destroyed$ = new Subject<void>();

  private onChange?: DeploymentFormValueCallback;
  private onTouched?: DeploymentFormValueCallback;

  ngOnInit(): void {
    combineLatest([this.deployForm.valueChanges, this.deployForm.statusChanges])
      .pipe(takeUntil(this.destroyed$))
      .subscribe(([value, status]) => {
        const callbackArg = status === 'VALID' ? value : undefined;
        this.onChange?.(callbackArg);
        this.onTouched?.(callbackArg);
      });

    this.licenseControlEnabled$.pipe(takeUntil(this.destroyed$)).subscribe((licenseControlEnabled) => {
      if (licenseControlEnabled) {
        this.deployForm.controls.applicationLicenseId.enable();
      } else {
        this.deployForm.controls.applicationLicenseId.disable();
      }
    });

    this.deploymentType$.pipe(takeUntil(this.destroyed$)).subscribe((type) => {
      if (type) {
        if (type === 'kubernetes') {
          this.deployForm.controls.releaseName.enable();
          this.deployForm.controls.valuesYaml.enable();
          this.deployForm.controls.helmOptionsEnabled.enable();
          this.deployForm.controls.envFileData.disable();

          const targetName = this.deploymentTargetName();
          if (!this.deployForm.value.releaseName && targetName) {
            this.deployForm.patchValue({
              releaseName: targetName.trim().toLowerCase().replaceAll(/\W+/g, '-'),
            });
          }
        } else {
          this.deployForm.controls.envFileData.enable();
          this.deployForm.controls.releaseName.disable();
          this.deployForm.controls.valuesYaml.disable();
          this.deployForm.controls.helmOptionsEnabled.disable();
        }
      }
    });

    combineLatest([this.deploymentId$, this.deploymentType$]).subscribe(([deploymentId, deploymentType]) => {
      if (deploymentType === 'kubernetes' && deploymentId) {
        this.deployForm.controls.ignoreRevisionSkew.enable();
      } else {
        this.deployForm.controls.ignoreRevisionSkew.disable();
      }
    });

    this.deploymentId$.pipe(takeUntil(this.destroyed$)).subscribe((id) => {
      if (id) {
        this.deployForm.controls.applicationId.disable();
        this.deployForm.controls.swarmMode.disable();
        this.deployForm.controls.forceRestart.enable();
      } else {
        if (!this.disableApplicationSelect()) {
          this.deployForm.controls.applicationId.enable();
        }
        this.deployForm.controls.swarmMode.enable();
        this.deployForm.controls.forceRestart.disable();
      }
    });

    combineLatest([this.applicationId$, this.applicationVersionId$, this.deploymentId$, this.deploymentType$])
      .pipe(
        debounceTime(5),
        switchMap(([applicationId, versionId, deploymentId, type]) =>
          versionId && applicationId && !deploymentId
            ? this.applications.getTemplateFile(applicationId, versionId).pipe(
                catchError(() => NEVER),
                map((templateFile) => ({templateFile, type}))
              )
            : NEVER
        ),
        takeUntil(this.destroyed$)
      )
      .subscribe(({templateFile, type}) => {
        if (type === 'kubernetes') {
          this.deployForm.controls.valuesYaml.patchValue(templateFile ?? '');
        } else {
          this.deployForm.controls.envFileData.patchValue(templateFile ?? '');
        }
      });

    combineLatest([this.applicationId$, this.applicationVersionId$, this.deploymentType$])
      .pipe(
        debounceTime(5),
        switchMap(([applicationId, versionId, type]) =>
          versionId && applicationId && type === 'docker'
            ? this.applications.getComposeFile(applicationId, versionId).pipe(catchError(() => NEVER))
            : NEVER
        ),
        takeUntil(this.destroyed$)
      )
      .subscribe((composeFile) => {
        this.composeFile.patchValue(composeFile ?? '');
      });

    this.licenses$.pipe(takeUntil(this.destroyed$)).subscribe((licenses) => {
      // Only update the form control, if the previously selected license is no longer in the list
      if (
        licenses.length > 0 &&
        licenses[0].id &&
        licenses.every((l) => l.id !== this.deployForm.controls.applicationLicenseId.value)
      ) {
        this.licenseUpdateRequired$.next(true);
        this.deployForm.controls.applicationLicenseId.setValue(licenses[0].id);
      }
    });

    this.availableApplicationVersions$.pipe(takeUntil(this.destroyed$)).subscribe((versions) => {
      if (versions.length > 0) {
        this.deployForm.controls.applicationVersionId.enable();
        const version = versions[versions.length - 1];
        // Only update the form control, if the previously selected version is no longer in the list
        if (version.id && versions.every((version) => version.id !== this.deployForm.value.applicationVersionId)) {
          this.deployForm.controls.applicationVersionId.setValue(version.id);
        }
      } else {
        this.deployForm.controls.applicationVersionId.disable();
        // this.deployForm.controls.applicationVersionId.reset();
      }
    });

    this.deployForm.controls.helmOptionsEnabled.valueChanges.pipe(takeUntil(this.destroyed$)).subscribe((enabled) => {
      if (enabled) {
        this.deployForm.controls.helmOptions.enable();
      } else {
        this.deployForm.controls.helmOptions.disable();
      }
    });

    // This is needed because the first value could be missed otherwise
    // TODO: Find a better solution for this
    this.applicationLicenseId$.pipe(takeUntil(this.destroyed$)).subscribe();

    // Disable application selector if requested
    if (this.disableApplicationSelect()) {
      this.deployForm.controls.applicationId.disable();
    }
  }

  ngAfterViewInit(): void {
    // adapted from https://github.com/angular/angular/issues/45089
    this.injector
      .get(NgControl)
      .control!.events.pipe(
        takeUntil(this.destroyed$),
        filter((event) => event instanceof TouchedChangeEvent && event.touched)
      )
      .subscribe(() => this.deployForm.markAllAsTouched());
  }

  ngOnDestroy(): void {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  registerOnChange(fn: DeploymentFormValueCallback): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: DeploymentFormValueCallback): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    if (isDisabled) {
      console.warn('DeploymentFormComponent does not support setDisabledState');
    }
  }

  writeValue(obj: DeploymentFormValue | null | undefined): void {
    if (obj) {
      this.deployForm.patchValue({...obj, helmOptionsEnabled: !!obj.helmOptions});
    } else {
      this.deployForm.reset();
    }
  }
}
