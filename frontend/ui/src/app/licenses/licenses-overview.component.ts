import {AsyncPipe} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBuildingUser, faKey, faMagnifyingGlass} from '@fortawesome/free-solid-svg-icons';
import {startWith} from 'rxjs';
import {isExpired} from '../../util/dates';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {LicensesService} from '../services/licenses.service';
import {License} from '../types/license';

@Component({
  selector: 'app-licenses-overview',
  imports: [AsyncPipe, ReactiveFormsModule, FaIconComponent, AutotrimDirective, SecureImagePipe],
  templateUrl: './licenses-overview.component.html',
})
export class LicensesOverviewComponent {
  private readonly licensesService = inject(LicensesService);
  private readonly router = inject(Router);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faKey = faKey;

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

  protected navigateToCustomer(license: License) {
    this.router.navigate(['/licenses', license.customerOrganization.id]);
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
}
