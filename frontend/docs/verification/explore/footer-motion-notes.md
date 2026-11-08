# Reference footer motion notes

Source: `public/images/lab/comat.webm`
(VP9, 899 × 771, 60 fps, 11.164 s, 1.8 MB). Frames extracted at 0.5 s
intervals into `reference/frames/`; the contact sheet is
`reference/footer-contact.jpg`.

## What the video shows

- The footer fills the whole frame for the entire clip. The content
  (wordmark, navigation, bottom rows, legal line) is **static**: it is fully
  visible at t = 0 and does not move or reveal later.
- Only the background contour field moves. There is **no entrance sequence**
  in this clip, so the entrance timings come from the brief's specified
  defaults (documented below), not from the video.
- No brand artwork or text is copied; only the motion/geometry is used.

## Measured line geometry

Measured with `reference/analyze_lines.py` (structure-tensor orientation,
column peak detection, band contrast).

| Property | Measurement | Value used |
| --- | --- | --- |
| Base direction | dominant tangent −31° to −45° above horizontal (rising left → right), median −36° | 36° |
| Vertical gap at x = 450 | 19–26 px at 899 px wide, median 21 px | — |
| Perpendicular spacing | ≈ 17 px at 899 px wide (≈ 1.9 % of width) | 34 viewBox units (≈ 19 px at 899 px rendered, 1440 viewBox) |
| Line count over the frame | ≈ 45–60 visible | 64 generated, edges masked |
| Stroke width | ≈ 1 px at 899 px | 1 px, `non-scaling-stroke` |
| Line colour | neutral, slightly warm off-white; line luminance ≈ 0.11–0.14 vs background ≈ 0.06 | `rgba(235, 238, 232, 0.14)` |
| Background | median RGB ≈ (19, 15, 16) | existing footer colour kept (`#050607`) |
| Curvature | local tangent varies by roughly ±8–12°; line gap compresses from ≈ 20 px to ≈ 33 px at some phases | sine amplitude 24 units, wavelength 1050 units |

The field is not a set of straight stripes: the lines bow, and the bow travels
along the field. Lines fade out toward the top and bottom edges under a dark
vignette; contrast is lowest at the outer thirds.

## Measured motion

- 2D phase correlation between 0.5 s frames gives shifts of roughly 5–12 px,
  but the direction is inconsistent because the pattern undulates rather than
  translating rigidly.
- Normalised correlation of the clean central crop against t = 0 falls from
  1.00 to ≈ 0.00 after 0.5 s and never recovers above 0.1 anywhere in the
  11 s clip: the pattern never completes a visible repeat inside the video.
- Perceived speed is slow: a line travels a fraction of one spacing per
  second. One loop is therefore taken as **14 s** (within the brief's 10–20 s
  range; the video is too short to measure the period directly).
- No separate parallax layers are distinguishable. Depth is reproduced by
  phase-offsetting each line (≈ 0.35 rad per line) so the wave travels
  across the field instead of moving as one sheet.

## Entrance sequence used (brief defaults; not visible in the video)

| Step | Timing / easing |
| --- | --- |
| Contour field fades in | 0 → 1.2 s, `power2.out`, starts at 0 |
| Brand mark / divider | 0 → 0.8 s, scale 0.96 → 1 / scaleX 0 → 1, `power3.out` |
| Wordmark mask reveal | starts 0.35 s, 1.1 s, `yPercent 105 → 0`, `power4.out` |
| Navigation + contact details | starts 0.75 s, 0.7 s each, stagger 0.08, `y 24 → 0`, `power3.out` (overlaps wordmark) |
| Copyright / bottom row | starts 1.15 s, 0.5 s rise/fade |
| Legal line | starts 1.4 s, 0.5 s rise/fade |

All steps play once, triggered at `top 80%`, no pin, no scrub. Reduced motion
shows the static contour frame and all content immediately.

## Implementation choice

A `gsap.ticker` callback recomputes each path's `d` from a sine wave and writes
it with `setAttribute`, capped at 30 updates/s. This gives a genuinely
travelling wave (the reference's lines change shape, not just position) with
no React state updates. It pauses when the footer is outside the viewport
(ScrollTrigger `onToggle`) or the tab is hidden. Under reduced motion the
paths render a single static frame and the ticker is never attached.
