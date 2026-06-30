import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, effect, input} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {DeploymentRevisionResponse, DeploymentTarget} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faTriangleExclamation, faXmark} from '@fortawesome/free-solid-svg-icons';
import {fromBase64} from '../../../util/encoding';
import {OrganizationKindPipe} from '../../../util/organization-kind';
import {EditorComponent} from '../../components/editor.component';

@Component({
  selector: 'app-deployment-revision-details',
  templateUrl: './deployment-revision-details.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, EditorComponent, FaIconComponent, DatePipe, OrganizationKindPipe],
})
export class DeploymentRevisionDetailsComponent {
  public readonly revision = input.required<DeploymentRevisionResponse>();
  public readonly deploymentTarget = input.required<DeploymentTarget>();

  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  protected readonly isKubernetes = computed(() => this.deploymentTarget().type === 'kubernetes');

  protected readonly valuesControl = new FormControl({value: '', disabled: true});
  protected readonly envControl = new FormControl({value: '', disabled: true});

  constructor() {
    effect(() => {
      const revision = this.revision();
      this.valuesControl.setValue(revision.valuesYaml ? fromBase64(revision.valuesYaml) : '');
      this.envControl.setValue(revision.envFileData ? fromBase64(revision.envFileData) : '');
    });
  }
}
