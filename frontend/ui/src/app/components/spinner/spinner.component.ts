import {ChangeDetectionStrategy, Component} from '@angular/core';

@Component({
  selector: 'app-spinner',
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './spinner.component.svg',
})
export class SpinnerComponent {}
