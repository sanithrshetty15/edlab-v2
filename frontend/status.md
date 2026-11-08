# Edwins Lab frontend status

## Current state

- Branded intro completed in `src/components/intro/Intro.jsx`: lime panel
  rises to black, reveals the `Edwins Lab` logo/wordmark, holds, then
  dissolves into the site and reveals the main content.
- Navbar restored as the second step after intro: fixed pill navigation with
  `EDWINS LAB` wordmark, About/Linux/Testimonials/Download/Contact links and
  a `Download app` CTA. The selected/active section is shown as a black pill
  (Hero shows no extra selected link), and the mobile hamburger opens a
  full-screen Maze-style lime menu.
- Hero is a stable two-column layout (desktop) with GSAP text/stage reveal
  (autoAlpha always 1; only position/scale animate) and a scroll-linked
  parallax exit. Content is always visible even if intro/animation fails.
- Interactive Tux is the author's Blender model (`Linux mascot Tux.blend`),
  exported to `public/models/tux.glb` and loaded with Three.js `GLTFLoader` in
  `Tux3D.jsx`. The GLB is scaled at runtime to a 1-unit working height, centred
  on X/Z with its feet at y = 0, and framed by a bbox-fit PerspectiveCamera
  (FOV 32, padding 1.2 including the animation budget). The asset is one rigid
  mesh with no rig, shape keys or clips, so head/eye tracking and blinking are
  not applied; idle float, cursor lean/tilt and scroll parallax remain.
  Lazy-loaded behind `TuxStage` with WebGL detection, an SVG loading state,
  error boundary and SVG fallback. Canvas fills the stage with ResizeObserver
  and capped DPR. Native GLB materials (black/white/orange Principled BSDF) are
  preserved; see `public/models/README.md` for the export recipe.
- Tux stage simplified: subtle lime halo + thin rings behind canvas only;
  cluttering labels/marks/meta removed. Stage is independent (4:5 aspect,
  overflow hidden, max-height ~70vh desktop). Hero grid fixed (copy column +
  stage column, >=3.5rem gap). No intersection with text/CTAs.
- About, Linux, testimonials, download and footer sections are present with
  GSAP ScrollTrigger reveals and respectful reduced-motion fallbacks.
- Testimonials render the designed empty state because
  `labContent.testimonials.items` is empty; no fabricated quotes are shown.
- Download CTAs are driven by `labContent.download`; empty URLs render honest
  "in preparation" actions instead of fake links.
- Core content/config is centralized in `src/content/labContent.js`.
- Brand assets are wired from `frontend/figma assets/logo/` for favicon,
  navbar wordmark, footer mark and intro.
- Explore Lab page is available at `/explore` through a lightweight custom
  router (`src/utils/router.js`). The homepage "Explore the Lab" CTA opens it,
  and the fixed Back to Home control returns to `/` without reloading. The
  homepage intro and other sections are unchanged.
- Explore includes a dark editorial hero, lab story/principles, pinned 3D
  gallery, experience cards and footer. Copy is centralized in
  `src/content/exploreContent.js`; gallery entries are in
  `src/content/exploreGallery.js`.
- The supplied lab photos are wired at `/images/lab/pic1.jpg` through
  `pic5.jpg`. `pic1` renders as a contained logo cover on a light surface; the
  other four use photo-specific crops. Titles, captions and alt text describe
  the actual images.
- The desktop gallery places the five entries on a 3D arc ring with buffer
  cards for loop previews (7 rendered cards; duplicates are `aria-hidden`).
  Card spacing is 30°, tablet sizing shrinks automatically to prevent overlap,
  mobile uses a swipe track, and reduced motion uses a static grid.
- Missing photographs still render a labelled placeholder; adding a photo
  requires one file in `public/images/lab/` plus one `labGallery` entry.

## Explore Lab improvements (Items A/B/C)

All three requested improvements to the `/explore` page are fully implemented
and verified:

- **Item A - Landscape 16:9 gallery cards**: Every card is 16:9 with
  `object-fit: cover`, per-image `object-position`, radius derived from card
  width, and `translateZ(-r)` pivot to prevent wide-side clipping. Verified at
  1440/1024/768/390/320 breakpoints with ratios 1.777–1.783 and no horizontal
  overflow. Reduced-motion falls back to a static grid.

- **Item B - Automatic slideshow in "How the Lab Works"**: Data array in
  `src/content/howItWorksSlides.js` with `SLIDE_HOLD_MS = 3500` and
  `SLIDE_TRANSITION_S = 1.15`. Two stacked GSAP layers with clip-path/opacity
  transitions, preload/decode next image, pause onscreen/offscreen and on
  hidden tab, 0 blank frames, no controls/dots, right-hand text unchanged.
  Verified constant box 594×743 at 1440, label gaps, offscreen pause/resume,
  and reduced motion static.

- **Item C - Footer contour animation**: 62 generated diagonal contour SVG
  paths with `vectorEffect="non-scaling-stroke"`, 30 fps GSAP ticker loop
  (16.6 ms average, 16.8 ms worst), IntersectionObserver pause offscreen,
  mask-reveal wordmark entrance timeline (contours 1.2 s → brand 0.8 s →
  divider 0.8 s → wordmark 1.1 s → contact/nav 0.7 s stagger → bottom 0.6 s
  stagger → legal 0.5 s), reduced motion static/visible. Verified entrance
  order, motion states, and 320 px width.

- **Lint/build**: `npm run lint` exit 0; `npm run build` exit 0, 73 modules
  transformed successfully.

## Verification

- Headless browser checks at 1440/1280/390/320: no horizontal overflow,
  console clean, all five photos decode with no fallback placeholders.
- Gallery sweep active index 01→05 at 0/25/50/75/98.5% with ring rotation
  0→~−118°.
- Slideshow: unique labels "Lab frame / 01"/"02"/"03"/"04"/"05", consistent
  timing, offscreen hold confirmed on frame "03".
- Footer: entrance progression measured across 12 frames, contours reach
  1.0 opacity at frame 12, wordmark reaches top position 7.8 px at frame 12.

## Known limitations

- Tux is a single static mesh: per-part head/eye tracking and blinking are not
  supported by the asset and are therefore not implemented. Whole-model motion
  only.
- Headless SwiftShader screenshots are functional verification; visual
  judgment should be confirmed in a GPU-capable browser.
- `public/models/Linux mascot Tux.glb` is an earlier export whose materials
  lost their PBR data; `tux.glb` supersedes it and is the file the renderer loads.
- 8% projected-bounds padding at animation extremes is enforced by
  bbox-fit + animation-budget expansion in camera math; not re-projected
  per-frame in production.
- The supplied lab photos are AVIF data despite their `.jpg` filenames.
- 8% projected-bounds padding at animation extremes.

## Open items

- Add approved real member stories to `testimonials.items` to activate the
  marquee.
- Publish real APK and PWA URLs in `labContent.js` when releases are approved.
- Optional: real demo target for hero "Watch demo" CTA.
- Confirm final gallery captions and crops with lab members, and add more
  approved photographs through `labGallery`.
- Convert or rename the AVIF-encoded `.jpg` assets before deployment if the
  hosting stack requires matching file extensions and MIME types.