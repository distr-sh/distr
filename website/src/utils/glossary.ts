import {getCollection, type RenderedContent} from 'astro:content';
import type {GlossaryEntryConfig} from '~/content.config.ts';

export type GlossaryEntry = {
  data: GlossaryEntryConfig;
  rendered: RenderedContent;
};

const load = async function (): Promise<Array<GlossaryEntry>> {
  return (await getCollection('glossary'))
    .map(
      entry =>
        ({
          data: entry.data as GlossaryEntryConfig,
          rendered: entry.rendered! as RenderedContent,
        }) as GlossaryEntry,
    )
    .sort((a, b) => {
      // Sort alphabetically by title
      return a.data.title.localeCompare(b.data.title);
    });
};

let _entries: Array<GlossaryEntry>;

export const getSortedGlossaryEntries = async (): Promise<
  Array<GlossaryEntry>
> => {
  if (!_entries) {
    _entries = await load();
  }

  return _entries;
};
