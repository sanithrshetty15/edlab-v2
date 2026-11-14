export const site = {
  name: "Edwins Lab",
  tagline: "College Open Source Technology Lab",
  email: "edwin.lab.aiet@gmail.com",
  description:
    "Edwins Lab is a college open-source technology lab: a welcoming point for people who learn by exploring Linux, sharing what they discover and building in the open.",
};

export const navigation = [
  { label: "About", href: "#about" },
  { label: "Linux", href: "#linux" },
  { label: "Voices", href: "#testimonials" },
  { label: "Download", href: "#download" },
  { label: "Contact", href: "#contact" },
];

export const hero = {
  eyebrow: "College Open Source Technology Lab",
  // Each entry renders as its own Orbitron line, so the heading stays dominant
  // without relying on fixed pixel widths.
  titleLines: ["Build. Explore.", "Contribute."],
  subtitle: "Where curiosity meets open source.",
  body:
    "A student space to explore technology, test ideas and build together, then contribute what you learn back to open source.",
  chips: ["Linux", "Open source", "Hands-on"],
  // Both destinations are existing sections. "Discover Projects" currently
  // points at the lab's practice section: there is no dedicated projects
  // section yet, and no invented destination should be added.
  primaryCta: { label: "Explore the Lab", href: "/explore" },
  secondaryCta: { label: "Discover Projects", href: "#linux" },
};

export const about = {
  eyebrow: "01 / The lab",
  title: "A lab for people who learn by making.",
  intro:
    "Edwins Lab is a college open-source technology lab: a place to experiment with technology, share what you discover and build things that matter.",
  paragraphs: [
    "The lab is built around hands-on exploration and open, shared practice. Members try tools, compare approaches and keep the work visible so everyone can learn from it.",
    "The work starts with curiosity. Get comfortable with the terminal, version control and the open-source workflow, then turn that comfort into something you can show.",
  ],
  manifesto: [
    {
      title: "Hands-on first",
      description:
        "The quickest way to understand a system is to open it, run it and change it.",
    },
    {
      title: "Open by default",
      description:
        "Share the process, the dead ends and the result. That is how a lab gets better.",
    },
    {
      title: "Learn together",
      description:
        "Questions are part of the work. Asking them out loud makes the whole room smarter.",
    },
  ],
  cardNote: "A shared space for experiments, questions and the occasional broken build.",
};

export const linux = {
  eyebrow: "02 / About Linux",
  title: "Open source, in practice.",
  intro:
    "Linux is the lab's shared starting point: a free, open operating system and a way of working that rewards curiosity.",
  topics: [
    {
      id: "terminal",
      index: "01",
      title: "The terminal",
      part: "belly",
      summary:
        "Start with the command line. Small commands, clear feedback and a habit of understanding what just happened.",
    },
    {
      id: "kernel",
      index: "02",
      title: "The kernel",
      part: "head",
      summary:
        "The core of the system. Learn how hardware, processes and files fit together under the surface.",
    },
    {
      id: "community",
      index: "03",
      title: "The community",
      part: "feet",
      summary:
        "Open source is a conversation. Read the code, ask better questions and contribute something back.",
    },
  ],
  footerNote: "Free to use. Open to study. Yours to change.",
};

export const testimonials = {
  eyebrow: "03 / Voices",
  title: "The lab, in their words.",
  intro:
    "Honest notes from the people who learn and build here are being collected. Nothing on this page is scripted.",
  items: [],
  emptyTitle: "Your story belongs here.",
  emptyBody:
    "If Edwins Lab has shaped how you learn, build or share, we would like to hear it. Approved member stories will appear in this space.",
};

export const downloads = {
  eyebrow: "04 / Download",
  title: "Get Edwins Lab on your device.",
  intro:
    "Install Edwins Lab for a more convenient experience. Access the lab community directly from your device.",
  android: {
    title: "Android APK",
    description:
      "Install the Android build directly on your phone or tablet. No app store required.",
  },
  pwa: {
    title: "Install web app",
    description:
      "Install Edwins Lab straight from your browser. It opens in its own window and updates automatically.",
  },
  notifyLabel: "Get release updates",
  notifyHref:
    "mailto:edwin.lab.aiet@gmail.com?subject=Edwins%20Lab%20app%20release%20updates",
};

/*
 * Unconfirmed content
 * -------------------
 * These sections stay empty until the lab supplies verified information.
 * Do not populate them with invented dates, names, counts, awards or links.
 * See docs/CONTENT-VERIFICATION.md for the questions that are still open.
 */
export const history = {
  intro: null, // NEEDS_USER_CONFIRMATION: when and how the lab started
  milestones: [], // NEEDS_USER_CONFIRMATION: dated milestones only
};

export const coordinators = {
  intro: null, // NEEDS_USER_CONFIRMATION
  items: [], // NEEDS_USER_CONFIRMATION: { name, role, affiliation }
};

export const socialLinks = []; // NEEDS_USER_CONFIRMATION: { label, href }

export const contact = {
  email: site.email,
  institution: null, // NEEDS_USER_CONFIRMATION
  location: null, // NEEDS_USER_CONFIRMATION: campus, building or room
};

export const footer = {
  ctaTitle: "Let's build something.",
  ctaBody:
    "Questions, collaborations or a story to share? Write to the lab and start a conversation.",
  backToTop: "Back to top",
};
