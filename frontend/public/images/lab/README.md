# Lab photographs

Place real Edwins Lab photographs in this directory. The Explore page reads
their filenames from `src/content/exploreGallery.js`.

The initial slots expect:

- `pic1.jpg`
- `pic2.jpg`
- `pic3.jpg`
- `pic4.jpg`
- `pic5.jpg`

## Adding a photograph

1. Copy the file into this directory.
2. Open `src/content/exploreGallery.js` and add one entry:

```js
labPhoto("pic6.jpg", {
  title: "A short title",
  caption: "A short caption.",
  alt: "A description of what is actually visible in the photograph.",
}),
```

The desktop ring, caption counter, mobile gallery and reduced-motion gallery
derive their length from the `labGallery` array, so no card count needs editing.

If a photograph uses a different extension, change the filename in the entry
(for example `pic6.png` or `pic7.webp`).

Until a file is added, the page renders a labelled placeholder for that slot.
No stock photography is used.
