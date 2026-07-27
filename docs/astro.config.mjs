// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import cloudflare from '@astrojs/cloudflare';

// Set SITE_URL in production when a canonical URL and sitemap are desired.
const site = process.env.SITE_URL || undefined;

export default defineConfig({
  site,

  integrations: [
    starlight({
      title: 'things-cli',
      description: 'A macOS CLI for Things 3, designed for agents and scripts.',
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/thaodangspace/things-cli',
        },
      ],
      sidebar: [
        {
          label: 'Start here',
          items: [
            { label: 'Overview', slug: '' },
            { label: 'Getting started', slug: 'getting-started' },
          ],
        },
        {
          label: 'Using things-cli',
          items: [
            { label: 'Commands', slug: 'commands' },
            { label: 'Output and errors', slug: 'output-and-errors' },
            { label: 'Things automation', slug: 'things-automation' },
          ],
        },
        {
          label: 'Reference',
          items: [{ label: 'Deploy to Cloudflare Pages', slug: 'deploy' }],
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],

  adapter: cloudflare(),
});