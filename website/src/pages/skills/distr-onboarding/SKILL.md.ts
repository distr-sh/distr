import type {APIRoute} from 'astro';
import skill from '../../../../../skills/distr-onboarding/SKILL.md?raw';

export const GET: APIRoute = () =>
  new Response(skill, {
    headers: {'Content-Type': 'text/markdown; charset=utf-8'},
  });
