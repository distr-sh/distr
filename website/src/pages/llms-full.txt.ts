import type {APIRoute} from 'astro';
import {getAgentDocs, toAgentMarkdown} from '~/utils/agent-docs';

export const GET: APIRoute = async () => {
  const entries = await getAgentDocs();
  const knownIds = new Set(entries.map(entry => entry.id));
  const pages = entries.map(entry => toAgentMarkdown(entry, knownIds));
  return new Response(pages.join('\n---\n\n'), {
    headers: {'Content-Type': 'text/plain; charset=utf-8'},
  });
};
