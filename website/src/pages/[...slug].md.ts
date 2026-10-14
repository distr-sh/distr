import type {APIRoute, GetStaticPaths} from 'astro';
import {getAgentDocs, toAgentMarkdown} from '~/utils/agent-docs';

export const getStaticPaths = (async () => {
  const entries = await getAgentDocs();
  const knownIds = new Set(entries.map(entry => entry.id));
  return entries.map(entry => ({
    params: {slug: entry.id},
    props: {markdown: toAgentMarkdown(entry, knownIds)},
  }));
}) satisfies GetStaticPaths;

export const GET: APIRoute = ({props}) =>
  new Response(props.markdown, {
    headers: {'Content-Type': 'text/markdown; charset=utf-8'},
  });
