import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {
  AfterViewInit,
  Component,
  ElementRef,
  forwardRef,
  inject,
  Injector,
  OnDestroy,
  OnInit,
  signal,
  viewChild,
} from '@angular/core';
import {
  AbstractControl,
  ControlValueAccessor,
  FormArray,
  FormBuilder,
  FormControl,
  NG_VALUE_ACCESSOR,
  NgControl,
  ReactiveFormsModule,
  TouchedChangeEvent,
  Validators,
} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faMagnifyingGlass, faPen, faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {distinctUntilChanged, first, firstValueFrom, of, Subject, switchMap, takeUntil} from 'rxjs';
import {RelativeDatePipe} from '../../../util/dates';
import {ArtifactsHashComponent} from '../../artifacts/components';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ArtifactsService, ArtifactWithTags} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {ArtifactEntitlement, ArtifactEntitlementSelection} from '../../types/artifact-entitlement';

@Component({
  selector: 'app-edit-artifact-entitlement',
  templateUrl: './edit-artifact-entitlement.component.html',
  imports: [
    AsyncPipe,
    AutotrimDirective,
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
      useExisting: forwardRef(() => EditArtifactEntitlementComponent),
      multi: true,
    },
  ],
})
export class EditArtifactEntitlementComponent implements OnInit, OnDestroy, AfterViewInit, ControlValueAccessor {
  private readonly injector = inject(Injector);
  protected readonly auth = inject(AuthService);
  private readonly destroyed$ = new Subject<void>();
  private readonly artifactsService = inject(ArtifactsService);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);
  allArtifacts$ = this.artifactsService.list();
  customers$ = this.customerOrganizationService.getCustomerOrganizations().pipe(first());

  private fb = inject(FormBuilder);
  editForm = this.fb.nonNullable.group({
    id: this.fb.nonNullable.control<string | undefined>(undefined),
    name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
    expiresAt: this.fb.nonNullable.control(''),
    artifacts: this.fb.nonNullable.array<{
      artifactId?: string;
      artifact?: ArtifactWithTags;
      includeAllTags: boolean;
      artifactTags: boolean[];
    }>([], Validators.required),
    customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
  });
  editFormLoading = false;
  readonly entitlement = signal<ArtifactEntitlement | undefined>(undefined);

  readonly openedArtifactIdx = signal<number | undefined>(undefined);
  dropdownWidth: number = 0;
  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faPen = faPen;

  ngOnInit() {
    this.editForm.valueChanges.pipe(takeUntil(this.destroyed$)).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      if (this.editForm.valid) {
        this.onChange({
          id: val.id,
          name: val.name,
          expiresAt: val.expiresAt ? new Date(val.expiresAt) : undefined,
          artifacts: val.artifacts.map((artifact) => {
            return {
              artifactId: artifact.artifact!.id,
              versionIds: this.getSelectedVersions(artifact.includeAllTags, artifact.artifactTags, artifact.artifact!),
            };
          }),
          customerOrganizationId: val.customerOrganizationId,
        });
      } else {
        this.onChange(undefined);
      }
    });
  }

  private getSelectedVersions(
    includeAllTags: boolean,
    itemControls: (boolean | null)[],
    artifact: ArtifactWithTags
  ): string[] {
    if (includeAllTags) {
      return [];
    }
    return itemControls
      .map((v, idx) => {
        if (v) {
          return artifact?.versions?.[idx]?.id;
        }
        return undefined;
      })
      .filter((v) => v !== undefined && v !== null);
  }

  ngAfterViewInit() {
    // from https://github.com/angular/angular/issues/45089
    this.injector
      .get(NgControl)
      .control!.events.pipe(takeUntil(this.destroyed$))
      .subscribe((event) => {
        if (event instanceof TouchedChangeEvent) {
          if (event.touched) {
            this.editForm.markAllAsTouched();
          }
        }
      });
  }

  toggleDropdown(i: number, artifactCtrl: AbstractControl) {
    if (this.openedArtifactIdx() === i) {
      if (
        !artifactCtrl.get('includeAllTags')?.value &&
        !artifactCtrl.get('artifactTags')?.value.some((v: boolean) => v)
      ) {
        artifactCtrl.get('includeAllTags')?.patchValue(true);
      }
    }
    this.openedArtifactIdx.update((idx) => {
      return idx === i ? undefined : i;
    });
    this.dropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
  }

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  get artifacts() {
    return this.editForm.controls.artifacts as FormArray;
  }

  asFormArray(control: AbstractControl): FormArray {
    return control as FormArray;
  }

  asFormControl(control: AbstractControl): FormControl {
    return control as FormControl;
  }

  asArtifactWithTags(control: AbstractControl): ArtifactWithTags | undefined {
    return control.get('artifact')?.value as ArtifactWithTags;
  }

  getSelectedItemsCount(control: AbstractControl): number {
    return ((control.get('artifactTags')?.value ?? []) as boolean[]).filter((v) => v).length;
  }

  addArtifactGroup(selection?: ArtifactEntitlementSelection) {
    const artifactGroup = this.fb.group({
      artifactId: this.fb.nonNullable.control<string | undefined>('', Validators.required),
      artifact: this.fb.nonNullable.control<ArtifactWithTags | undefined>(undefined, Validators.required),
      includeAllTags: this.fb.nonNullable.control<boolean>(false, Validators.required),
      artifactTags: this.fb.array<boolean>([]),
    });
    artifactGroup.controls.includeAllTags.valueChanges.pipe(takeUntil(this.destroyed$)).subscribe((includeAll) => {
      if (includeAll) {
        artifactGroup.controls.artifactTags.controls.forEach((c) => c.patchValue(false, {emitEvent: false}));
      }
    });
    artifactGroup.controls.artifactTags.valueChanges.pipe(takeUntil(this.destroyed$)).subscribe((val) => {
      if (artifactGroup.controls.includeAllTags.value && val.some((v) => !!v)) {
        artifactGroup.controls.includeAllTags.patchValue(false, {emitEvent: false});
      }
    });
    artifactGroup.controls.artifactId.valueChanges
      .pipe(
        takeUntil(this.destroyed$),
        distinctUntilChanged(),
        switchMap(async (artifactId) => {
          const artifacts = await firstValueFrom(this.artifactsService.list());
          return artifacts.find((a) => a.id === artifactId);
        }),
        // TODO calls are made multiple times right now !
        switchMap((artifact) => (artifact ? this.artifactsService.getByIdAndCache(artifact.id) : of(undefined)))
      )
      .subscribe((selectedArtifact) => {
        artifactGroup.controls.artifact.patchValue(selectedArtifact);
        artifactGroup.controls.artifactTags.clear({emitEvent: false});
        const allTagsOfArtifact = (selectedArtifact as ArtifactWithTags)?.versions ?? [];
        const entitlementItems = this.entitlement()?.artifacts?.find(
          (a) => a.artifactId === selectedArtifact?.id
        )?.versionIds;
        let anySelected = false;
        for (let i = 0; i < allTagsOfArtifact.length; i++) {
          const item = allTagsOfArtifact[i];
          const selected = !!entitlementItems?.some((v) => v === item.id);
          artifactGroup.controls.artifactTags.push(this.fb.control(selected), {
            emitEvent: i === allTagsOfArtifact.length - 1,
          });
          anySelected = anySelected || selected;
        }
        if (!anySelected) {
          artifactGroup.controls.includeAllTags.patchValue(true);
        }
      });
    if (selection) {
      artifactGroup.patchValue({
        artifactId: selection.artifactId,
        includeAllTags: (selection?.versionIds || []).length === 0,
      });
    }
    this.artifacts.push(artifactGroup);
  }

  deleteArtifactGroup(i: number) {
    this.artifacts.removeAt(i);
  }

  private onChange: (l: ArtifactEntitlement | undefined) => void = () => {};
  private onTouched: () => void = () => {};

  registerOnChange(fn: (l: ArtifactEntitlement | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }

  writeValue(entitlement: ArtifactEntitlement | undefined): void {
    this.entitlement.set(entitlement);
    if (entitlement) {
      this.editForm.patchValue({
        id: entitlement.id,
        name: entitlement.name,
        artifacts: [],
        expiresAt: entitlement.expiresAt ? dayjs(entitlement.expiresAt).format('YYYY-MM-DD') : '',
        customerOrganizationId: entitlement.customerOrganizationId,
      });
      for (let selection of entitlement.artifacts || []) {
        this.addArtifactGroup(selection);
      }
      this.editForm.controls.customerOrganizationId.disable({emitEvent: false});
    } else {
      this.editForm.reset();
      this.addArtifactGroup();
    }

    if (!this.auth.hasAnyRole('admin', 'read_write')) {
      this.editForm.disable({emitEvent: false});
    }
  }
}
