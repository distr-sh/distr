import {Injectable} from '@angular/core';
import {captureException} from '@sentry/angular';
import {parse, parseInline, Renderer} from 'marked';

@Injectable({providedIn: 'root'})
export class MarkdownService {
  private readonly renderer = this.createRenderer();

  parse(value: string | null | undefined): string {
    return parse(value ?? '', {renderer: this.renderer, async: false});
  }

  private createRenderer(): Renderer {
    const renderer = new Renderer();

    renderer.link = ({href, text}) => {
      try {
        text = text === href ? text : parseInline(text, {async: false});
      } catch (e) {
        captureException(e);
      }
      return `<a href="${href}" target="_blank" rel="noopener noreferrer">${text}</a>`;
    };

    return renderer;
  }
}
