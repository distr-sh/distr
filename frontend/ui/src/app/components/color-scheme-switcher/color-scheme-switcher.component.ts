import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faMoon, faSun} from '@fortawesome/free-solid-svg-icons';
import {ColorSchemeService} from '../../services/color-scheme.service';

@Component({
  selector: 'app-color-scheme-switcher',
  standalone: true,
  templateUrl: './color-scheme-switcher.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent],
})
export class ColorSchemeSwitcherComponent {
  private colorSchemeService = inject(ColorSchemeService);
  public colorSchemeSignal = this.colorSchemeService.colorScheme;

  protected readonly faSun = faSun;
  protected readonly faMoon = faMoon;

  constructor() {}

  switchColorScheme() {
    let newColorScheme: 'dark' | '' = 'dark';
    if ('dark' === this.colorSchemeSignal()) {
      newColorScheme = '';
    }
    this.colorSchemeSignal.set(newColorScheme);
  }
}
