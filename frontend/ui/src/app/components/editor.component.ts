import {Component, ElementRef, forwardRef, inject, Input, OnDestroy, OnInit} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR} from '@angular/forms';
import {defaultKeymap, history, historyKeymap, indentWithTab} from '@codemirror/commands';
import {json} from '@codemirror/lang-json';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, indentOnInput, syntaxHighlighting} from '@codemirror/language';
import {EditorState, StateEffect} from '@codemirror/state';
import {EditorView, highlightSpecialChars, keymap} from '@codemirror/view';
import {tags} from '@lezer/highlight';
import {Subject} from 'rxjs';

export type EditorLanguage = 'yaml' | 'json';

@Component({
  selector: 'app-editor',
  template: '',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => EditorComponent),
      multi: true,
    },
  ],
})
export class EditorComponent implements OnInit, OnDestroy, ControlValueAccessor {
  @Input() language: EditorLanguage | undefined;
  private readonly host = inject(ElementRef);
  private view!: EditorView;
  private readonly destroyed$ = new Subject<void>();

  disabled = false;

  public ngOnInit(): void {
    this.view = new EditorView({
      extensions: [
        indentOnInput(),
        history(),
        syntaxHighlighting(
          HighlightStyle.define([
            // TODO: Improve highlight style
            {tag: tags.comment, class: 'italic text-gray-400'},
            {tag: tags.propertyName, class: 'text-blue-800 dark:text-blue-300'},
          ])
        ),
        highlightSpecialChars(),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        EditorView.updateListener.of((update) => {
          this.onTouched();
          if (update.docChanged) {
            this.onChange(this.view.state.doc.toString());
          }
        }),
        ...(this.language === 'yaml' ? [yaml()] : this.language === 'json' ? [json()] : []),
      ],
      parent: this.host.nativeElement,
    });
  }

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
    this.view.destroy();
  }

  writeValue(value: string) {
    const tr = this.view.state.update({changes: {from: 0, to: this.view.state.doc.length, insert: value ?? ''}});
    this.view.dispatch(tr);
  }

  registerOnChange(fn: (value: string) => void) {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void) {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean) {
    const tr = this.view.state.update({
      effects: [
        StateEffect.appendConfig.of(EditorState.readOnly.of(isDisabled)),
        StateEffect.appendConfig.of(EditorView.editable.of(!isDisabled)),
      ],
    });
    this.view.dispatch(tr);
  }

  private onChange = (_: any) => {};

  private onTouched = () => {};
}
