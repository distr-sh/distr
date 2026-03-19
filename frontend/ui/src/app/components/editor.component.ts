import {Component, computed, ElementRef, forwardRef, inject, input, OnDestroy, OnInit} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR} from '@angular/forms';
import {defaultKeymap, history, historyKeymap, indentWithTab} from '@codemirror/commands';
import {json} from '@codemirror/lang-json';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, indentOnInput, syntaxHighlighting} from '@codemirror/language';
import {EditorState, Extension, StateEffect} from '@codemirror/state';
import {EditorView, highlightSpecialChars, keymap} from '@codemirror/view';
import {tags} from '@lezer/highlight';
import {never} from '../../util/exhaust';

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
  language = input<EditorLanguage>();
  private readonly host = inject(ElementRef);
  private view!: EditorView;

  private readonly languageExtension = computed((): Extension => {
    const lang = this.language();
    switch (lang) {
      case 'yaml':
        return yaml();
      case 'json':
        return json();
      case undefined:
        return [];
      default:
        return never(lang);
    }
  });

  public ngOnInit(): void {
    this.view = new EditorView({
      extensions: [
        indentOnInput(),
        history(),
        syntaxHighlighting(
          HighlightStyle.define([
            {tag: tags.comment, class: 'italic text-gray-400'},
            {tag: tags.propertyName, class: 'text-blue-500 dark:text-blue-300'},
            {tag: tags.literal, class: 'text-orange-500 dark:text-orange-300'},
            {tag: tags.string, class: 'text-green-600 dark:text-green-300'},
            {tag: tags.bool, class: 'text-purple-400 dark:text-purple-300'},
            {tag: tags.punctuation, class: 'text-gray-400'},
            {tag: tags.bracket, class: 'text-orange-600 dark:text-orange-300'},
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
        this.languageExtension(),
      ],
      parent: this.host.nativeElement,
    });
  }

  ngOnDestroy() {
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
