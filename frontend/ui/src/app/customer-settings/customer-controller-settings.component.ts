import {Component, inject, input, signal} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk, faSpinner} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, firstValueFrom, startWith, switchMap} from 'rxjs';
import {WEBSITE_URL} from '../../constants';
import {getFormDisplayedError} from '../../util/errors';
import {AlertComponent} from '../components/alert/alert.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {CustomerSettingsService} from '../services/customer-settings.service';
import {ToastService} from '../services/toast.service';
import {CustomerSettings} from '../types/customer-settings';

@Component({
  selector: 'app-customer-controller-settings',
  templateUrl: './customer-controller-settings.component.html',
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, AlertComponent],
})
export class CustomerControllerSettingsComponent {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faSpinner = faSpinner;

  private readonly customerSettingsService = inject(CustomerSettingsService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  public readonly customerOrganizationId = input.required<string>();

  protected readonly controllerRepositoryDocsUrl = `${WEBSITE_URL}/docs/platform/branding/#controller-image-repository`;
  protected readonly settings = signal<CustomerSettings | undefined>(undefined);
  protected readonly formLoading = signal(false);
  protected readonly loadFailed = signal(false);

  protected readonly form = this.fb.group({
    controllerImageRepositoryEnabled: this.fb.control(false),
    controllerImageRepository: this.fb.control<string | undefined>(undefined),
  });

  protected readonly isControllerImageRepositoryEnabled = toSignal(
    this.form.controls.controllerImageRepositoryEnabled.valueChanges.pipe(
      startWith(this.form.controls.controllerImageRepositoryEnabled.value)
    ),
    {initialValue: false}
  );

  constructor() {
    toObservable(this.customerOrganizationId)
      .pipe(
        switchMap((id) => {
          this.settings.set(undefined);
          this.loadFailed.set(false);
          return this.customerSettingsService.get(id).pipe(
            catchError((e) => {
              this.loadFailed.set(true);
              const msg = getFormDisplayedError(e);
              if (msg) {
                this.toast.error(msg);
              }
              return EMPTY;
            })
          );
        }),
        takeUntilDestroyed()
      )
      .subscribe((settings) => this.applySettings(settings));
  }

  protected async save() {
    const {controllerImageRepositoryEnabled, controllerImageRepository} = this.form.getRawValue();
    this.formLoading.set(true);
    try {
      const settings = await firstValueFrom(
        this.customerSettingsService.update(this.customerOrganizationId(), {
          controllerImageRepository: (controllerImageRepositoryEnabled && controllerImageRepository) || undefined,
        })
      );
      this.applySettings(settings);
      this.toast.success('Settings saved successfully');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  private applySettings(settings: CustomerSettings) {
    this.settings.set(settings);
    this.form.reset({
      controllerImageRepositoryEnabled: !!settings.controllerImageRepository,
      controllerImageRepository: settings.controllerImageRepository,
    });
  }
}
