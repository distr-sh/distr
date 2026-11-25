import {ChangeDetectionStrategy, Component, effect, input, linkedSignal, output} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Application, ApplicationVersion, ApplicationVersionResource} from '@distr-sh/distr-sdk';
import {EditorComponent} from '../components/editor.component';
import {InnerMarkdownDirective} from '../directives/inner-markdown.directive';

export interface ApplicationVersionDetail {
  application: Application;
  version: ApplicationVersion;
  linkTemplate: string;
  kubernetes?: {
    baseValues: string | null;
    template: string | null;
  };
  docker?: {
    compose: string | null;
    template: string | null;
  };
  resources: ApplicationVersionResource[];
}

@Component({
  selector: 'app-application-version-detail-modal',
  templateUrl: './application-version-detail-modal.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, EditorComponent, InnerMarkdownDirective],
})
export class ApplicationVersionDetailModalComponent {
  versionDetail = input.required<ApplicationVersionDetail>();
  closed = output<void>();
  activeTab = linkedSignal<string>(() => (this.versionDetail().application.type === 'docker' ? 'compose' : 'link'));

  versionDetailsForm = new FormGroup({
    name: new FormControl(''),
    linkTemplate: new FormControl(''),
    kubernetes: new FormGroup({
      chartUrl: new FormControl(''),
      chartName: new FormControl(''),
      chartVersion: new FormControl(''),
      baseValues: new FormControl(''),
      template: new FormControl(''),
    }),
    docker: new FormGroup({
      compose: new FormControl(''),
      template: new FormControl(''),
    }),
  });

  constructor() {
    this.versionDetailsForm.disable();

    effect(() => {
      const detail = this.versionDetail();
      this.versionDetailsForm.patchValue({
        linkTemplate: detail.linkTemplate,
        kubernetes: {
          ...detail.kubernetes,
          chartUrl: detail.version.chartUrl,
          chartName: detail.version.chartName,
          chartVersion: detail.version.chartVersion,
        },
        docker: detail.docker,
      });
    });
  }

  close() {
    this.closed.emit();
  }
}
