import type {RenderResult} from 'astro:content';
import type {BlogPostConfig} from './content.config';

export interface Post {
  data: BlogPostConfig;
  rendered: Promise<RenderResult>;
}

export type PostData = BlogPostConfig;
