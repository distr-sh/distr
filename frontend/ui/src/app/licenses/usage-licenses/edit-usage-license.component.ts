import {AsyncPipe} from '@angular/common';
import {AfterViewInit, Component, forwardRef, inject, Injector, OnDestroy, signal} from '@angular/core';
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
import {first, Subject, takeUntil} from 'rxjs';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {UsageLicense} from '../../types/usage-license';

@Component({
  selector: 'app-edit-usage-license',
  templateUrl: './edit-usage-license.component.html',
  imports: [AsyncPipe, AutotrimDirective, EditorComponent, ReactiveFormsModule, FaIconComponent],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => EditUsageLicenseComponent),
      multi: true,
    },
  ],
})
export class EditUsageLicenseComponent implements AfterViewInit, OnDestroy, ControlValueAccessor {
  private readonly injector = inject(Injector);
  private readonly destroyed$ = new Subject<void>();
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);
  customers$ = this.customerOrganizationService.getCustomerOrganizations().pipe(first());

  protected readonly faCircleInfo = faCircleInfo;

  private readonly today = dayjs().startOf('day').format('YYYY-MM-DDTHH:mm');
  private readonly inOneYear = dayjs().add(1, 'year').startOf('day').format('YYYY-MM-DDTHH:mm');

  private fb = inject(FormBuilder);
  editForm = this.fb.nonNullable.group(
    {
      id: this.fb.nonNullable.control<string | undefined>(undefined),
      name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      description: this.fb.nonNullable.control<string | undefined>(undefined),
      expiresAt: this.fb.nonNullable.control(this.inOneYear, Validators.required),
      notBefore: this.fb.nonNullable.control(this.today, Validators.required),
      payload: this.fb.nonNullable.control('{}', [Validators.required, this.jsonValidator]),
      customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
    },
    {validators: this.dateRangeValidator}
  );

  readonly isEditMode = signal(false);

  constructor() {
    this.editForm.valueChanges.pipe(takeUntil(this.destroyed$)).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      if (this.editForm.valid) {
        const license: UsageLicense = {
          id: val.id,
          name: val.name,
          description: val.description,
          token: '',
          payload: this.isEditMode() ? {} : JSON.parse(val.payload),
          notBefore: dayjs(val.notBefore).toISOString(),
          expiresAt: dayjs(val.expiresAt).toISOString(),
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
      .control!.events.pipe(takeUntil(this.destroyed$))
      .subscribe((event) => {
        if (event instanceof TouchedChangeEvent && event.touched) {
          this.editForm.markAllAsTouched();
        }
      });
  }

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  writeValue(license: UsageLicense | undefined): void {
    if (license) {
      const isEdit = !!license.id;
      this.isEditMode.set(isEdit);
      this.editForm.patchValue({
        id: license.id,
        name: license.name,
        description: license.description,
        expiresAt: license.expiresAt ? dayjs(license.expiresAt).format('YYYY-MM-DDTHH:mm') : '',
        notBefore: license.notBefore ? dayjs(license.notBefore).format('YYYY-MM-DDTHH:mm') : '',
        payload: JSON.stringify(license.payload, null, 2),
        customerOrganizationId: license.customerOrganizationId,
      });
      if (isEdit) {
        this.editForm.controls.expiresAt.disable();
        this.editForm.controls.notBefore.disable();
        this.editForm.controls.payload.disable();
        this.editForm.controls.customerOrganizationId.disable();
      } else {
        this.editForm.controls.expiresAt.enable();
        this.editForm.controls.notBefore.enable();
        this.editForm.controls.payload.enable();
        this.editForm.controls.customerOrganizationId.enable();
      }
    } else {
      this.isEditMode.set(false);
      this.editForm.reset({payload: '{}', notBefore: this.today, expiresAt: this.inOneYear});
      this.editForm.controls.expiresAt.enable();
      this.editForm.controls.notBefore.enable();
      this.editForm.controls.payload.enable();
      this.editForm.controls.customerOrganizationId.enable();
    }
  }

  private onChange: (l: UsageLicense | undefined) => void = () => {};
  private onTouched: () => void = () => {};

  registerOnChange(fn: (l: UsageLicense | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  private dateRangeValidator(group: {value: {notBefore: string; expiresAt: string}}) {
    const {notBefore, expiresAt} = group.value;
    if (notBefore && expiresAt && !dayjs(expiresAt).isAfter(dayjs(notBefore))) {
      return {dateRange: 'Expires At must be after Not Before'};
    }
    return null;
  }

  private jsonValidator(control: {value: string}) {
    try {
      const parsed = JSON.parse(control.value);
      if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
        return {json: 'Payload must be a JSON object'};
      }
      return null;
    } catch {
      return {json: 'Invalid JSON'};
    }
  }
}
