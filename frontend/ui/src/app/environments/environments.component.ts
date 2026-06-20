import {DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {
  faCheck,
  faEdit,
  faLayerGroup,
  faMagnifyingGlass,
  faPlus,
  faRotateRight,
  faTrash,
  faTriangleExclamation,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {filter, firstValueFrom, map, startWith} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {EnvironmentsService} from '../services/environments.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {CreateUpdateEnvironmentRequest, Environment} from '../types/environment';

@Component({
  templateUrl: './environments.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FontAwesomeModule, DecimalPipe, AutotrimDirective],
})
export class EnvironmentsComponent {
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faLayerGroup = faLayerGroup;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faEdit = faEdit;
  protected readonly faCheck = faCheck;
  protected readonly faRotateRight = faRotateRight;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  private readonly environmentsService = inject(EnvironmentsService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly environments = signal<Environment[]>([]);
  protected readonly loading = signal(true);
  protected readonly loadError = signal<string | undefined>(undefined);
  protected readonly formLoading = signal(false);

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });

  protected readonly filteredEnvironments = signal<Environment[]>([]);

  private readonly environmentDialog = viewChild.required<TemplateRef<unknown>>('environmentDialog');
  private modalRef?: DialogRef;

  protected readonly environmentForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control('', [Validators.required]),
    description: this.fb.control(''),
    sortOrder: this.fb.control(0, [Validators.required, Validators.min(0)]),
    isProduction: this.fb.control(false),
    allowDynamicTargets: this.fb.control(false),
  });

  constructor() {
    this.filterForm.controls.search.valueChanges.pipe(startWith('')).subscribe((search) => {
      this.applyFilter(search);
    });
    this.load();
  }

  protected load() {
    this.loading.set(true);
    this.loadError.set(undefined);
    this.environmentsService.list().subscribe({
      next: (environments) => {
        this.environments.set(environments);
        this.applyFilter(this.filterForm.controls.search.value);
        this.loading.set(false);
      },
      error: (e) => {
        this.loadError.set(getFormDisplayedError(e) ?? 'Failed to load environments.');
        this.loading.set(false);
      },
    });
  }

  protected showCreateDialog() {
    this.closeDialog();
    this.environmentForm.reset({
      id: '',
      name: '',
      description: '',
      sortOrder: this.nextSortOrder(),
      isProduction: false,
      allowDynamicTargets: false,
    });
    this.modalRef = this.overlay.showModal(this.environmentDialog());
  }

  protected showUpdateDialog(environment: Environment) {
    this.closeDialog(false);
    this.environmentForm.setValue({
      id: environment.id,
      name: environment.name,
      description: environment.description,
      sortOrder: environment.sortOrder,
      isProduction: environment.isProduction,
      allowDynamicTargets: environment.allowDynamicTargets,
    });
    this.modalRef = this.overlay.showModal(this.environmentDialog());
  }

  protected closeDialog(reset = true) {
    this.modalRef?.close();
    if (reset) {
      this.environmentForm.reset();
    }
  }

  protected async submitForm() {
    this.environmentForm.markAllAsTouched();
    if (this.environmentForm.invalid) {
      return;
    }

    this.formLoading.set(true);
    try {
      const value = this.environmentForm.getRawValue();
      const request: CreateUpdateEnvironmentRequest = {
        name: value.name,
        description: value.description,
        sortOrder: value.sortOrder,
        isProduction: value.isProduction,
        allowDynamicTargets: value.allowDynamicTargets,
      };
      if (value.id) {
        await firstValueFrom(this.environmentsService.update(value.id, request));
      } else {
        await firstValueFrom(this.environmentsService.create(request));
      }
      this.closeDialog();
      this.load();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected delete(environment: Environment) {
    this.overlay
      .confirm({
        message: {
          message: 'Are you sure you want to delete this environment?',
        },
        requiredConfirmInputText: environment.name,
      })
      .pipe(
        filter((it) => it === true),
        map(() => environment.id)
      )
      .subscribe({
        next: async (id) => {
          try {
            await firstValueFrom(this.environmentsService.delete(id));
            this.load();
          } catch (e) {
            const msg = getFormDisplayedError(e);
            if (msg) {
              this.toast.error(msg);
            }
          }
        },
      });
  }

  private applyFilter(search: string) {
    const normalized = search.toLowerCase();
    this.filteredEnvironments.set(
      this.environments().filter(
        (environment) =>
          normalized.length === 0 ||
          environment.name.toLowerCase().includes(normalized) ||
          environment.description.toLowerCase().includes(normalized)
      )
    );
  }

  private nextSortOrder(): number {
    const maxSortOrder = Math.max(0, ...this.environments().map((environment) => environment.sortOrder));
    return maxSortOrder + 10;
  }
}
