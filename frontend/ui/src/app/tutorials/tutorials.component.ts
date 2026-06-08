import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircle, faCircleCheck} from '@fortawesome/free-regular-svg-icons';
import {faArrowRight, faCheck, faLightbulb} from '@fortawesome/free-solid-svg-icons';
import {TutorialsService} from '../services/tutorials.service';

@Component({
  selector: 'app-tutorials',
  imports: [ReactiveFormsModule, FaIconComponent, RouterLink, AsyncPipe],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './tutorials.component.html',
})
export class TutorialsComponent {
  protected readonly faLightbulb = faLightbulb;
  protected readonly faArrowRight = faArrowRight;
  protected readonly faCheck = faCheck;
  protected readonly tutorialsService = inject(TutorialsService);

  ngOnInit() {
    this.tutorialsService.refreshList();
  }

  protected readonly faCircle = faCircle;
  protected readonly faCircleCheck = faCircleCheck;
}
