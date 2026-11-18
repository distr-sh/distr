import {Component, input} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {PodSchedulingFormGroup} from '../../../util/pod-scheduling';

@Component({
  selector: 'app-pod-scheduling-fields',
  imports: [ReactiveFormsModule],
  host: {class: 'flex flex-col gap-4 rounded-md border border-gray-300 p-4 dark:border-gray-700'},
  templateUrl: './pod-scheduling-fields.component.html',
})
export class PodSchedulingFieldsComponent {
  public readonly group = input.required<PodSchedulingFormGroup>();
}
