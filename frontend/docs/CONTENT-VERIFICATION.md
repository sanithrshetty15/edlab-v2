# Edwins Lab — content verification

Internal audit record. **Not** published on the website.

Audit date: 6 November 2026
Scope: every visitor-facing string in `src/`, `index.html` metadata, all links
and actions, and the content model in `src/content/labContent.js`.

Classification: **A** verified · **B** needs confirmation · **C** incorrect
(corrected) · **D** placeholder / invented-sounding copy.

The lab's official identity could not be independently verified: a web search
returned no official Edwins Lab source. The one near-match
(`https://www.aiet.org.in/clubs-activities/edwin-linux-lab/`, Alva's Institute
of Engineering & Technology, "Edwin Linux Lab") uses a **different name** and is
not treated as evidence. Everything below therefore rests on the internal
project information only, and identity fields are marked **B** until the lab
confirms them.

## 1. Audit table

| # | Section | Existing statement | Class | Evidence | Required action |
|---|---|---|---|---|---|
| 1 | Brand | `Edwins Lab` (`site.name`, wordmark, footer copyright) | B | Internal project name only; no public official source | Confirm exact official name and whether "Edwins OS Lab" is used |
| 2 | Brand | `College Open Source Technology Lab` (tagline, eyebrow) | B | Internal positioning; no public official source | Confirm official tagline |
| 3 | Brand | Email `edwin.lab.aiet@gmail.com` | B | New official lab contact and mailer address | Confirm the address is owned and may be published |
| 4 | Metadata | `<title>` "Edwins Lab — College Open Source Technology Lab" | B | Follows #1/#2 | Update if the official name differs |
| 5 | Metadata | Meta description ("college open-source technology lab…") | B | Restates internal positioning; no factual claim | Confirm or supply the official description |
| 6 | Metadata | Favicon `href="../../assets/images/edwins-lab-logo.png"` with `type="image/svg+xml"` | **C** | Request resolved to `text/html` (SPA fallback) — the icon never loaded; mismatched MIME type; `public/favicon.svg` was unused | **Fixed**: now `/favicon.svg` (correct type and asset) |
| 7 | Hero | Eyebrow `College Open Source Technology Lab` | B | Same as #2 | Confirm |
| 8 | Hero | Title `About Edwins Lab` / `Welcoming Point.` | D | Generic marketing phrasing; reads as an About heading inside the hero | Rewrite only with a confirmed one-line description of the lab |
| 9 | Hero | Body "A welcoming point for curious minds…" | D | Generic copy; no verifiable claim | Acceptable as positioning; keep or replace with confirmed wording |
| 10 | Hero | Chips `Linux`, `Open source`, `Hands-on` | B | Implies the lab's focus areas | Confirm these describe actual activity |
| 11 | Hero | CTA **"Watch demo"** → `#linux` with a play icon | **C** | No demo video or demo build exists anywhere in the project | **Fixed**: label now "Explore Linux", Terminal icon, same `#linux` target |
| 12 | Hero | CTA "Explore the lab" → `#about` | A | `#about` section exists | — |
| 13 | Hero | Foot meta `Edwins Lab / Open source practice` | B | Internal | Confirm |
| 14 | About | Title "A lab for people who learn by making." | D | Generic | Keep or replace with confirmed wording |
| 15 | About | Intro "Edwins Lab is a college open-source technology lab…" | B | Internal | Confirm institution type and lab nature |
| 16 | About | "…a shared **frontend foundation**" | **C** | Unsupported technical specificity: nothing in the project establishes that the lab is frontend-focused | **Fixed**: "open, shared practice" |
| 17 | About | "Members try tools, compare approaches…" | B | Asserts activities and membership | Confirm; no member count claimed (good) |
| 18 | About | "…terminal, version control and the open-source workflow" | B | Asserts curriculum/activities | Confirm |
| 19 | About | Manifesto: Hands-on first / Open by default / Learn together | D | Values statements, not verifiable facts | Acceptable as stated values once the lab approves them |
| 20 | About | Card "Learn. Build. Innovate." | B | Motto; also used in the menu overlay | Confirm it is the lab's motto |
| 21 | About | Card "Edwins Lab / 01", portrait alt "Portrait from the Edwins Lab logo" | D | Decorative labels | — |
| 22 | About | "A shared space for experiments, questions and the occasional broken build." | D | Generic | Optional rewrite |
| 23 | About | Closing "The goal is not to know everything…" | D | Opinion, not a fact | — |
| 24 | History | *No history, founding date, timeline or founder is shown* | A | Absence is correct | Do **not** add until supplied — see `history` in `labContent.js` |
| 25 | Linux | "Linux is the lab's shared starting point" | B | Claim about lab practice | Confirm |
| 26 | Linux | Topic copy: terminal, kernel, community | A | General, accurate Linux education; no lab-specific claim | — |
| 27 | Linux | "Free to use. Open to study. Yours to change." | A | General truth about free/open-source software | — |
| 28 | Linux | Stage labels "Tux", "Terminal ready", `$ learn --build --share` | D | Decorative UI | — |
| 29 | Testimonials | Empty state: no quotes, "Your story belongs here." | A | `testimonials.items` is empty; nothing fabricated | Populate only with approved, attributed stories |
| 30 | Testimonials | "Honest notes … are being collected. Nothing on this page is scripted." | B | Sounds like content exists; none does | Reword when items are added |
| 31 | Downloads | "The Edwins Lab app is in preparation" + "In preparation" badges | A | `apkUrl`/`pwaUrl` are `null`; UI gates the buttons | Confirm there is no released build |
| 32 | Downloads | "No release dates are promised until the builds are ready." | A | Accurate to current state | — |
| 33 | Downloads | "Get release updates" → `mailto:` | B | Depends on #3 | Confirm mailbox |
| 34 | Footer | "Write to the lab" → `mailto:` | B | Depends on #3 | Confirm |
| 35 | Footer | `© {year} Edwins Lab` | B | Copyright owner unverified | Confirm the legal/owning entity |
| 36 | Footer | No coordinator, campus, location or social links shown | A | Absence is correct | Supply confirmed data before adding |
| 37 | Navigation | About / Linux / Voices / Download / Contact anchors | A | All five target sections exist (`#about`, `#linux`, `#testimonials`, `#download`, `#contact`) | Keep labels in sync with section headings |
| 38 | Navigation | "Download app" CTA → `#download` | A | Target exists; no fake file URL | — |
| 39 | Footer | "Back to top" → `#top` | A | `#top` exists on the app root | — |
| 40 | Social / GitHub | *No social or GitHub links are rendered* | A | Nothing invented | Add only verified profiles |
| 41 | Menu overlay | "Learn. Build. Innovate." | B | Same as #20 | Confirm |

**Totals:** 41 items audited · 12 verified (A) · 20 need confirmation (B) ·
4 incorrect/corrected (C) · 12 placeholder or generic (D, overlapping B where
copy also needs confirmation).

## 2. Incorrect information removed or corrected

| Fix | File | Why |
|---|---|---|
| "Watch demo" CTA → "Explore Linux" (+ Terminal icon) | `src/sections/Hero.jsx` | The control promised a demo that does not exist; it now describes where it really goes |
| "…a shared frontend foundation" → "…open, shared practice" | `src/content/labContent.js` | Removed an unsupported technical claim about the lab's focus |
| Broken favicon link → `/favicon.svg` | `index.html` | The old path resolved to HTML and never displayed; MIME type now matches |

No statistics, dates, member counts, awards, testimonials, partnerships or
download URLs were present to remove — the earlier implementation already gated
those.

## 3. Information required from the user

Please supply, only where it is true and safe to publish:

1. Official name: "Edwins Lab", "Edwins OS Lab", or both (and which is primary)?
2. Official institution, campus and location.
3. Official contact email (is `edwin.lab.aiet@gmail.com` correct and publishable?).
4. Verified GitHub organisation / repository URL.
5. Verified social profiles (none are currently linked).
6. Founding year and any dated milestones — or explicit confirmation that no
   history should be shown.
7. Coordinator / faculty advisor names and roles.
8. A one-sentence description of what the lab actually does, for the hero.
9. Whether "Explore Linux", "Open source", "Hands-on", "Learn. Build.
   Innovate." match the lab's own language.
10. Any genuine, attributed member testimonial.
11. Release status of the Android APK and the PWA (currently both "in
    preparation" with no URLs).

## 4. Missing content structures

`src/content/labContent.js` now exposes `history`, `coordinators`,
`socialLinks` and `contact` with empty values. Nothing renders from them yet, so
no page can leak an invented fact. Populate them **only** with confirmed data;
the sections that will use them are already guarded against missing values.

## 5. External sources consulted

- No official Edwins Lab source was found.
- `aiet.org.in` "Edwin Linux Lab" — different name, **not** used as evidence.
- Linux facts in the Linux section are general and verifiable independently
  (free/open-source licensing), not claims about the lab.

## 6. Suggested improvements (pending confirmation)

- Replace the hero headline ("About Edwins Lab / Welcoming Point.") with a
  factual one-liner once #8 is answered.
- Add a `history` block only if dated milestones exist; skip it entirely
  otherwise.
- Link a real demo when one exists, and restore a "Watch demo" label only then.
- Add one verified testimonial rather than a marquee of generic praise.
- Point the "Get release updates" CTA at a maintained channel instead of a
  personal mailbox, if one exists.

## 7. Verification after the changes

- `npm run lint` — 0 errors.
- `npm run build` — production build succeeds (Tux chunk 626 kB / 157 kB gzip).
- Favicon request now returns `image/svg+xml`.
- All anchor destinations exist; no external links were added.
- Testimonials empty state and download "in preparation" states still render.
