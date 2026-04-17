import {docsLoader} from '@astrojs/starlight/loaders';
import {docsSchema} from '@astrojs/starlight/schema';
import {glob} from 'astro/loaders';
import {z} from 'astro/zod';
import {defineCollection} from 'astro:content';

export const BlogPostConfigSchema = ({image}) =>
  z.object({
    title: z.string(),
    description: z.string(),
    publishDate: z.coerce.date(),
    lastUpdated: z.coerce.date(),
    slug: z.string(),
    authors: z.array(
      z.object({
        name: z.string(),
        role: z.string(),
        image: image(),
        linkedIn: z.string(),
        gitHub: z.string(),
      }),
    ),
    image: image(),
    tags: z.array(z.string()),
  });

export const GlossaryEntryConfigSchema = z.object({
  title: z.string(),
  description: z.string(),
  slug: z.string(),
  // Optional structured-data fields. When present, the glossary page emits
  // JSON-LD (WebPage + DefinedTerm, and FAQPage) so AI Overviews and search
  // engines can cite the entry as a canonical definition.
  term: z.string().optional(),
  alternateNames: z.array(z.string()).optional(),
  shortDefinition: z.string().optional(),
  lastUpdated: z.coerce.date().optional(),
  faq: z
    .array(
      z.object({
        question: z.string(),
        answer: z.string(),
      }),
    )
    .optional(),
});

export const collections = {
  docs: defineCollection({loader: docsLoader(), schema: docsSchema()}),
  blog: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/blog'}),
    schema: BlogPostConfigSchema,
  }),
  glossary: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/glossary'}),
    schema: GlossaryEntryConfigSchema,
  }),
};

export type BlogPostConfig = z.output<ReturnType<typeof BlogPostConfigSchema>>;
export type GlossaryEntryConfig = z.output<typeof GlossaryEntryConfigSchema>;
