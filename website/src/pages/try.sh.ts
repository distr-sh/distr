import type {APIRoute} from 'astro';
import script from '../../../deploy/docker/quickstart/try.sh?raw';

export const GET: APIRoute = () =>
  new Response(script, {
    headers: {'Content-Type': 'text/x-shellscript; charset=utf-8'},
  });
