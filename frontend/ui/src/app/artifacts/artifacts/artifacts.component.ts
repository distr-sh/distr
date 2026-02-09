import {AsyncPipe} from '@angular/common';
import {Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faLightbulb, faMagnifyingGlass, faTrash, faUserCircle} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, map, startWith} from 'rxjs';
import {fromPromise} from 'rxjs/internal/observable/innerFrom';
import {getRemoteEnvironment} from '../../../env/remote';
import {SecureImagePipe} from '../../../util/secureImage';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireCustomerDirective, RequireVendorDirective} from '../../directives/required-role.directive';
import {ArtifactsService} from '../../services/artifacts.service';
import {CustomerOrganizationsCache} from '../../services/customer-organizations.service';
import {OrganizationService} from '../../services/organization.service';
import {ArtifactsDownloadCountComponent, ArtifactsDownloadedByComponent} from '../components';

@Component({
  selector: 'app-artifacts',
  imports: [
    ReactiveFormsModule,
    AsyncPipe,
    FaIconComponent,
    UuidComponent,
    RouterLink,
    ArtifactsDownloadCountComponent,
    ArtifactsDownloadedByComponent,
    AutotrimDirective,
    RequireVendorDirective,
    RequireCustomerDirective,
    SecureImagePipe,
  ],
  templateUrl: './artifacts.component.html',
  providers: [CustomerOrganizationsCache],
})
export class ArtifactsComponent {
  private readonly artifacts = inject(ArtifactsService);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBox = faBox;
  protected readonly faTrash = faTrash;

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  protected readonly artifacts$ = this.artifacts.list();

  protected readonly filteredArtifacts$ = combineLatest([
    this.artifacts$,
    this.filterForm.valueChanges.pipe(startWith(this.filterForm.value)),
  ]).pipe(
    map(([artifacts, formValue]) =>
      artifacts.filter((it) => !formValue.search || it.name.toLowerCase().includes(formValue.search.toLowerCase()))
    )
  );
  protected readonly faLightbulb = faLightbulb;

  private readonly organizationService = inject(OrganizationService);
  protected readonly registrySlug$ = this.organizationService.get().pipe(map((org) => org.slug));
  protected readonly registryHost$ = combineLatest([
    fromPromise(getRemoteEnvironment()),
    this.organizationService.get(),
  ]).pipe(map(([env, org]) => org.registryDomain ?? env.registryHost));
  protected readonly faUserCircle = faUserCircle;
}
