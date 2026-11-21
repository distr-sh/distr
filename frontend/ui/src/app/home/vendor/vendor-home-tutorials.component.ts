import {Component, computed, input, output} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircle} from '@fortawesome/free-regular-svg-icons';
import {faCircleCheck, faCircleHalfStroke, faListCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {TutorialView} from '../../services/tutorials.service';

type TutorialStatus = 'open' | 'in_progress' | 'done';

@Component({
  selector: 'app-vendor-home-tutorials',
  templateUrl: './vendor-home-tutorials.component.html',
  imports: [RouterLink, FaIconComponent, CardHeadingComponent],
})
export class VendorHomeTutorialsComponent {
  public readonly tutorials = input.required<TutorialView[]>();
  public readonly dismissed = output<void>();

  protected readonly faCircle = faCircle;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faCircleHalfStroke = faCircleHalfStroke;
  protected readonly faListCheck = faListCheck;
  protected readonly faXmark = faXmark;

  protected readonly items = computed(() =>
    this.tutorials().map((tutorial) => {
      const status: TutorialStatus = tutorial.progress?.completedAt
        ? 'done'
        : tutorial.progress?.createdAt
          ? 'in_progress'
          : 'open';
      return {
        id: tutorial.id,
        name: tutorial.name,
        description: tutorial.description,
        status,
        link: status === 'done' ? tutorial.completedRoute : `/tutorials/${tutorial.id}`,
      };
    })
  );

  protected readonly doneCount = computed(() => this.items().filter((item) => item.status === 'done').length);
}
