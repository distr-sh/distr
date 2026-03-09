import {AsyncPipe} from '@angular/common';
import {AfterViewInit, Component, DestroyRef, forwardRef, inject, Injector, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {
  ControlValueAccessor,
  FormBuilder,
  NG_VALUE_ACCESSOR,
  NgControl,
  ReactiveFormsModule,
  TouchedChangeEvent,
  Validators,
} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleInfo} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {first} from 'rxjs';
import {jsonObjectValidator} from '../../../util/validation';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {LicenseKey} from '../../types/license-key';

@Component({
  selector: 'app-edit-license-key',
  templateUrl: './edit-license-key.component.html',
  imports: [AsyncPipe, AutotrimDirective, EditorComponent, ReactiveFormsModule, FaIconComponent],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => EditLicenseKeyComponent),
      multi: true,
    },
  ],
})
export class EditLicenseKeyComponent implements AfterViewInit, ControlValueAccessor {
  private readonly injector = inject(Injector);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);
  customers$ = this.customerOrganizationService.getCustomerOrganizations().pipe(first());

  protected readonly faCircleInfo = faCircleInfo;

  private readonly today = dayjs().startOf('day').format('YYYY-MM-DD');
  private readonly inOneYear = dayjs().add(1, 'year').startOf('day').format('YYYY-MM-DD');

  private fb = inject(FormBuilder);
  editForm = this.fb.nonNullable.group(
    {
      id: this.fb.nonNullable.control<string | undefined>(undefined),
      name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      description: this.fb.nonNullable.control<string | undefined>(undefined),
      expiresAt: this.fb.nonNullable.control(this.inOneYear, Validators.required),
      notBefore: this.fb.nonNullable.control(this.today, Validators.required),
      payload: this.fb.nonNullable.control('{}', [Validators.required, jsonObjectValidator]),
      customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
    },
    {validators: this.dateRangeValidator}
  );

  readonly isEditMode = signal(false);

  constructor() {
    this.editForm.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      if (this.editForm.valid) {
        const license: LicenseKey = {
          id: val.id,
          name: val.name,
          description: val.description,
          token: '',
          payload: this.isEditMode() ? {} : JSON.parse(val.payload),
          notBefore: dayjs(val.notBefore).toDate().toISOString(),
          expiresAt: dayjs(val.expiresAt).toDate().toISOString(),
          customerOrganizationId: val.customerOrganizationId,
        };
        this.onChange(license);
      } else {
        this.onChange(undefined);
      }
    });
  }

  ngAfterViewInit() {
    this.injector
      .get(NgControl)
      .control!.events.pipe(takeUntilDestroyed(this.injector.get(DestroyRef)))
      .subscribe((event) => {
        if (event instanceof TouchedChangeEvent && event.touched) {
          this.editForm.markAllAsTouched();
        }
      });
  }

  writeValue(license: LicenseKey | undefined): void {
    if (license) {
      const isEdit = !!license.id;
      this.isEditMode.set(isEdit);
      this.editForm.patchValue({
        id: license.id,
        name: license.name,
        description: license.description,
        expiresAt: license.expiresAt ? dayjs(license.expiresAt).format('YYYY-MM-DD') : this.inOneYear,
        notBefore: license.notBefore ? dayjs(license.notBefore).format('YYYY-MM-DD') : this.today,
        payload: license.payload ? JSON.stringify(license.payload, null, 2) : '{}',
        customerOrganizationId: license.customerOrganizationId,
      });
      this.editForm.controls.customerOrganizationId.disable({emitEvent: false});
      if (isEdit) {
        this.editForm.controls.expiresAt.disable();
        this.editForm.controls.notBefore.disable();
        this.editForm.controls.payload.disable();
      } else {
        this.editForm.controls.expiresAt.enable();
        this.editForm.controls.notBefore.enable();
        this.editForm.controls.payload.enable();
      }
    } else {
      this.isEditMode.set(false);
      this.editForm.reset({payload: '{}', notBefore: this.today, expiresAt: this.inOneYear});
      this.editForm.controls.expiresAt.enable();
      this.editForm.controls.notBefore.enable();
      this.editForm.controls.payload.enable();
      this.editForm.controls.customerOrganizationId.disable({emitEvent: false});
    }
  }

  private onChange: (l: LicenseKey | undefined) => void = () => {};
  private onTouched: () => void = () => {};

  registerOnChange(fn: (l: LicenseKey | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  private dateRangeValidator(group: {value: {notBefore: string; expiresAt: string}}) {
    const {notBefore, expiresAt} = group.value;
    if (notBefore && expiresAt && !dayjs(expiresAt).isAfter(notBefore)) {
      return {dateRange: 'Expires At must be after Not Before'};
    }
    return null;
  }
}
