import {DatePipe} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faMagnifyingGlass, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {AuthService} from '../services/auth.service';
import {DeploymentStatusNotificationConfigurationsService} from '../services/deployment-status-notification-configurations.service';
import {DeploymentStatusNotificationConfiguration} from '../types/deployment-status-notification-configuration';

@Component({
  templateUrl: './deployment-status-notification-configurations.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, DatePipe],
})
export class DeploymentStatusNotificationConfigurationsComponent {
  private readonly svc = inject(DeploymentStatusNotificationConfigurationsService);
  private readonly fb = inject(FormBuilder).nonNullable;
  protected readonly auth = inject(AuthService);

  protected readonly configs = toSignal(this.svc.list());

  protected readonly filterForm = this.fb.group({
    search: '',
  });

  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);

  protected readonly filteredConfigs = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const configs = this.configs();
    return !value ? configs : configs?.filter((it) => it.name.toLowerCase().includes(value));
  });

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;

  protected showDialog(config?: DeploymentStatusNotificationConfiguration) {}

  protected deleteConfig(config: DeploymentStatusNotificationConfiguration) {}
}
