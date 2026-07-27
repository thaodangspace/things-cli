# things-cli documentation

This is the [Starlight](https://starlight.astro.build/) documentation site for
[`things-cli`](https://github.com/thaodangspace/things-cli).

## Local development

```bash
npm ci
npm run dev
```

Build and preview the production site locally with:

```bash
npm run build
npm run preview
```

The static site is written to `dist/`.

## Cloudflare Pages

Create a Pages project connected to this repository with these settings:

- **Root directory:** `docs`
- **Build command:** `npm run build`
- **Build output directory:** `dist`
- **Node.js version:** use the current supported LTS version, or set `NODE_VERSION` in the Pages environment variables

Cloudflare Pages installs dependencies from `package-lock.json` before running the
build. No Cloudflare adapter or runtime functions are needed because Astro emits
static HTML. Set `SITE_URL` to the canonical Pages URL if you want Astro to emit
a sitemap; it is optional for deployment.

If the Pages project must use the repository root as its root directory, use:

- **Build command:** `npm --prefix docs ci && npm --prefix docs run build`
- **Build output directory:** `docs/dist`
