import {ChangeDetectionStrategy, Component, input, output, signal} from '@angular/core';
import {FormsModule} from '@angular/forms';
import {CustomDomain} from '../types/custom-domain';
import {DomainFieldComponent} from './domain-field.component';

// The registry domain is optional and starts hidden behind a checkbox, since most organizations serve
// registry traffic from their app domain instead; once enabled, or once a domain already exists, it is
// the same field as any other domain.
@Component({
  selector: 'app-registry-domain-field',
  templateUrl: './registry-domain-field.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FormsModule, DomainFieldComponent],
})
export class RegistryDomainFieldComponent {
  public readonly domain = input<CustomDomain>();
  public readonly cnameTarget = input<string>();
  public readonly saving = input(false);

  public readonly save = output<string>();
  public readonly remove = output<void>();

  protected readonly enabled = signal(false);
}
