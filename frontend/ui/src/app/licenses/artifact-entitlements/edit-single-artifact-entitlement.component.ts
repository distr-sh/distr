import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {AsyncPipe, NgPlural, NgPluralCase} from '@angular/common';
import {
  AfterViewInit,
  Component,
  DestroyRef,
  ElementRef,
  forwardRef,
  inject,
  Injector,
  OnInit,
  output,
  signal,
  viewChild,
} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {
  ControlValueAccessor,
  FormArray,
  FormBuilder,
  FormControl,
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
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {catchError, distinctUntilChanged, of, switchMap, tap} from 'rxjs';
import {WEBSITE_URL} from '../../../constants';
import {RelativeDatePipe} from '../../../util/dates';
import {noSurroundingWhitespaceValidator} from '../../../util/validation';
import {ArtifactsHashComponent} from '../../artifacts/components';
import {ExpiresAtPickerComponent} from '../../components/expires-at-picker/expires-at-picker.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ArtifactsService, ArtifactWithTags} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {ArtifactEntitlement} from '../../types/artifact-entitlement';

type TagScope = 'all' | 'specific';

// An artifact without selected tags grants all of them, so "Specific tags" needs at least one.
const specificTagsValidator: ValidatorFn = (control) => {
  const tagScope = control.get('tagScope')?.value as TagScope;
  const selected = (control.get('artifactTags') as FormArray<FormControl<boolean | null>>)
    .getRawValue()
    .some((v) => !!v);
  return tagScope === 'all' || selected ? null : {tagsRequired: true};
};

@Component({
  selector: 'app-edit-single-artifact-entitlement',
  templateUrl: './edit-single-artifact-entitlement.component.html',
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
    ArtifactsHashComponent,
    RelativeDatePipe,
  ],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => EditSingleArtifactEntitlementComponent),
      multi: true,
    },
    {
      provide: NG_VALIDATORS,
      useExisting: forwardRef(() => EditSingleArtifactEntitlementComponent),
      multi: true,
    },
  ],
})
export class EditSingleArtifactEntitlementComponent implements OnInit, AfterViewInit, ControlValueAccessor, Validator {
  private readonly injector = inject(Injector);
  private readonly destroyRef = inject(DestroyRef);
  private readonly artifactsService = inject(ArtifactsService);
  protected readonly auth = inject(AuthService);

  protected readonly allArtifacts$ = this.artifactsService.list();

  private readonly fb = inject(FormBuilder);
  protected readonly editForm = this.fb.nonNullable.group(
    {
      id: this.fb.nonNullable.control<string | undefined>(undefined),
      name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      expiresAt: this.fb.nonNullable.control(''),
      artifactId: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      tagScope: this.fb.nonNullable.control<TagScope>('all'),
      artifactTags: this.fb.array<boolean>([]),
      autoAddVersionRegex: this.fb.nonNullable.control({value: '', disabled: true}, [
        Validators.maxLength(200),
        noSurroundingWhitespaceValidator,
      ]),
      customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
    },
    {validators: specificTagsValidator}
  );

  /** Emits the entitlement on every change, also while the form is invalid. */
  readonly draftChange = output<ArtifactEntitlement>();
  /** Emits whether "Specific tags" is chosen without any tag checked, which the draft cannot tell from all tags. */
  readonly tagsMissingChange = output<boolean>();

  private readonly entitlement = signal<ArtifactEntitlement | undefined>(undefined);
  protected readonly selectedArtifact = signal<ArtifactWithTags | undefined>(undefined);
  protected readonly tagsSelected = signal(0);
  protected readonly dropdownOpen = signal(false);
  protected readonly dropdownWidth = signal(0);
  private readonly dropdownTriggerButton = viewChild<ElementRef<HTMLElement>>('dropdownTriggerButton');

  protected readonly faChevronDown = faChevronDown;
  protected readonly autoAddTagRegexDocsUrl = `${WEBSITE_URL}/docs/platform/auto-add-regex/`;

  constructor() {
    if (!this.auth.hasAnyRole('admin', 'read_write')) {
      this.editForm.disable({emitEvent: false});
    }
  }

  ngOnInit() {
    this.editForm.controls.tagScope.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe((tagScope) => {
      const autoAddVersionRegex = this.editForm.controls.autoAddVersionRegex;
      if (tagScope === 'all') {
        autoAddVersionRegex.disable({emitEvent: false});
      } else if (this.editForm.enabled) {
        autoAddVersionRegex.enable({emitEvent: false});
      }
    });
    this.editForm.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      const allTags = val.tagScope === 'all';
      const versionIds = this.getSelectedVersions(val.artifactId, val.artifactTags);
      this.tagsSelected.set(versionIds.length);
      const entitlement: ArtifactEntitlement = {
        id: val.id,
        name: val.name,
        expiresAt: val.expiresAt ? new Date(val.expiresAt) : undefined,
        artifacts: [{artifactId: val.artifactId ?? '', versionIds: allTags ? [] : versionIds}],
        autoAddVersionRegex: (!allTags && val.autoAddVersionRegex) || undefined,
        customerOrganizationId: val.customerOrganizationId,
      };
      this.draftChange.emit(entitlement);
      this.tagsMissingChange.emit(!allTags && versionIds.length === 0);
      this.onChange(this.editForm.valid ? entitlement : undefined);
      this.onValidatorChange();
    });
    this.editForm.controls.artifactId.valueChanges
      .pipe(
        distinctUntilChanged(),
        tap(() => {
          this.selectedArtifact.set(undefined);
          this.editForm.controls.artifactTags.clear({emitEvent: false});
        }),
        switchMap((artifactId) =>
          artifactId ? this.artifactsService.get(artifactId).pipe(catchError(() => of(undefined))) : of(undefined)
        ),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe((artifact) => {
        this.selectedArtifact.set(artifact);
        const tags = artifact?.versions ?? [];
        const entitledVersionIds = this.entitlement()?.artifacts?.find(
          (a) => a.artifactId === artifact?.id
        )?.versionIds;
        const artifactTags = this.editForm.controls.artifactTags;
        artifactTags.clear({emitEvent: tags.length === 0});
        for (let i = 0; i < tags.length; i++) {
          const selected = !!entitledVersionIds?.includes(tags[i].id);
          artifactTags.push(this.fb.control(selected), {emitEvent: i === tags.length - 1});
        }
      });
  }

  private getSelectedVersions(artifactId: string | undefined, tagControls: (boolean | null)[]): string[] {
    const artifact = this.selectedArtifact();
    if (!artifact) {
      // Until the tags have loaded, keep the saved selection, since an empty one grants all tags.
      return this.entitlement()?.artifacts?.find((a) => a.artifactId === artifactId)?.versionIds ?? [];
    }
    return tagControls.map((v, idx) => (v ? artifact.versions?.[idx]?.id : undefined)).filter((v) => v !== undefined);
  }

  ngAfterViewInit() {
    // from https://github.com/angular/angular/issues/45089
    this.injector
      .get(NgControl)
      .control!.events.pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((event) => {
        if (event instanceof TouchedChangeEvent && event.touched) {
          this.editForm.markAllAsTouched();
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

  private onChange: (l: ArtifactEntitlement | undefined) => void = () => {};
  private onTouched: () => void = () => {};
  private onValidatorChange: () => void = () => {};

  validate(): ValidationErrors | null {
    return this.editForm.disabled || this.editForm.valid ? null : {invalidEntitlement: true};
  }

  registerOnValidatorChange(fn: () => void): void {
    this.onValidatorChange = fn;
  }

  registerOnChange(fn: (l: ArtifactEntitlement | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  writeValue(entitlement: ArtifactEntitlement | undefined): void {
    this.entitlement.set(entitlement);
    if (entitlement) {
      const selection = entitlement.artifacts?.[0];
      this.editForm.patchValue({
        id: entitlement.id,
        name: entitlement.name,
        expiresAt: entitlement.expiresAt ? dayjs(entitlement.expiresAt).format('YYYY-MM-DD') : '',
        artifactId: selection?.artifactId,
        tagScope: selection?.versionIds?.length ? 'specific' : 'all',
        autoAddVersionRegex: entitlement.autoAddVersionRegex ?? '',
        customerOrganizationId: entitlement.customerOrganizationId,
      });
      this.editForm.controls.customerOrganizationId.disable({emitEvent: false});
    } else {
      this.editForm.reset();
    }
  }
}
