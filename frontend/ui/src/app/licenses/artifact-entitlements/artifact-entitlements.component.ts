import {AsyncPipe, DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, input, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, EMPTY, filter, firstValueFrom, map, Observable, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {AlertComponent} from '../../components/alert/alert.component';
import {SearchBarComponent} from '../../components/search-bar.component';
import {ClickableRowDirective} from '../../directives/clickable-row.directive';
import {ArtifactEntitlementsService} from '../../services/artifact-entitlements.service';
import {ArtifactsService} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ArtifactEntitlement, ArtifactEntitlementSelection} from '../../types/artifact-entitlement';
import {EditArtifactEntitlementComponent} from './edit-artifact-entitlement.component';
import {EditSingleArtifactEntitlementComponent} from './edit-single-artifact-entitlement.component';

type EntitlementForm = 'standard' | 'legacy';

@Component({
  selector: 'app-artifact-entitlements',
  imports: [
    ReactiveFormsModule,
    AsyncPipe,
    FaIconComponent,
    DatePipe,
    EditArtifactEntitlementComponent,
    EditSingleArtifactEntitlementComponent,
    SearchBarComponent,
    AlertComponent,
    ClickableRowDirective,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './artifact-entitlements.component.html',
})
export class ArtifactEntitlementsComponent {
  readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);
  private readonly artifactsService = inject(ArtifactsService);

  protected readonly canOpenDetails = this.auth.isVendor() || this.auth.isPartner();

  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  entitlements$: Observable<ArtifactEntitlement[]> = combineLatest([
    filteredByFormControl(
      this.artifactEntitlementsService.list(),
      this.filterForm.controls.search,
      (it: ArtifactEntitlement, search: string) =>
        !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
    ),
    toObservable(this.customerOrganizationId),
  ]).pipe(
    map(([entitlements, id]) => (id ? entitlements.filter((e) => e.customerOrganizationId === id) : entitlements)),
    takeUntilDestroyed()
  );

  editForm = new FormGroup({
    entitlement: new FormControl<ArtifactEntitlement | undefined>(undefined, {
      nonNullable: true,
    }),
  });
  editFormLoading = false;

  protected readonly entitlementForm = signal<EntitlementForm>('standard');
  protected readonly draft = signal<ArtifactEntitlement | undefined>(undefined);
  protected readonly tagsMissing = signal(false);
  protected readonly otherEntitlementForm = computed<EntitlementForm>(() =>
    this.entitlementForm() === 'standard' ? 'legacy' : 'standard'
  );
  private readonly hasAutoAddVersionRegex = computed(() => !!this.draft()?.autoAddVersionRegex);
  protected readonly entitlementFormSwitchDisabled = computed(() =>
    this.entitlementForm() === 'standard'
      ? this.hasAutoAddVersionRegex() || this.tagsMissing()
      : (this.draft()?.artifacts?.length ?? 0) > 1
  );
  protected readonly entitlementFormSwitchTitle = computed(() => {
    if (this.otherEntitlementForm() === 'legacy') {
      const hint =
        'The legacy view can bundle multiple artifacts in one entitlement but does not support an auto-add tag regex.';
      if (this.hasAutoAddVersionRegex()) {
        return `${hint} Clear the auto-add tag regex to switch.`;
      } else if (this.tagsMissing()) {
        return `${hint} To switch, select at least one tag or pick the "All tags" option.`;
      }
      return hint;
    }
    const hint = 'The standard view supports an auto-add tag regex but only for an entitlement with a single artifact.';
    return this.entitlementFormSwitchDisabled() ? `${hint} Remove all but one artifact to switch.` : hint;
  });

  private manageEntitlementDrawerRef?: DialogRef;

  private readonly customerOrganizations$ = this.customerOrganizationService.getCustomerOrganizations();
  private readonly customerOrganizations = toSignal(this.customerOrganizations$);
  protected readonly drawerCustomerName = computed(() => {
    const customerOrganizationId = this.draft()?.customerOrganizationId;
    return customerOrganizationId
      ? this.customerOrganizations()?.find((c) => c.id === customerOrganizationId)?.name
      : undefined;
  });
  private readonly artifacts$ = this.artifactsService.list();

  openDrawer(templateRef: TemplateRef<unknown>, entitlement?: ArtifactEntitlement) {
    this.hideDrawer();
    this.entitlementForm.set((entitlement?.artifacts?.length ?? 0) > 1 ? 'legacy' : 'standard');
    if (entitlement) {
      this.loadEntitlement(entitlement);
    } else if (this.customerOrganizationId()) {
      this.editForm.patchValue({
        entitlement: {customerOrganizationId: this.customerOrganizationId()} as ArtifactEntitlement,
      });
    }
    this.manageEntitlementDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadEntitlement(entitlement: ArtifactEntitlement) {
    this.editForm.patchValue({entitlement});
  }

  hideDrawer() {
    this.manageEntitlementDrawerRef?.close();
    this.editForm.reset({entitlement: undefined});
    this.draft.set(undefined);
    this.tagsMissing.set(false);
  }

  protected switchEntitlementForm(form: EntitlementForm) {
    // The form being replaced only reports a valid entitlement to the control, so hand its draft to the new form.
    this.editForm.controls.entitlement.setValue(this.draft(), {emitModelToViewChange: false});
    this.tagsMissing.set(false);
    this.entitlementForm.set(form);
  }

  async saveEntitlement() {
    this.editForm.markAllAsTouched();
    const {entitlement} = this.editForm.value;
    if (this.editForm.valid && entitlement) {
      this.editFormLoading = true;
      const action = entitlement.id
        ? this.artifactEntitlementsService.update(entitlement)
        : this.artifactEntitlementsService.create(entitlement);
      try {
        const saved = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${saved.name} saved successfully`);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  deleteEntitlement(entitlement: ArtifactEntitlement) {
    this.overlay
      .confirm(`Really delete ${entitlement.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.artifactEntitlementsService.delete(entitlement)),
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

  getArtifactColumn(selection?: ArtifactEntitlementSelection[]): Observable<string | undefined> {
    return selection?.[0]?.artifactId
      ? this.artifacts$.pipe(
          map((artifacts) => artifacts.find((a) => a.id === selection?.[0]?.artifactId)),
          map((a) => a?.name + (selection?.length > 1 ? ' (+' + (selection.length - 1) + ')' : ''))
        )
      : EMPTY;
  }

  getOwnerColumn(customerOrganizationId?: string): Observable<string | undefined> {
    return customerOrganizationId
      ? this.customerOrganizations$.pipe(map((users) => users.find((u) => u.id === customerOrganizationId)?.name))
      : EMPTY;
  }
}
