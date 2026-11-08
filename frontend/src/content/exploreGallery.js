const labPhoto = (file, meta) => ({
  id: file.replace(/\.[^.]+$/, ""),
  src: `/images/lab/${file}`,
  filename: file,
  ...meta,
});

/*
 * To add a photograph:
 * 1. Drop the file into public/images/lab/.
 * 2. Append one entry to this array (or replace a slot below).
 *
 * The desktop ring, caption counter, mobile gallery and static gallery all
 * derive their length from this array. No card count is hardcoded.
 *
 * Update `title`, `caption` and `alt` for each real photograph. The alt text
 * currently names the intended slot so it does not describe an image we have
 * not seen. If a file is missing, the image component renders a labelled
 * placeholder instead of a broken image.
 */
export const labGallery = [
  labPhoto("pic1.jpg", {
    title: "The mark",
    caption:
      "A name, a face and a place to begin.",
    alt: "The Edwin's Lab wordmark and a portrait set inside a square frame.",
    fit: "contain",
    surface: "light",
  }),
  labPhoto("pic2.jpg", {
    title: "Around one screen",
    caption:
      "A question pulls everyone in. Someone takes the keyboard and the room leans closer.",
    alt: "Students gathered around a laptop in the lab, pointing at the screen and comparing notes.",
    position: "50% 45%",
  }),
  labPhoto("pic3.jpg", {
    title: "Learning out loud",
    caption:
      "One person at the projector, a room full of questions.",
    alt: "A student presenting at a projector screen while classmates listen from their desks.",
    position: "32% 50%",
  }),
  labPhoto("pic4.jpg", {
    title: "Building together",
    caption:
      "Different skills, one shared room and one open process.",
    alt: "A group of Edwins Lab students standing together for a portrait.",
    position: "50% 38%",
  }),
  labPhoto("pic5.jpg", {
    title: "Kept visible",
    caption:
      "Experiments, notes and progress that stay open to the next person.",
    alt: "Students working together on laptops in front of a projector displaying the words Hammer Mindset.",
    position: "50% 50%",
  }),
];
