import {Component, computed, effect, inject, input, linkedSignal, signal} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule} from '@angular/forms';
import {firstValueFrom, map, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ContextService} from '../services/context.service';
import {ControllerImageRegistryService} from '../services/controller-image-registry.service';
import {OrganizationService} from '../services/organization.service';
import {ToastService} from '../services/toast.service';
import {ControllerImageRegistry} from '../types/controller-image-registry';

@Component({
  selector: 'app-controller-image-registry',
  templateUrl: './controller-image-registry.component.html',
  imports: [ReactiveFormsModule, AutotrimDirective],
})
export class ControllerImageRegistryComponent {
  private readonly service = inject(ControllerImageRegistryService);
  private readonly toast = inject(ToastService);
  private readonly contextService = inject(ContextService);
  private readonly organizationService = inject(OrganizationService);

  public readonly customerOrganizationId = input<string>();

  private readonly fetched = toSignal(
    toObservable(this.customerOrganizationId).pipe(switchMap((id) => this.service.get(id)))
  );
  protected readonly state = linkedSignal<ControllerImageRegistry | undefined>(() => this.fetched());

  private readonly registryHost = toSignal(this.contextService.getRegistryHost());
  private readonly organizationSlug = toSignal(this.organizationService.get().pipe(map((org) => org.slug)));
  protected readonly exampleRegistry = computed(
    () => `${this.registryHost() || 'registry.distr.sh'}/${this.organizationSlug() || 'your-org'}`
  );
  protected readonly saving = signal(false);

  protected readonly control = inject(FormBuilder).nonNullable.control('');
  private readonly value = toSignal(this.control.valueChanges, {initialValue: this.control.value});
  protected readonly unchanged = computed(() => this.value().trim() === (this.state()?.imageRegistry ?? ''));

  constructor() {
    effect(() => this.control.reset(this.state()?.imageRegistry ?? ''));
  }

  protected async save(event: Event) {
    // Not (ngSubmit): that would need FormsModule to attach an NgForm to a form whose only control is reactive.
    event.preventDefault();
    this.saving.set(true);
    try {
      const state = await firstValueFrom(
        this.service.update({
          imageRegistry: this.control.value,
          customerOrganizationId: this.customerOrganizationId(),
        })
      );
      this.state.set(state);
      this.toast.success(state.imageRegistry ? 'Controller registry saved' : 'Controller registry reset');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
    }
  }
}
