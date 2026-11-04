import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  forwardRef,
  inject,
  input,
  OnDestroy,
  OnInit,
} from '@angular/core';
import {ControlValueAccessor, NG_VALUE_ACCESSOR} from '@angular/forms';
import {defaultKeymap, history, historyKeymap, indentWithTab} from '@codemirror/commands';
import {json} from '@codemirror/lang-json';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, indentOnInput, StreamLanguage, syntaxHighlighting} from '@codemirror/language';
import {shell} from '@codemirror/legacy-modes/mode/shell';
import {Compartment, EditorState, Extension} from '@codemirror/state';
import {EditorView, highlightSpecialChars, keymap} from '@codemirror/view';
import {tags} from '@lezer/highlight';
import {never} from '../../util/exhaust';

export type EditorLanguage = 'yaml' | 'json' | 'shell';

@Component({
  selector: 'app-editor',
  template: '',
  host: {
    class:
      'font-mono focus-within:border-primary-600 focus-within:ring-1 focus-within:ring-primary-600 dark:focus-within:border-primary-500 dark:focus-within:ring-primary-500',
  },
  changeDetection: ChangeDetectionStrategy.Eager,
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
  // Names the editor and focuses it on click, which `<label for>` cannot do for CodeMirror's contenteditable.
  label = input<HTMLElement>();
  private readonly host = inject(ElementRef);
  private view!: EditorView;
  private readonly readOnlyCompartment = new Compartment();

  private readonly languageExtension = computed((): Extension => {
    const lang = this.language();
    switch (lang) {
      case 'yaml':
        return yaml();
      case 'json':
        return json();
      case 'shell':
        return StreamLanguage.define(shell);
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
            {tag: tags.meta, class: 'text-gray-500 dark:text-gray-400'},
            {tag: tags.keyword, class: 'text-orange-500 dark:text-orange-300'},
          ])
        ),
        highlightSpecialChars(),
        EditorView.theme({
          '&.cm-focused': {outline: 'none'},
          // `inherit` replaces the black caret of CodeMirror's light base theme with the text color.
          '.cm-content': {caretColor: 'inherit', minHeight: 'var(--editor-min-height, 100%)'},
          // Read-only content stays contenteditable, so the browser would keep drawing a blinking caret.
          '.cm-content[aria-readonly=true]': {caretColor: 'transparent'},
        }),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        EditorView.updateListener.of((update) => {
          this.onTouched();
          if (update.docChanged) {
            this.onChange(this.view.state.doc.toString());
          }
        }),
        this.languageExtension(),
        this.readOnlyCompartment.of([]),
      ],
      parent: this.host.nativeElement,
    });

    const label = this.label();
    if (label) {
      this.view.contentDOM.ariaLabelledByElements = [label];
      label.addEventListener('click', this.focus);
    }
  }

  ngOnDestroy() {
    this.label()?.removeEventListener('click', this.focus);
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
    this.view.dispatch({
      effects: this.readOnlyCompartment.reconfigure(isDisabled ? EditorState.readOnly.of(true) : []),
    });
  }

  private readonly focus = () => this.view.focus();

  private onChange = (_: any) => {};

  private onTouched = () => {};
}
