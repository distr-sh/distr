import {inject, Pipe, PipeTransform} from '@angular/core';
import {SafeHtml} from '@angular/platform-browser';
import {MarkdownService} from '../services/markdown.service';

@Pipe({name: 'markdown'})
export class MarkdownPipe implements PipeTransform {
  private readonly markdown = inject(MarkdownService);

  transform(value: string | null | undefined): SafeHtml {
    return this.markdown.parse(value);
  }
}
