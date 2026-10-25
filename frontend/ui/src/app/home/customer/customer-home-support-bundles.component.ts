import {DatePipe} from '@angular/common';
import {Component, computed, inject, input, output, TemplateRef, viewChild} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faLifeRing, faPlus} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../../util/dates';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundleCreateDialogComponent} from '../../support-bundles/create-dialog/support-bundle-create-dialog.component';
import {SupportBundleStatusBadgeComponent} from '../../support-bundles/support-bundle-status-badge.component';
import {CustomerHomeSupportBundle} from '../../types/customer-home';

const maxSupportBundles = 5;

@Component({
  selector: 'app-customer-home-support-bundles',
  templateUrl: './customer-home-support-bundles.component.html',
  imports: [
    DatePipe,
    RouterLink,
    FaIconComponent,
    RelativeDatePipe,
    SupportBundleCreateDialogComponent,
    SupportBundleStatusBadgeComponent,
    CardHeadingComponent,
  ],
})
export class CustomerHomeSupportBundlesComponent {
  private readonly overlay = inject(OverlayService);

  public readonly supportBundles = input.required<CustomerHomeSupportBundle[]>();
  public readonly canCreate = input.required<boolean>();
  public readonly created = output<void>();

  protected readonly faLifeRing = faLifeRing;
  protected readonly faPlus = faPlus;
  protected readonly visibleSupportBundles = computed(() => this.supportBundles().slice(0, maxSupportBundles));

  private readonly createDialog = viewChild.required<TemplateRef<unknown>>('createDialog');
  private dialogRef?: DialogRef;

  protected openCreateDialog() {
    this.closeCreateDialog();
    this.dialogRef = this.overlay.showModal(this.createDialog());
  }

  protected closeCreateDialog() {
    this.dialogRef?.dismiss();
    this.dialogRef = undefined;
  }
}
