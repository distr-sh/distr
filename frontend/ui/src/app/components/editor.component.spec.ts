import {json} from '@codemirror/lang-json';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, LanguageSupport} from '@codemirror/language';
import {highlightTree, tags} from '@lezer/highlight';

// Regression test for a duplicated @lezer/common in the dependency tree. When the
// CodeMirror language packages resolve two copies of @lezer/common, the style-tag
// NodeProp id differs between the copy that stores the tags and the one that reads
// them, so highlightTree crashes with:
//   TypeError: can't access property Symbol.iterator, tags is undefined
// (thrown from @lezer/highlight). This test drives the exact code path that failed
// in the editor (highlightTree over the real YAML/JSON parse trees) so the error
// can never silently come back through a dependency bump.

// Mirrors the HighlightStyle configured in EditorComponent.
const highlightStyle = HighlightStyle.define([
  {tag: tags.comment, class: 'italic text-gray-400'},
  {tag: tags.propertyName, class: 'text-blue-500 dark:text-blue-300'},
  {tag: tags.literal, class: 'text-orange-500 dark:text-orange-300'},
  {tag: tags.string, class: 'text-green-600 dark:text-green-300'},
  {tag: tags.bool, class: 'text-purple-400 dark:text-purple-300'},
  {tag: tags.punctuation, class: 'text-gray-400'},
  {tag: tags.bracket, class: 'text-orange-600 dark:text-orange-300'},
]);

function countHighlightedRanges(language: LanguageSupport, doc: string): number {
  const tree = language.language.parser.parse(doc);
  let count = 0;
  highlightTree(tree, highlightStyle, () => count++);
  return count;
}

describe('editor syntax highlighting', () => {
  it('highlights YAML without crashing', () => {
    expect(countHighlightedRanges(yaml(), 'foo: bar\nbaz: 123\nqux: true\n# comment\n')).toBeGreaterThan(0);
  });

  it('highlights JSON without crashing', () => {
    expect(countHighlightedRanges(json(), '{"foo": "bar", "count": 123, "enabled": true}')).toBeGreaterThan(0);
  });
});
