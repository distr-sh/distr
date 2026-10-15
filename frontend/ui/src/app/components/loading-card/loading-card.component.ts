import {NgTemplateOutlet} from '@angular/common';
import {booleanAttribute, Component, input} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faSpinner} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-loading-card',
  templateUrl: './loading-card.component.html',
  imports: [NgTemplateOutlet, FaIconComponent],
})
export class LoadingCardComponent {
  public readonly heading = input<string>();
  public readonly icon = input<IconDefinition>();
  /**
   * Renders a placeholder with the dimensions of one overview list row, so the card keeps its height
   * once the list arrives. Keep it in line with the layout of `app-overview-artifact-item`, the shortest row.
   */
  public readonly row = input(false, {transform: booleanAttribute});

  protected readonly faSpinner = faSpinner;
}
