import {ChangeDetectionStrategy, Component, inject, signal, viewChild} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk} from '@fortawesome/free-solid-svg-icons';
import {BrandingFormComponent} from '../components/branding/branding-form.component';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-organization-branding',
  templateUrl: './organization-branding.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [BrandingFormComponent, FaIconComponent],
})
export class OrganizationBrandingComponent {
  protected readonly faFloppyDisk = faFloppyDisk;

  protected readonly auth = inject(AuthService);

  private readonly brandingForm = viewChild.required(BrandingFormComponent);

  protected readonly formLoading = signal(false);

  protected async save() {
    this.formLoading.set(true);
    try {
      await this.brandingForm().save();
    } finally {
      this.formLoading.set(false);
    }
  }
}
