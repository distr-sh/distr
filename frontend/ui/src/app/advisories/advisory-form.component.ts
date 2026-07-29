import {ChangeDetectionStrategy, Component, computed, effect, inject, input, output, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormArray, FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {InnerMarkdownDirective} from '../directives/inner-markdown.directive';
import {AdvisoriesService} from '../services/advisories.service';
import {ApplicationsService} from '../services/applications.service';
import {ArtifactsService, TaggedArtifactVersion} from '../services/artifacts.service';
import {ToastService} from '../services/toast.service';
import {
  AdvisoryDetail,
  AdvisorySeverity,
  AdvisoryVersionRelation,
  CreateUpdateAdvisoryRequest,
} from '../types/advisory';
import {advisorySeverities} from './advisory-display';

/** Maps a version id to the relation it has with the advisory. Absent means unselected. */
type VersionSelection = Record<string, AdvisoryVersionRelation>;

export interface AdvisoryFormDraft {
  title: string;
  description: string;
  severity: AdvisorySeverity;
  cveId: string;
  references: {url: string; label: string}[];
  tags: string[];
  tagInput: string;
  applicationVersionSelection: VersionSelection;
  artifactVersionSelection: VersionSelection;
  activeTab: 'details' | 'versions';
  descriptionPreview: boolean;
  expandedApplicationId: string | null;
  expandedArtifactId: string | null;
  versionsTab: 'applications' | 'artifacts';
}

@Component({
  selector: 'app-advisory-form',
  templateUrl: './advisory-form.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, InnerMarkdownDirective],
})
export class AdvisoryFormComponent {
  private readonly advisoriesService = inject(AdvisoriesService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly artifactsService = inject(ArtifactsService);
  private readonly toast = inject(ToastService);

  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly severities = advisorySeverities;

  /** The advisory to edit, or undefined to create a new one. */
  public readonly advisory = input<AdvisoryDetail>();
  public readonly draft = input<AdvisoryFormDraft>();

  public readonly saved = output<AdvisoryDetail>();
  public readonly cancelled = output<void>();
  public readonly dismissed = output<void>();
  public readonly draftChanged = output<AdvisoryFormDraft>();

  protected readonly applications = toSignal(this.applicationsService.list(), {initialValue: []});
  protected readonly artifacts = toSignal(this.artifactsService.list(), {initialValue: []});

  protected readonly form = new FormGroup({
    title: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
    description: new FormControl('', {nonNullable: true}),
    severity: new FormControl<AdvisorySeverity>('none', {nonNullable: true}),
    cveId: new FormControl('', {nonNullable: true}),
    references: new FormArray<FormGroup<{url: FormControl<string>; label: FormControl<string>}>>([]),
  });

  protected readonly activeTab = signal<'details' | 'versions'>('details');
  protected readonly descriptionPreview = signal(false);
  protected readonly loading = signal(false);

  protected readonly tags = signal<string[]>([]);
  protected readonly tagInput = new FormControl('', {nonNullable: true});

  protected readonly applicationVersionSelection = signal<VersionSelection>({});
  protected readonly artifactVersionSelection = signal<VersionSelection>({});

  protected readonly expandedApplicationId = signal<string | null>(null);
  protected readonly expandedArtifactId = signal<string | null>(null);
  protected readonly artifactVersions = signal<Record<string, TaggedArtifactVersion[]>>({});
  protected readonly loadingArtifactIds = signal<ReadonlySet<string>>(new Set());
  protected readonly versionsTab = signal<'applications' | 'artifacts'>('applications');

  protected readonly selectedCount = computed(
    () => Object.keys(this.applicationVersionSelection()).length + Object.keys(this.artifactVersionSelection()).length
  );

  private restored = false;

  constructor() {
    effect(() => {
      if (this.restored) {
        return;
      }
      const draft = this.draft();
      if (draft) {
        this.restored = true;
        this.restoreDraft(draft);
        return;
      }

      const existing = this.advisory();
      if (!existing) {
        return;
      }
      this.restored = true;
      this.form.patchValue({
        title: existing.title,
        description: existing.description,
        severity: existing.severity,
        cveId: existing.cveId ?? '',
      });
      this.form.controls.references.clear();
      for (const reference of existing.references) {
        this.addReference(reference.url, reference.label ?? '');
      }
      this.tags.set([...existing.tags]);
      this.applicationVersionSelection.set(
        Object.fromEntries(existing.applicationVersions.map((v) => [v.applicationVersionId, v.relation]))
      );
      this.artifactVersionSelection.set(
        Object.fromEntries(existing.artifactVersions.map((v) => [v.artifactVersionId, v.relation]))
      );
    });

    this.form.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => this.emitDraft());
    this.tagInput.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => this.emitDraft());
    effect(() => {
      this.tags();
      this.applicationVersionSelection();
      this.artifactVersionSelection();
      this.activeTab();
      this.descriptionPreview();
      this.expandedApplicationId();
      this.expandedArtifactId();
      this.versionsTab();
      this.emitDraft();
    });
  }

  protected addReference(url = '', label = ''): void {
    this.form.controls.references.push(
      new FormGroup({
        url: new FormControl(url, {nonNullable: true, validators: [Validators.required]}),
        label: new FormControl(label, {nonNullable: true}),
      })
    );
  }

  protected removeReference(index: number): void {
    this.form.controls.references.removeAt(index);
  }

  protected addTag(): void {
    const tag = this.tagInput.value.trim();
    if (tag && !this.tags().includes(tag)) {
      this.tags.update((tags) => [...tags, tag]);
    }
    this.tagInput.setValue('');
  }

  protected removeTag(tag: string): void {
    this.tags.update((tags) => tags.filter((t) => t !== tag));
  }

  protected toggleApplication(applicationId: string): void {
    this.expandedApplicationId.update((id) => (id === applicationId ? null : applicationId));
  }

  protected isLoadingArtifactVersions(artifactId: string): boolean {
    return this.loadingArtifactIds().has(artifactId);
  }

  protected async toggleArtifact(artifactId: string): Promise<void> {
    const next = this.expandedArtifactId() === artifactId ? null : artifactId;
    this.expandedArtifactId.set(next);
    // A present key means "loaded", which is what distinguishes an artifact without any
    // versions from one whose versions are still on their way.
    if (next === null || this.artifactVersions()[next] || this.isLoadingArtifactVersions(next)) {
      return;
    }
    this.loadingArtifactIds.update((ids) => new Set(ids).add(next));
    try {
      const artifact = await firstValueFrom(this.artifactsService.getByIdAndCache(next));
      this.artifactVersions.update((versions) => ({...versions, [next]: artifact?.versions ?? []}));
    } catch (e) {
      // Leave the key absent so that reopening the accordion retries.
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.loadingArtifactIds.update((ids) => {
        const remaining = new Set(ids);
        remaining.delete(next);
        return remaining;
      });
    }
  }

  protected relationOf(selection: VersionSelection, versionId: string): AdvisoryVersionRelation | '' {
    return selection[versionId] ?? '';
  }

  protected setApplicationVersionRelation(versionId: string, relation: string): void {
    this.applicationVersionSelection.update((selection) => updateSelection(selection, versionId, relation));
  }

  protected setArtifactVersionRelation(versionId: string, relation: string): void {
    this.artifactVersionSelection.update((selection) => updateSelection(selection, versionId, relation));
  }

  /** Artifact versions are identified by digest, but tags are what users recognize. */
  protected artifactVersionLabel(version: TaggedArtifactVersion): string {
    if (version.tags.length > 0) {
      return version.tags.map((tag) => tag.name).join(', ');
    }
    return version.digest.substring(0, 19);
  }

  protected cancel(): void {
    this.cancelled.emit();
  }

  protected async submit(): Promise<void> {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.loading.set(true);
    try {
      const request = this.buildRequest();
      const existing = this.advisory();
      const result = await firstValueFrom(
        existing
          ? this.advisoriesService.update(existing.id, request)
          : // An advisory written here is the vendor's own, so it skips the triage inbox
            // that exists for issues reported through the API.
            this.advisoriesService.create({...request, status: 'draft'})
      );
      this.toast.success(existing ? 'Advisory updated' : 'Advisory created');
      this.saved.emit(result);
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.loading.set(false);
    }
  }

  private buildRequest(): CreateUpdateAdvisoryRequest {
    const value = this.form.getRawValue();
    const cveId = value.cveId.trim();
    return {
      title: value.title,
      description: value.description,
      severity: value.severity,
      cveId: cveId || undefined,
      tags: this.tags(),
      references: value.references.map((reference) => ({
        url: reference.url,
        label: reference.label.trim() || undefined,
      })),
      affectedApplicationVersionIds: idsWithRelation(this.applicationVersionSelection(), 'affected'),
      fixedApplicationVersionIds: idsWithRelation(this.applicationVersionSelection(), 'fixed'),
      affectedArtifactVersionIds: idsWithRelation(this.artifactVersionSelection(), 'affected'),
      fixedArtifactVersionIds: idsWithRelation(this.artifactVersionSelection(), 'fixed'),
    };
  }

  private restoreDraft(draft: AdvisoryFormDraft): void {
    this.form.patchValue({
      title: draft.title,
      description: draft.description,
      severity: draft.severity,
      cveId: draft.cveId,
    });
    this.form.controls.references.clear();
    for (const reference of draft.references) {
      this.addReference(reference.url, reference.label);
    }
    this.tags.set([...draft.tags]);
    this.tagInput.setValue(draft.tagInput);
    this.applicationVersionSelection.set({...draft.applicationVersionSelection});
    this.artifactVersionSelection.set({...draft.artifactVersionSelection});
    this.activeTab.set(draft.activeTab);
    this.descriptionPreview.set(draft.descriptionPreview);
    this.expandedApplicationId.set(draft.expandedApplicationId);
    this.expandedArtifactId.set(draft.expandedArtifactId);
    this.versionsTab.set(draft.versionsTab);
  }

  private emitDraft(): void {
    const value = this.form.getRawValue();
    this.draftChanged.emit({
      title: value.title,
      description: value.description,
      severity: value.severity,
      cveId: value.cveId,
      references: value.references,
      tags: [...this.tags()],
      tagInput: this.tagInput.value,
      applicationVersionSelection: {...this.applicationVersionSelection()},
      artifactVersionSelection: {...this.artifactVersionSelection()},
      activeTab: this.activeTab(),
      descriptionPreview: this.descriptionPreview(),
      expandedApplicationId: this.expandedApplicationId(),
      expandedArtifactId: this.expandedArtifactId(),
      versionsTab: this.versionsTab(),
    });
  }
}

function updateSelection(selection: VersionSelection, versionId: string, relation: string): VersionSelection {
  const next = {...selection};
  if (relation === 'affected' || relation === 'fixed') {
    next[versionId] = relation;
  } else {
    delete next[versionId];
  }
  return next;
}

function idsWithRelation(selection: VersionSelection, relation: AdvisoryVersionRelation): string[] {
  return Object.entries(selection)
    .filter(([, value]) => value === relation)
    .map(([id]) => id);
}
