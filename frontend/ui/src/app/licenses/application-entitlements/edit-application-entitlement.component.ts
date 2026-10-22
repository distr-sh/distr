import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {AsyncPipe, NgPlural, NgPluralCase} from '@angular/common';
import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  forwardRef,
  inject,
  Injector,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {
  ControlValueAccessor,
  FormArray,
  FormBuilder,
  NG_VALIDATORS,
  NG_VALUE_ACCESSOR,
  NgControl,
  ReactiveFormsModule,
  TouchedChangeEvent,
  ValidationErrors,
  Validator,
  ValidatorFn,
  Validators,
} from '@angular/forms';
import {Application, ApplicationVersion} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {filter, firstValueFrom, switchMap} from 'rxjs';
import {WEBSITE_URL} from '../../../constants';
import {isArchived} from '../../../util/dates';
import {sortApplicationVersions} from '../../../util/versions';
import {AlertComponent} from '../../components/alert/alert.component';
import {ExpiresAtPickerComponent} from '../../components/expires-at-picker/expires-at-picker.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {ArtifactEntitlement} from '../../types/artifact-entitlement';

type VersionScope = 'all' | 'specific';

// An entitlement without versions grants all of them, so "Specific versions" needs at least one.
const specificVersionsValidator: ValidatorFn = (control) => {
  const versionScope = control.get('versionScope')?.value as VersionScope;
  const selected = [
    ...(control.get('activeVersions') as FormArray).getRawValue(),
    ...(control.get('archivedVersions') as FormArray).getRawValue(),
  ].some((v) => !!v);
  return versionScope === 'all' || selected ? null : {versionsRequired: true};
};

const noSurroundingWhitespaceValidator: ValidatorFn = (control) => {
  const value = control.value as string;
  return value === value.trim() ? null : {surroundingWhitespace: true};
};

@Component({
  selector: 'app-edit-application-entitlement',
  templateUrl: './edit-application-entitlement.component.html',
  imports: [
    AsyncPipe,
    NgPlural,
    NgPluralCase,
    AutotrimDirective,
    ExpiresAtPickerComponent,
    ReactiveFormsModule,
    CdkOverlayOrigin,
    CdkConnectedOverlay,
    FaIconComponent,
    AlertComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => EditApplicationEntitlementComponent),
      multi: true,
    },
    {
      provide: NG_VALIDATORS,
      useExisting: forwardRef(() => EditApplicationEntitlementComponent),
      multi: true,
    },
  ],
})
export class EditApplicationEntitlementComponent implements OnInit, AfterViewInit, ControlValueAccessor, Validator {
  private readonly injector = inject(Injector);
  private readonly destroyRef = inject(DestroyRef);
  private readonly applicationsService = inject(ApplicationsService);
  protected readonly auth = inject(AuthService);

  applications$ = this.applicationsService.list();
  private fb = inject(FormBuilder);
  editForm = this.fb.nonNullable.group(
    {
      id: this.fb.nonNullable.control<string | undefined>(undefined),
      name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      expiresAt: this.fb.nonNullable.control(''),
      subjectId: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      versionScope: this.fb.nonNullable.control<VersionScope>('all'),
      activeVersions: this.fb.array<boolean>([]),
      archivedVersions: this.fb.array<boolean>([]),
      autoAddVersionRegex: this.fb.nonNullable.control('', [
        Validators.maxLength(200),
        noSurroundingWhitespaceValidator,
      ]),
      customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
      registry: this.fb.nonNullable.group(
        {
          url: this.fb.nonNullable.control(''),
          username: this.fb.nonNullable.control(''),
          password: this.fb.nonNullable.control(''),
        },
        {
          validators: (control) => {
            if (!control.get('url')?.value && !control.get('username')?.value && !control.get('password')?.value) {
              return null;
            }
            if (control.get('url')?.value && control.get('username')?.value && control.get('password')?.value) {
              return null;
            }
            return {
              required: true,
            };
          },
        }
      ),
    },
    {validators: specificVersionsValidator}
  );
  readonly entitlement = signal<ApplicationEntitlement | undefined>(undefined);
  readonly selectedSubject = signal<Application | undefined>(undefined);
  readonly includedArchivedVersions = signal<ApplicationVersion[]>([]);

  readonly dropdownOpen = signal(false);
  protected readonly versionsSelected = signal(0);
  protected readonly dropdownWidth = signal(0);

  protected readonly faChevronDown = faChevronDown;
  protected readonly autoAddVersionRegexDocsUrl = `${WEBSITE_URL}/docs/platform/application-entitlements/#auto-add-version-regex`;

  protected readonly dropdownTriggerButton = viewChild<ElementRef<HTMLElement>>('dropdownTriggerButton');

  constructor() {
    if (!this.auth.hasAnyRole('admin', 'read_write')) {
      this.editForm.disable({emitEvent: false});
    }
  }

  ngOnInit() {
    this.editForm.controls.versionScope.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((versionScope) => {
        const autoAddVersionRegex = this.editForm.controls.autoAddVersionRegex;
        if (versionScope === 'all') {
          autoAddVersionRegex.disable({emitEvent: false});
        } else if (this.editForm.enabled) {
          autoAddVersionRegex.enable({emitEvent: false});
        }
      });
    this.editForm.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      const allVersions = val.versionScope === 'all';
      const versions = allVersions ? [] : this.getSelectedVersions(val.activeVersions, val.archivedVersions);
      this.versionsSelected.set(versions.length);
      if (this.editForm.valid) {
        this.onChange({
          id: val.id,
          name: val.name,
          expiresAt: val.expiresAt ? new Date(val.expiresAt) : undefined,
          applicationId: val.subjectId,
          versions,
          autoAddVersionRegex: (!allVersions && val.autoAddVersionRegex) || undefined,
          customerOrganizationId: val.customerOrganizationId,
          registryUrl: val.registry.url?.trim() || undefined,
          registryUsername: val.registry.username?.trim() || undefined,
          registryPassword: val.registry.password?.trim() || undefined,
        });
      } else {
        this.onChange(undefined);
      }
      this.onValidatorChange();
    });
    this.editForm.controls.subjectId.valueChanges
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        switchMap(async (subjectId) => {
          const apps = await firstValueFrom(this.applicationsService.list());
          return apps.find((a) => a.id === subjectId);
        }),
        filter((a) => a !== undefined)
      )
      .subscribe((selectedApp) => {
        const allVersions = sortApplicationVersions(selectedApp.versioningStrategy, selectedApp.versions ?? []);
        const activeVersions = allVersions.filter((v) => !isArchived(v));
        const archivedVersions = allVersions.filter((v) => isArchived(v));

        const appWithResortedVersions = structuredClone(selectedApp);
        appWithResortedVersions.versions = [...activeVersions, ...archivedVersions];

        this.selectedSubject.set(appWithResortedVersions);
        this.activeVersionsArray.clear({emitEvent: activeVersions.length === 0});
        this.archivedVersionsArray.clear({emitEvent: archivedVersions.length === 0});

        const entitledVersions = this.entitlement()?.versions;
        const archivedSelected = [];

        for (let i = 0; i < activeVersions.length; i++) {
          const version = activeVersions[i];
          const selected = !!entitledVersions?.some((v) => v.id === version.id);
          this.activeVersionsArray.push(this.fb.control(selected), {emitEvent: i === activeVersions.length - 1});
        }

        for (let i = 0; i < archivedVersions.length; i++) {
          const version = archivedVersions[i];
          const selected = !!entitledVersions?.some((v) => v.id === version.id);
          if (selected) {
            archivedSelected.push(version);
          }
          const ctrl = this.fb.control(selected);
          ctrl.disable();
          this.archivedVersionsArray.push(ctrl, {emitEvent: i === archivedVersions.length - 1});
        }

        this.includedArchivedVersions.set(archivedSelected);
      });
  }

  private getSelectedVersions(
    activeVersionControls: (boolean | null)[],
    archivedVersionControls: (boolean | null)[]
  ): ApplicationVersion[] {
    const app = this.selectedSubject();
    const activeSelected = activeVersionControls
      .map((v, idx) => (v ? app?.versions?.[idx] : undefined))
      .filter((v) => !!v);
    const archivedSelected = archivedVersionControls
      .map((v, idx) => (v ? app?.versions?.[idx + activeVersionControls.length] : undefined))
      .filter((v) => !!v);
    return [...activeSelected, ...archivedSelected];
  }

  ngAfterViewInit() {
    // from https://github.com/angular/angular/issues/45089
    this.injector
      .get(NgControl)
      .control!.events.pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((event) => {
        if (event instanceof TouchedChangeEvent) {
          if (event.touched) {
            this.editForm.markAllAsTouched();
          }
        }
      });
  }

  toggleDropdown() {
    this.dropdownOpen.update((v) => !v);
    const trigger = this.dropdownTriggerButton();
    if (this.dropdownOpen() && trigger) {
      this.dropdownWidth.set(trigger.nativeElement.getBoundingClientRect().width);
    }
  }

  get activeVersionsArray() {
    return this.editForm.controls.activeVersions as FormArray;
  }

  get archivedVersionsArray() {
    return this.editForm.controls.archivedVersions as FormArray;
  }

  private onChange: (l: ApplicationEntitlement | ArtifactEntitlement | undefined) => void = () => {};
  private onTouched: () => void = () => {};
  private onValidatorChange: () => void = () => {};

  validate(): ValidationErrors | null {
    return this.editForm.disabled || this.editForm.valid ? null : {invalidEntitlement: true};
  }

  registerOnValidatorChange(fn: () => void): void {
    this.onValidatorChange = fn;
  }

  registerOnChange(fn: (l: ApplicationEntitlement | ArtifactEntitlement | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  writeValue(entitlement: ApplicationEntitlement | undefined): void {
    this.entitlement.set(entitlement);
    if (entitlement) {
      this.editForm.patchValue({
        id: entitlement.id,
        name: entitlement.name,
        expiresAt: entitlement.expiresAt ? dayjs(entitlement.expiresAt).format('YYYY-MM-DD') : '',
        subjectId: entitlement.applicationId,
        activeVersions: [], // will be set by on-change,
        archivedVersions: [], // will be set by on-change,
        versionScope: entitlement.versions?.length ? 'specific' : 'all',
        autoAddVersionRegex: entitlement.autoAddVersionRegex ?? '',
        customerOrganizationId: entitlement.customerOrganizationId,
        registry: {
          url: entitlement.registryUrl || '',
          username: entitlement.registryUsername || '',
          password: entitlement.registryPassword || '',
        },
      });
      if (entitlement.id) {
        this.editForm.controls.subjectId.disable({emitEvent: false});
      }
      this.editForm.controls.customerOrganizationId.disable({emitEvent: false});
    } else {
      this.editForm.reset();
    }
  }

  protected readonly isArchived = isArchived;
}
