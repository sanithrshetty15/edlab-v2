import { Container, SectionHeading } from "../components/common";
import { ArrowUpRight, Sparkle } from "../components/common/Icons";
import { useReveal } from "../hooks/useReveal";
import { useParallax } from "../hooks/useParallax";
import { about } from "../content/labContent";
import wordmark from "../assets/images/edwins-lab-wordmark.png";
import portrait from "../assets/images/edwins-portrait.png";

export default function About() {
  const revealRef = useReveal();
  const cardRef = useParallax({ distance: 16, triggerRef: revealRef });

  return (
    <section
      ref={revealRef}
      id="about"
      aria-labelledby="about-title"
      className="bg-white py-24 sm:py-28 lg:py-36"
    >
      <Container>
        <div className="grid gap-8 lg:grid-cols-[0.92fr_1.08fr] lg:items-end lg:gap-16">
          <SectionHeading
            eyebrow={about.eyebrow}
            title={about.title}
            id="about-title"
          />
          <p
            data-reveal
            className="max-w-[52ch] text-pretty text-lg leading-8 text-edwins-dark/65 sm:text-xl sm:leading-9"
          >
            {about.intro}
          </p>
        </div>

        <div className="mt-16 grid gap-12 sm:mt-20 lg:grid-cols-[0.82fr_1.08fr] lg:gap-20">
          <div className="order-2 self-start lg:order-1 lg:sticky lg:top-28">
            <div data-reveal>
              <div ref={cardRef}>
                <article className="about-card relative overflow-hidden rounded-[26px] bg-maze-lime p-6 text-edwins-dark sm:p-8 lg:p-9">
                  <div className="about-card__grid" aria-hidden="true" />
                  <div className="relative flex items-center justify-between gap-4">
                    <span className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.2em] text-edwins-dark/72">
                      Edwins Lab / 01
                    </span>
                    <span className="flex items-center gap-2 rounded-full border border-edwins-dark/15 px-3 py-1 font-orbitron text-[0.5625rem] font-semibold uppercase tracking-[0.16em] text-edwins-dark/72">
                      <span className="h-1 w-1 rounded-full bg-edwins-dark" />
                      Open source
                    </span>
                  </div>

                  <img
                    src={wordmark}
                    alt="Edwins Lab"
                    width="554"
                    height="252"
                    loading="lazy"
                    className="relative mt-16 w-[76%] max-w-[300px]"
                  />

                  <p className="relative mt-9 font-orbitron text-[clamp(1.9rem,3vw,2.8rem)] font-semibold leading-[1.04] tracking-[-0.04em]">
                    Learn.
                    <br />
                    Build.
                    <br />
                    Innovate.
                  </p>

                  <p className="relative mt-6 max-w-[30ch] text-sm leading-6 text-edwins-dark/70">
                    {about.cardNote}
                  </p>

                  <div className="relative mt-10 flex items-center gap-3 border-t border-edwins-dark/15 pt-5">
                    <img
                      src={portrait}
                      alt="Portrait from the Edwins Lab logo"
                      width="219"
                      height="219"
                      loading="lazy"
                      className="h-12 w-12 rounded-full border border-edwins-dark/20 object-cover"
                    />
                    <div className="min-w-0">
                      <p className="text-sm font-semibold">Edwins Lab</p>
                      <p className="text-xs leading-5 text-edwins-dark/70">
                        College Open Source Technology Lab
                      </p>
                    </div>
                    <span className="ml-auto flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-edwins-dark/20">
                      <ArrowUpRight className="h-4 w-4" />
                    </span>
                  </div>
                </article>
              </div>
            </div>
          </div>

          <div className="order-1 lg:order-2">
            <div className="max-w-[58ch] space-y-6 text-[0.9375rem] leading-7 text-edwins-dark/62 sm:text-base sm:leading-8">
              {about.paragraphs.map((paragraph) => (
                <p key={paragraph} data-reveal>
                  {paragraph}
                </p>
              ))}
            </div>

            <div className="mt-10 border-t border-edwins-dark/10">
              {about.manifesto.map((item, index) => (
                <article
                  key={item.title}
                  data-reveal
                  className="group grid gap-2 border-b border-edwins-dark/10 py-5 sm:grid-cols-[92px_1fr] sm:gap-6 sm:py-6"
                >
                  <span className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.18em] text-edwins-dark/70 transition-colors duration-300 group-hover:text-edwins-dark">
                    / 0{index + 1}
                  </span>
                  <div>
                    <h3 className="text-[0.9375rem] font-semibold text-edwins-dark">
                      {item.title}
                    </h3>
                    <p className="mt-1.5 max-w-[52ch] text-sm leading-6 text-edwins-dark/70">
                      {item.description}
                    </p>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </div>

        <div
          data-reveal
          className="mt-20 flex flex-col gap-6 border-t border-edwins-dark/10 pt-10 sm:mt-28 sm:flex-row sm:items-start sm:justify-between"
        >
          <Sparkle className="h-7 w-7 shrink-0 text-maze-lime" />
          <p className="max-w-[30ch] text-balance font-orbitron text-[clamp(1.5rem,2.6vw,2.25rem)] font-medium leading-[1.2] tracking-[-0.03em] text-edwins-dark sm:max-w-[42ch]">
            The goal is not to know everything. It is to make the next
            experiment possible.
          </p>
        </div>
      </Container>
    </section>
  );
}