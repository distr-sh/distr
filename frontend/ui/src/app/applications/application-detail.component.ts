import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe, NgOptimizedImage} from '@angular/common';
import {
  Component,
  ElementRef,
  inject,
  OnDestroy,
  OnInit,
  signal,
  TemplateRef,
  viewChild,
  ViewChild,
} from '@angular/core';
import {FormArray, FormControl, FormGroup, FormsModule, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {Application, ApplicationVersion, ApplicationVersionResource, HelmChartType} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArchive,
  faBox,
  faBoxesStacked,
  faCheck,
  faChevronDown,
  faEdit,
  faEye,
  faMagnifyingGlass,
  faMinus,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {MarkdownPipe} from 'ngx-markdown';
import {
  catchError,
  combineLatestWith,
  distinctUntilChanged,
  EMPTY,
  filter,
  firstValueFrom,
  lastValueFrom,
  map,
  Observable,
  startWith,
  Subject,
  switchMap,
  takeUntil,
  tap,
} from 'rxjs';
import {isArchived} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {disableControlsWithoutEvent, enableControlsWithoutEvent} from '../../util/forms';
import {SecureImagePipe} from '../../util/secureImage';
import {dropdownAnimation} from '../animations/dropdown';
import {EditorComponent} from '../components/editor.component';
import {UuidComponent} from '../components/uuid';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ImageUploadService} from '../services/image-upload.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {
  ApplicationVersionDetail,
  ApplicationVersionDetailModalComponent,
} from './application-version-detail-modal.component';

@Component({
  selector: 'app-application-detail',
  imports: [
    ReactiveFormsModule,
    OverlayModule,
    AsyncPipe,
    RouterLink,
    FaIconComponent,
    NgOptimizedImage,
    UuidComponent,
    AutotrimDirective,
    DatePipe,
    EditorComponent,
    SecureImagePipe,
    FormsModule,
    ApplicationVersionDetailModalComponent,
    MarkdownPipe,
  ],
  templateUrl: './application-detail.component.html',
  animations: [dropdownAnimation],
})
export class ApplicationDetailComponent implements OnInit, OnDestroy {
  private readonly destroyed$ = new Subject<void>();
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly applicationService = inject(ApplicationsService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  protected readonly auth = inject(AuthService);
  readonly applications$: Observable<Application[]> = this.applicationService.list();
  filterForm = new FormGroup({
    showArchived: new FormControl<boolean>(false),
  });
  readonly application$: Observable<Application | undefined> = this.route.paramMap.pipe(
    map((params) => params.get('applicationId')?.trim()),
    distinctUntilChanged(),
    tap(() => {
      this.newVersionForm.reset();
      this.newVersionFormLoading.set(false);
    }),
    combineLatestWith(this.applications$),
    map(([id, applications]) => applications.find((a) => a.id === id)),
    tap((app) => {
      this.editForm.disable();
      if (app) {
        this.editForm.patchValue({name: app.name});
        this.enableTypeSpecificGroups(app);
      }
    })
  );
  readonly visibleVersions$ = this.application$.pipe(
    combineLatestWith(this.filterForm.valueChanges.pipe(startWith({showArchived: false}))),
    map(([app, filter]) => {
      if (app && !filter.showArchived) {
        return (app.versions ?? []).filter((av) => !isArchived(av));
      }
      return app?.versions;
    })
  );

  protected selectedVersionIds = signal(new Set<string>());

  newVersionForm = new FormGroup({
    versionName: new FormControl('', Validators.required),
    linkTemplate: new FormControl(''),
    kubernetes: new FormGroup(
      {
        chartType: new FormControl<HelmChartType>('repository', {
          nonNullable: true,
          validators: Validators.required,
        }),
        chartName: new FormControl('', Validators.required),
        chartUrl: new FormControl('', Validators.required),
        chartVersion: new FormControl('', Validators.required),
        baseValues: new FormControl(''),
        template: new FormControl(''),
      },
      (v) => {
        if (v.value.chartType === 'oci' && v.value.chartUrl && !/^oci:\/\/.+/.test(v.value.chartUrl)) {
          return {chartUrlOci: true};
        }
        if (v.value.chartType === 'repository' && v.value.chartUrl && !/^https:\/\/.+/.test(v.value.chartUrl)) {
          return {chartUrlHttps: true};
        }
        return null;
      }
    ),
    docker: new FormGroup({
      compose: new FormControl(''),
      template: new FormControl(''),
    }),
    resources: new FormArray<
      FormGroup<{
        name: FormControl<string>;
        content: FormControl<string>;
        visibleToCustomers: FormControl<boolean>;
      }>
    >([]),
  });
  newVersionFormLoading = signal(false);
  editForm = new FormGroup({
    name: new FormControl('', Validators.required),
  });
  editFormLoading = signal(false);

  protected readonly versionDetail = signal<ApplicationVersionDetail | undefined>(undefined);
  protected readonly versionDetailsModal = viewChild.required<TemplateRef<unknown>>('versionDetailsModal');
  private versionDetailsModalRef?: DialogRef;

  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faEdit = faEdit;
  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;
  protected readonly faTrash = faTrash;
  protected readonly faArchive = faArchive;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBox = faBox;

  protected readonly isArchived = isArchived;
  readonly breadcrumbDropdown = signal(false);
  readonly isVersionFormExpanded = signal(false);
  breadcrumbDropdownWidth: number = 0;
  @ViewChild('dropdownTriggerButton') dropdownTriggerButton!: ElementRef<HTMLElement>;
  @ViewChild('nameInput') nameInputElem?: ElementRef<HTMLInputElement>;

  ngOnInit() {
    this.route.url.subscribe(() => this.breadcrumbDropdown.set(false));

    this.newVersionForm.controls.kubernetes.controls.chartType.valueChanges
      .pipe(takeUntil(this.destroyed$))
      .subscribe((type) => {
        if (type === 'repository') {
          this.newVersionForm.controls.kubernetes.controls.chartName.enable();
        } else {
          this.newVersionForm.controls.kubernetes.controls.chartName.disable();
        }
      });
  }

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton.nativeElement.getBoundingClientRect().width;
    }
  }

  enableApplicationEdit(application: Application) {
    this.editForm.enable();
    this.editForm.patchValue({name: application.name});
    setTimeout(() => this.nameInputElem?.nativeElement.focus(), 10);
  }

  cancelApplicationEdit() {
    this.editForm.disable();
  }

  async saveApplication(application: Application) {
    if (this.editForm.valid) {
      this.editFormLoading.set(true);
      try {
        await lastValueFrom(
          this.applicationService.update({
            ...application,
            name: this.editForm.value.name!.trim(),
          })
        );
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editFormLoading.set(false);
      }
    }
  }

  async createVersion(application: Application) {
    this.newVersionForm.markAllAsTouched();
    if (this.newVersionForm.valid && application) {
      this.newVersionFormLoading.set(true);
      let res;
      const resources: ApplicationVersionResource[] = this.newVersionForm.controls.resources.controls
        .map((g) => ({
          name: g.controls.name.value,
          content: g.controls.content.value,
          visibleToCustomers: g.controls.visibleToCustomers.value,
        }))
        .filter((r) => r.name && r.content);

      if (application.type === 'docker') {
        res = this.applicationService.createApplicationVersionForDocker(
          application,
          {
            name: this.newVersionForm.controls.versionName.value!,
            linkTemplate: this.newVersionForm.controls.linkTemplate.value!,
            resources,
          },
          this.newVersionForm.controls.docker.controls.compose.value!,
          this.newVersionForm.controls.docker.controls.template.value
        );
      } else {
        const versionFormVal = this.newVersionForm.controls.kubernetes.value;
        res = this.applicationService.createApplicationVersionForKubernetes(
          application,
          {
            name: this.newVersionForm.controls.versionName.value!,
            linkTemplate: this.newVersionForm.controls.linkTemplate.value!,
            chartType: versionFormVal.chartType!,
            chartName: versionFormVal.chartName ?? undefined,
            chartUrl: versionFormVal.chartUrl!,
            chartVersion: versionFormVal.chartVersion!,
            resources,
          },
          versionFormVal.baseValues,
          versionFormVal.template
        );
      }

      try {
        const av = await firstValueFrom(res);
        this.toast.success(`${av.name} created successfully`);
        this.newVersionForm.reset();
        this.enableTypeSpecificGroups(application);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.newVersionFormLoading.set(false);
      }
    }
  }

  async fillVersionFormWith(application: Application, version: ApplicationVersion) {
    this.isVersionFormExpanded.set(true);
    const val = await this.loadVersionDetails(application, version);
    if (val) {
      this.newVersionForm.controls.resources.clear();
      for (const _ of val.resources ?? []) {
        this.addResource();
      }
      this.newVersionForm.patchValue(val);
    }
  }

  async loadVersionDetails(application: Application, version: ApplicationVersion) {
    try {
      const resources = await firstValueFrom(this.applicationService.getResources(application.id!, version.id!));
      if (application.type === 'kubernetes') {
        const template = await firstValueFrom(this.applicationService.getTemplateFile(application.id!, version.id!));
        const baseValues = await firstValueFrom(this.applicationService.getValuesFile(application.id!, version.id!));
        return {
          linkTemplate: version.linkTemplate,
          kubernetes: {
            chartType: version.chartType,
            chartName: version.chartName,
            chartUrl: version.chartUrl,
            chartVersion: version.chartVersion,
            baseValues,
            template,
          },
          resources,
        };
      } else if (application.type === 'docker') {
        const template = await firstValueFrom(this.applicationService.getTemplateFile(application.id!, version.id!));
        const compose = await firstValueFrom(this.applicationService.getComposeFile(application.id!, version.id!));
        return {
          linkTemplate: version.linkTemplate,
          docker: {
            compose,
            template,
          },
          resources,
        };
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      return undefined;
    }
    return undefined;
  }

  deleteApplication(application: Application) {
    this.overlay
      .confirm(`Really delete ${application.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(async () => {
          await lastValueFrom(this.applicationService.delete(application));
          return this.router.navigate(['/applications']);
        }),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return EMPTY;
        })
      )
      .subscribe();
  }

  archiveVersion(application: Application, version: ApplicationVersion) {
    this.overlay
      .confirm(`Really archive ${version.name}? Existing deployments will continue to work.`)
      .pipe(
        filter((result) => result === true),
        switchMap(() =>
          this.applicationService.updateApplicationVersion(application, {
            ...version,
            archivedAt: new Date().toISOString(),
          })
        ),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return EMPTY;
        })
      )
      .subscribe();
  }

  async bulkArchiveVersions(app: Application) {
    this.overlay
      .confirm(`Really archive ${this.selectedVersionIds().size} versions? Existing deployments will continue to work.`)
      .pipe(
        filter((it) => it === true),
        switchMap(() =>
          this.applicationService.patch(app.id!, {
            versions: app?.versions
              ?.filter((version) => this.isVersionSelected(version))
              .map((version) => ({id: version.id!, archivedAt: new Date().toISOString()})),
          })
        )
      )
      .subscribe({
        next: () => {
          this.toast.success(`${this.selectedVersionIds().size} archived`);
          this.selectedVersionIds.set(new Set());
        },
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  async unArchiveVersion(application: Application, version: ApplicationVersion) {
    try {
      await lastValueFrom(
        this.applicationService.updateApplicationVersion(application, {
          ...version,
          archivedAt: undefined,
        })
      );
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected hideVersionDetails(): void {
    this.versionDetailsModalRef?.close();
    this.versionDetail.set(undefined);
  }

  async openVersionDetails(application: Application, version: ApplicationVersion) {
    this.hideVersionDetails();
    const val = await this.loadVersionDetails(application, version);
    if (val) {
      this.versionDetail.set({
        application,
        version,
        linkTemplate: val.linkTemplate ?? '',
        kubernetes: val.kubernetes,
        docker: val.docker,
        resources: val.resources ?? [],
      });
      this.versionDetailsModalRef = this.overlay.showModal(this.versionDetailsModal(), {
        positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
      });
    }
  }

  private enableTypeSpecificGroups(app: Application) {
    if (app.type === 'kubernetes') {
      enableControlsWithoutEvent(this.newVersionForm.controls.kubernetes);
      disableControlsWithoutEvent(this.newVersionForm.controls.docker);
    } else {
      enableControlsWithoutEvent(this.newVersionForm.controls.docker);
      disableControlsWithoutEvent(this.newVersionForm.controls.kubernetes);
    }
  }

  public async uploadImage(data: Application) {
    const fileId = await firstValueFrom(this.imageUploadService.showDialog({imageUrl: data.imageUrl}));
    if (!fileId || data.imageUrl?.includes(fileId)) {
      return;
    }
    await firstValueFrom(this.applicationService.patchImage(data.id!, fileId));
  }

  protected toggleVersionSelected(version: ApplicationVersion): void {
    const id = version.id;
    if (id === undefined) {
      return;
    }

    this.selectedVersionIds.update((ids) => {
      if (this.isVersionSelected(version)) {
        ids.delete(id);
        return ids;
      } else {
        return ids.add(id);
      }
    });
  }

  protected isVersionSelected(version: ApplicationVersion): boolean {
    return version.id !== undefined && this.selectedVersionIds().has(version.id);
  }

  protected toggleAllVersionsSelected(versions: ApplicationVersion[]) {
    versions = versions.filter((version) => !isArchived(version));
    if (this.isAllVersionsSelected(versions)) {
      this.selectedVersionIds.set(new Set());
    } else {
      this.selectedVersionIds.set(new Set(versions.map((v) => v.id).filter((id) => id !== undefined)));
    }
  }

  protected isAllVersionsSelected(versions: ApplicationVersion[]): boolean {
    versions = versions.filter((version) => !isArchived(version));
    return (
      this.selectedVersionIds().size > 0 &&
      versions.length === this.selectedVersionIds().size &&
      versions.every((version) => this.isVersionSelected(version))
    );
  }

  protected readonly faEye = faEye;
  protected readonly faPlus = faPlus;
  protected readonly faMinus = faMinus;

  protected readonly resourcePreviewIndices = signal(new Set<number>());

  toggleResourcePreview(index: number) {
    this.resourcePreviewIndices.update((set) => {
      const next = new Set(set);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  }

  addResource() {
    this.newVersionForm.controls.resources.push(
      new FormGroup({
        name: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
        content: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
        visibleToCustomers: new FormControl(false, {nonNullable: true}),
      })
    );
  }

  removeResource(index: number) {
    this.newVersionForm.controls.resources.removeAt(index);
  }
}
