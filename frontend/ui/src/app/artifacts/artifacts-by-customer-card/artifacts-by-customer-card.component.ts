import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, inject, input} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {CustomerOrganization} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faAddressBook, faBox, faHeartPulse, faPen, faShip, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {SecureImagePipe} from '../../../util/secureImage';
import {AuthService} from '../../services/auth.service';
import {DashboardArtifact} from '../../services/dashboard.service';

@Component({
  selector: 'app-artifacts-by-customer-card',
  templateUrl: './artifacts-by-customer-card.component.html',
  imports: [FaIconComponent, OverlayModule, ReactiveFormsModule, AsyncPipe, SecureImagePipe, RouterLink],
})
export class ArtifactsByCustomerCardComponent {
  protected readonly auth = inject(AuthService);

  public readonly customer = input.required<CustomerOrganization>();
  public readonly artifacts = input.required<DashboardArtifact[]>();

  protected readonly faShip = faShip;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faHeartPulse = faHeartPulse;
  protected readonly faXmark = faXmark;
  protected readonly faBox = faBox;
  protected readonly faAddressBook = faAddressBook;
}
