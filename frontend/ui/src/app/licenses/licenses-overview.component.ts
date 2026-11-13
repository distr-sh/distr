import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faAddressBook, faCircleExclamation, faCopy, faKey} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith} from 'rxjs';
import {isExpired} from '../../util/dates';
import {SecureImagePipe} from '../../util/secureImage';
import {PageComponent} from '../components/page.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {AuthService} from '../services/auth.service';
import {LicensesService} from '../services/licenses.service';
import {OverlayService} from '../services/overlay.service';
import {License} from '../types/license';
import {CopyLicensesContext, CopyLicensesModalComponent} from './copy-licenses-modal.component';

@Component({
  selector: 'app-licenses-overview',
  imports: [AsyncPipe, ReactiveFormsModule, FaIconComponent, SecureImagePipe, PageComponent, SearchBarComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './licenses-overview.component.html',
})
export class LicensesOverviewComponent {
  private readonly licensesService = inject(LicensesService);
  private readonly router = inject(Router);
  private readonly overlay = inject(OverlayService);
  protected readonly auth = inject(AuthService);

  protected readonly faAddressBook = faAddressBook;
  protected readonly faKey = faKey;
  protected readonly faCopy = faCopy;
  protected readonly faCircleExclamation = faCircleExclamation;

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  private readonly allLicenses = toSignal(this.licensesService.list(), {initialValue: []});

  private readonly filterValue = toSignal(
    this.filterForm.controls.search.valueChanges.pipe(startWith(this.filterForm.controls.search.value))
  );

  protected readonly licenses = computed(() => {
    const search = this.filterValue()?.toLowerCase();
    const all = this.allLicenses();
    return !search ? all : all.filter((l) => l.customerOrganization.name.toLowerCase().includes(search));
  });

  protected hasNoLicenses(license: License): boolean {
    return (
      license.applicationEntitlements.length === 0 &&
      license.artifactEntitlements.length === 0 &&
      license.licenseKeys.length === 0
    );
  }

  protected canCopyFromAnotherCustomer(license: License): boolean {
    return this.allLicenses().some(
      (l) => l.customerOrganization.id !== license.customerOrganization.id && !this.hasNoLicenses(l)
    );
  }

  protected navigateToCustomer(license: License) {
    this.router.navigate(['/customers', license.customerOrganization.id, 'licenses']);
  }

  protected countExpired(license: License): number {
    let count = 0;
    for (const ae of license.applicationEntitlements) {
      if (isExpired(ae)) count++;
    }
    for (const ae of license.artifactEntitlements) {
      if (isExpired(ae)) count++;
    }
    for (const lk of license.licenseKeys) {
      if (isExpired(lk)) count++;
    }
    return count;
  }

  protected async openCopyModal(event: Event, license: License) {
    event.stopPropagation();
    const data: CopyLicensesContext = {targetCustomerOrganizationId: license.customerOrganization.id};
    const dialogRef = this.overlay.showModal<boolean>(CopyLicensesModalComponent, {data});
    if (await firstValueFrom(dialogRef.result())) {
      this.navigateToCustomer(license);
    }
  }
}
