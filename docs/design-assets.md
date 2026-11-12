# etheyl design assets

The homepage and authentication identity were replaced in November 2026. Only the product name was retained as a visual commitment.

- **Unbounded** — Cyrillic and Latin variable WOFF2 subsets, self-hosted in `frontend/src/lib/assets/fonts/`. Copyright The Unbounded Project Authors. SIL Open Font License in the same directory. Source: https://github.com/googlefonts/unbounded; packaged subsets from https://www.npmjs.com/package/@fontsource-variable/unbounded.
- **Golos Text** — existing open-source font dependency used for body text and controls. Only Cyrillic and Latin WOFF2 subsets ship.
- **Coast photograph** — Megan Nixon, Unsplash: https://unsplash.com/photos/a-view-of-the-ocean-from-the-shore-of-a-beach-p6jlhK4K41g. Self-hosted at `frontend/static/images/coast.jpg`; source provenance is also embedded in its JPEG metadata. This is illustrative photography in a fictional project, not evidence of an etheyl creator or publication.
- **Creative project covers** — original HTML/CSS typography and geometric linework. “Медленные маршруты”, “не тишина”, and “А что, если?” are illustrative projects. The visible collage caption identifies them as sketches. They require no real customer data to ship.
- **Brand mark and favicon** — newly authored eight-ray geometry, independent of the prior identity. The favicon is shipped from `frontend/static/favicon.svg`; its source counterpart is `frontend/src/lib/assets/favicon.svg`.

No external image or font requests are needed at runtime. Illustrative projects must stay distinguishable from real creators when publishing and subscriptions are implemented.
