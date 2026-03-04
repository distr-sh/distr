import {AsyncPipe} from '@angular/common';
import {Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBuildingUser, faKey, faMagnifyingGlass} from '@fortawesome/free-solid-svg-icons';
import {map, Observable, shareReplay} from 'rxjs';
import {isExpired} from '../../util/dates';
import {filteredByFormControl} from '../../util/filter';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {LicensesService} from '../services/licenses.service';
import {License} from '../types/license';

@Component({
  selector: 'app-licenses-overview',
  imports: [ReactiveFormsModule, FaIconComponent, AsyncPipe, AutotrimDirective, SecureImagePipe],
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

  private readonly allLicenses$: Observable<License[]> = this.licensesService.list().pipe(shareReplay(1));

  protected readonly licenses$ = filteredByFormControl(
    this.allLicenses$,
    this.filterForm.controls.search,
    (license, search) => license.customerOrganization.name.toLowerCase().includes(search.toLowerCase())
  );

  protected readonly summary$ = this.allLicenses$.pipe(
    map((licenses) => ({
      totalCustomers: licenses.length,
      totalAppEntitlements: licenses.reduce((sum, l) => sum + l.applicationEntitlements.length, 0),
      totalArtifactEntitlements: licenses.reduce((sum, l) => sum + l.artifactEntitlements.length, 0),
      totalLicenseKeys: licenses.reduce((sum, l) => sum + l.licenseKeys.length, 0),
    }))
  );

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
