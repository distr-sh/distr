import {computed, Directive, inject, input, SecurityContext} from '@angular/core';
import {DomSanitizer} from '@angular/platform-browser';
import {MarkdownService} from '../services/markdown.service';

@Directive({
  selector: '[innerMarkdown]',
  host: {'[innerHTML]': 'safeHtml()'},
})
export class InnerMarkdownDirective {
  private readonly markdown = inject(MarkdownService);
  private readonly sanitizer = inject(DomSanitizer);

  innerMarkdown = input<string>('');

  protected safeHtml = computed(() =>
    this.sanitizer.sanitize(SecurityContext.HTML, this.markdown.parse(this.innerMarkdown()))
  );
}
