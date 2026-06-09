import {docsLoader} from '@astrojs/starlight/loaders';
import {docsSchema} from '@astrojs/starlight/schema';
import {glob} from 'astro/loaders';
import {z} from 'astro/zod';
import {defineCollection, type SchemaContext} from 'astro:content';

export const BlogPostConfigSchema = ({image}: SchemaContext) =>
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

export const CustomerConfigSchema = ({image}: SchemaContext) =>
  z.object({
    company: z.string(),
    person: z.object({
      name: z.string(),
      role: z.string(),
      image: image(),
    }),
    quote: z.string(),
    industry: z.string(),
    useCase: z.string(),
    // Homepage testimonial control
    featured: z.boolean().default(false),
    outcome: z.string().optional(), // short testimonial headline
    // Present => rendered at /case-studies/<slug>
    caseStudy: z
      .object({
        logo: image(),
        logoLight: image().optional(),
        logoDark: image().optional(),
        pageTitle: z.string(),
        pageDescription: z.string(),
      })
      .optional(),
  });

export const collections = {
  docs: defineCollection({
    loader: docsLoader(),
    schema: docsSchema({
      extend: z.object({
        hideSidebar: z.boolean().default(false),
      }),
    }),
  }),
  blog: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/blog'}),
    schema: BlogPostConfigSchema,
  }),
  glossary: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/glossary'}),
    schema: GlossaryEntryConfigSchema,
  }),
  customers: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/customers'}),
    schema: CustomerConfigSchema,
  }),
};

export type BlogPostConfig = z.output<ReturnType<typeof BlogPostConfigSchema>>;
export type GlossaryEntryConfig = z.output<typeof GlossaryEntryConfigSchema>;
export type CustomerConfig = z.output<ReturnType<typeof CustomerConfigSchema>>;
