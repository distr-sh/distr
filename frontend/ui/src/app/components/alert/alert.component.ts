import {Component, computed, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faCircleCheck,
  faCircleInfo,
  faCircleXmark,
  faTriangleExclamation,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';

export type AlertType = 'success' | 'info' | 'warning' | 'error';

/**
 * An inline notice. The projected content is the description, an element with `alertActions` is
 * placed on the right.
 */
@Component({
  selector: 'app-alert',
  templateUrl: './alert.component.html',
  styleUrl: './alert.component.scss',
  imports: [FaIconComponent],
  host: {
    '[class]': 'typeClass()',
    '[attr.role]': 'role()',
  },
})
export class AlertComponent {
  public readonly type = input.required<AlertType>();
  public readonly heading = input<string>();
  public readonly dismissible = input(false);
  public readonly dismissed = output<void>();

  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faCircleInfo = faCircleInfo;
  protected readonly faCircleXmark = faCircleXmark;
  protected readonly faTriangleExclamation = faTriangleExclamation;
  protected readonly faXmark = faXmark;

  protected readonly typeClass = computed(() => `alert-${this.type()}`);
  protected readonly role = computed(() => (this.type() === 'warning' || this.type() === 'error' ? 'alert' : 'status'));
}
