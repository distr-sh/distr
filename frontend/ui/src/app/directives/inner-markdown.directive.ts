import {computed, Directive, inject, input} from '@angular/core';
import {MarkdownService} from '../services/markdown.service';

@Directive({
  selector: '[innerMarkdown]',
  host: {'[innerHTML]': 'safeHtml()'},
})
export class InnerMarkdownDirective {
  private readonly markdown = inject(MarkdownService);

  innerMarkdown = input<string>('');

  protected safeHtml = computed(() => this.markdown.parse(this.innerMarkdown()));
}
