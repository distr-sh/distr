import {computed, Directive, input} from '@angular/core';
import {PLACEHOLDER_EMAIL, PLACEHOLDER_NAME} from '../../constants';

const placeholderMap = {
  name: PLACEHOLDER_NAME,
  email: PLACEHOLDER_EMAIL,
};

@Directive({
  selector: '[appPlaceholder]',
  host: {'[attr.placeholder]': 'placeholder()'},
})
export class PlaceholderDirective {
  public readonly kind = input.required<keyof typeof placeholderMap>({alias: 'appPlaceholder'});
  protected readonly placeholder = computed(() => placeholderMap[this.kind()]);
}
