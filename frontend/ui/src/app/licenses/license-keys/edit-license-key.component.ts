import {AfterViewInit, Component, computed, DestroyRef, forwardRef, inject, Injector} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
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
import {jsonObjectValidator} from '../../../util/validation';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {LicenseTemplatesService} from '../../services/license-templates.service';
import {LicenseKey} from '../../types/license-key';

@Component({
  selector: 'app-edit-license-key',
  templateUrl: './edit-license-key.component.html',
  imports: [AutotrimDirective, EditorComponent, ReactiveFormsModule, FaIconComponent],
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
  private readonly licenseTemplatesService = inject(LicenseTemplatesService);
  protected readonly customers = toSignal(this.customerOrganizationService.getCustomerOrganizations());
  protected readonly templates = toSignal(this.licenseTemplatesService.list(), {initialValue: []});

  protected readonly faCircleInfo = faCircleInfo;

  private readonly today = dayjs().startOf('day').format('YYYY-MM-DD');
  private readonly inOneYear = dayjs().add(1, 'year').startOf('day').format('YYYY-MM-DD');

  private fb = inject(FormBuilder);
  editForm = this.fb.nonNullable.group(
    {
      id: this.fb.nonNullable.control<string | undefined>(undefined),
      name: this.fb.nonNullable.control<string | undefined>(undefined, Validators.required),
      description: this.fb.nonNullable.control<string | undefined>(undefined),
      expiresAt: this.fb.nonNullable.control(this.inOneYear),
      notBefore: this.fb.nonNullable.control(this.today),
      payload: this.fb.nonNullable.control('{}', jsonObjectValidator),
      customerOrganizationId: this.fb.nonNullable.control<string | undefined>(undefined),
      licenseTemplateId: this.fb.nonNullable.control<string | undefined>(undefined),
    },
    {validators: [this.dateRangeValidator, this.manualFieldsValidator]}
  );

  private readonly editFormValue = toSignal(this.editForm.valueChanges, {initialValue: this.editForm.value});

  protected readonly isEditMode = computed(() => !!this.editFormValue().id);

  protected readonly selectedCustomer = computed(() => {
    const id = this.editFormValue().customerOrganizationId;
    return id && this.customers()?.find((c) => c.id === id);
  });

  protected readonly selectedLicenseTemplate = computed(() => {
    const id = this.editFormValue().licenseTemplateId;
    return id && this.templates().find((t) => t.id === id);
  });

  protected get hasTemplate(): boolean {
    return !!this.editForm.getRawValue().licenseTemplateId;
  }

  constructor() {
    this.editForm.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
      this.onTouched();
      const val = this.editForm.getRawValue();
      const templateSelected = !!val.licenseTemplateId;
      if (this.editForm.valid) {
        const license: LicenseKey = {
          id: val.id,
          name: val.name,
          description: val.description,
          customerOrganizationId: val.customerOrganizationId,
          licenseTemplateId: val.licenseTemplateId || undefined,
          ...(templateSelected
            ? {}
            : {
                payload: JSON.parse(val.payload),
                notBefore: dayjs(val.notBefore).toISOString(),
                expiresAt: dayjs(val.expiresAt).toISOString(),
              }),
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
      this.editForm.patchValue({
        id: license.id,
        name: license.name,
        description: license.description,
        expiresAt: license.expiresAt ? dayjs(license.expiresAt).format('YYYY-MM-DD') : this.inOneYear,
        notBefore: license.notBefore ? dayjs(license.notBefore).format('YYYY-MM-DD') : this.today,
        payload: license.payload ? JSON.stringify(license.payload, null, 2) : '{}',
        customerOrganizationId: license.customerOrganizationId,
        licenseTemplateId: license.licenseTemplateId,
      });
    } else {
      this.editForm.reset({payload: '{}', notBefore: this.today, expiresAt: this.inOneYear});
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

  private dateRangeValidator(group: {value: {notBefore: string; expiresAt: string; licenseTemplateId?: string}}) {
    if (group.value.licenseTemplateId) {
      return null;
    }

    const {notBefore, expiresAt} = group.value;
    if (notBefore && expiresAt && !dayjs(expiresAt).isAfter(notBefore)) {
      return {dateRange: 'Expires At must be after Not Before'};
    }

    return null;
  }

  private manualFieldsValidator(group: {
    value: {payload: string; notBefore: string; expiresAt: string; licenseTemplateId?: string};
  }) {
    if (group.value.licenseTemplateId) {
      return null;
    }

    const errors: Record<string, string> = {};

    if (!group.value.notBefore) {
      errors['notBeforeRequired'] = 'required';
    }

    if (!group.value.expiresAt) {
      errors['expiresAtRequired'] = 'required';
    }

    if (!group.value.payload) {
      errors['payloadRequired'] = 'required';
    }

    return Object.keys(errors).length ? errors : null;
  }
}
