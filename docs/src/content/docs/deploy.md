---
title: Deploy to Cloudflare Pages
description: Publish the static things-cli documentation site with Cloudflare Pages.
---

The docs site is an Astro static build. It does not need a server adapter,
Cloudflare Worker, or runtime functions. An environment variable is optional:
set `SITE_URL` to the canonical public URL to enable sitemap generation.

## Configure a Pages project

Connect the repository to Cloudflare Pages and use these settings:

| Setting | Value |
| --- | --- |
| Root directory | `docs` |
| Build command | `npm run build` |
| Build output directory | `dist` |

Cloudflare Pages installs the dependencies from `docs/package-lock.json`, runs
Astro, and publishes the generated `dist/` directory.

Use a supported Node.js LTS version for the build. Cloudflare Pages lets you set
this with the `NODE_VERSION` environment variable when the default version does
not match the project’s supported runtime.

## Verify before publishing

Run the same install and build locally:

```sh
cd docs
npm ci
npm run build
```

To inspect the result locally:

```sh
npm run preview
```

The repository also provides a root-level shortcut:

```sh
make docs-build
```

:::tip[Repository-root alternative]
If the Pages project must keep the repository root as its root directory, use
`npm --prefix docs ci && npm --prefix docs run build` as the build command and
`docs/dist` as the output directory.
:::
