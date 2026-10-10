import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRight, faCircleCheck} from '@fortawesome/free-solid-svg-icons';
import {AlertComponent} from '../components/alert/alert.component';
import {PageComponent} from '../components/page.component';
import {TutorialsService} from '../services/tutorials.service';

@Component({
  selector: 'app-tutorials',
  imports: [ReactiveFormsModule, FaIconComponent, RouterLink, AsyncPipe, PageComponent, AlertComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './tutorials.component.html',
})
export class TutorialsComponent {
  protected readonly faArrowRight = faArrowRight;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly tutorialsService = inject(TutorialsService);

  ngOnInit() {
    this.tutorialsService.refreshList();
  }
}
