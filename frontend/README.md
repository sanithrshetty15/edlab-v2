# Edwins Lab — Website Frontend

The official Edwins Lab website: a React + Vite application for the college
open-source technology lab. The design is a premium, editorial take on the
lab's identity, built around the original Edwins Lab logo, a lime and charcoal
palette, and a real-time 3D classic Tux.

## Stack

- React 19 + Vite 8
- Tailwind CSS v4 via `@tailwindcss/vite`
- GSAP + ScrollTrigger
- Three.js (lazy-loaded with the hero Tux)
- Orbitron + Poppins from Google Fonts

## Status

The site is fully designed and implemented. The approved intro animation is
preserved unchanged. Production build and ESLint pass cleanly.

## Commands

```bash
npm run dev      # start the Vite dev server
npm run build    # production build into dist/
npm run lint     # ESLint check
npm run preview  # preview the production build
```

## Structure

```
src/
├── components/
│   ├── common/       Button, Container, SectionHeading, Icons
│   ├── intro/        Preserved approved intro sequence
│   ├── layout/       Navbar and full-screen menu overlay
│   └── tux/          Tux3D, TuxStage loader, TuxArtwork SVG
├── content/          labContent.js — all site copy and links
├── hooks/            useReveal, useParallax
├── sections/         Hero, About, Linux, Testimonials, Download, Footer
├── utils/            GSAP setup, reduced-motion helper
├── App.jsx
├── App.css
├── index.css         Tailwind theme and base styles
└── main.jsx
```

## Brand tokens

Defined in `src/index.css`:

- `--color-maze-lime: #C5FF41`
- `--color-edwins-dark: #1C1C1C`
- `--color-edwins-neutral: #F4F5F0`
- `--color-edwins-paper: #FBFCF8`
- `--font-orbitron`, `--font-poppins`

The supplied logo is kept at `src/assets/images/edwins-lab-logo.png`.
Wordmark and portrait derivatives live alongside it and stay true to the
original artwork.

## Content policy

- No testimonials, member stories, dates or lab history are invented.
  `Testimonials` shows an honest prompt for real stories until approved
  submissions exist.
- Download actions have no fake links. Android APK and PWA URLs are `null`
  until builds are approved, and the UI says "in preparation".
- The contact email is the real lab address found in the backend:
  `edwinsoslab@gmail.com`.

## 3D Tux

`Tux3D.jsx` loads the authored Blender model from `public/models/tux.glb` with
Three.js `GLTFLoader`, normalises it to a 1-unit working height (feet on the
ground plane) and frames it with a bounding-box camera fit. It floats, leans and
tilts toward the cursor, responds to scroll, and disposes of its renderer,
geometries, materials, textures, observers and listeners on unmount. The
component is lazy-loaded so Three.js does not block the first paint.

The asset is a single rigid mesh with no rig, shape keys or animation clips, so
head/eye tracking and blinking are not supported and are not faked. See
`public/models/README.md` for the model's verified structure and the Blender
export recipe; `TuxStage.jsx` shows the SVG artwork while the model loads and
keeps it as the fallback if WebGL or the download fails.

`TuxArtwork.jsx` is a detailed interactive SVG used in the Linux explainer and
as the accessible fallback when WebGL is unavailable. It is not presented as
the 3D hero model.

## Accessibility and motion

- Skip link, landmarks, labelled sections, visible focus styles.
- Full-screen menu traps focus, closes on Escape and restores focus.
- Linux explainer uses the ARIA tabs pattern with keyboard navigation.
- `prefers-reduced-motion` disables reveals, parallax, marquee and 3D motion.
- The site is verified without horizontal overflow at 1440, 1280, 768, 390
  and 320 px.
