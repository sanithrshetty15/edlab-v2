import { useRef, useState } from "react";
import { Container, SectionHeading } from "../components/common";
import { ArrowUpRight, Sparkle, Terminal } from "../components/common/Icons";
import TuxArtwork from "../components/tux/TuxArtwork";
import { useReveal } from "../hooks/useReveal";
import { useParallax } from "../hooks/useParallax";
import { linux } from "../content/labContent";

export default function Linux() {
  const revealRef = useReveal();
  const stageRef = useParallax({ distance: 26, triggerRef: revealRef });
  const [activeId, setActiveId] = useState(linux.topics[0].id);
  const tabRefs = useRef([]);

  const activeIndex = linux.topics.findIndex((topic) => topic.id === activeId);
  const activeTopic = linux.topics[activeIndex] ?? linux.topics[0];

  const selectTab = (index) => {
    const next = linux.topics[index];
    if (!next) return;
    setActiveId(next.id);
    tabRefs.current[index]?.focus();
  };

  const onTabKeyDown = (event) => {
    const keys = ["ArrowDown", "ArrowRight", "ArrowUp", "ArrowLeft", "Home", "End"];
    if (!keys.includes(event.key)) return;
    event.preventDefault();

    if (event.key === "Home") {
      selectTab(0);
      return;
    }
    if (event.key === "End") {
      selectTab(linux.topics.length - 1);
      return;
    }

    const direction =
      event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : -1;
    const nextIndex =
      (activeIndex + direction + linux.topics.length) % linux.topics.length;
    selectTab(nextIndex);
  };

  return (
    <section
      ref={revealRef}
      id="linux"
      aria-labelledby="linux-title"
      className="linux relative overflow-hidden bg-edwins-dark py-24 text-white sm:py-28 lg:py-36"
    >
      <div className="linux__grid" aria-hidden="true" />
      <div className="linux__glow" aria-hidden="true" />

      <Container className="relative z-10">
        <div className="grid gap-8 lg:grid-cols-[1fr_0.8fr] lg:items-end lg:gap-16">
          <SectionHeading
            eyebrow={linux.eyebrow}
            title={linux.title}
            id="linux-title"
            dark
          />
          <p
            data-reveal
            className="max-w-[48ch] text-pretty text-[0.9375rem] leading-7 text-white/60 sm:text-base"
          >
            {linux.intro}
          </p>
        </div>

        <div className="mt-14 grid gap-12 sm:mt-20 lg:grid-cols-[0.86fr_1.14fr] lg:items-center lg:gap-16">
          <div data-reveal className="order-2 lg:order-1">
            <div
              role="tablist"
              aria-label="Linux topics"
              aria-orientation="vertical"
              className="border-t border-white/12"
            >
              {linux.topics.map((topic, index) => {
                const selected = topic.id === activeId;
                return (
                  <button
                    key={topic.id}
                    ref={(element) => {
                      tabRefs.current[index] = element;
                    }}
                    type="button"
                    role="tab"
                    id={`linux-tab-${topic.id}`}
                    aria-selected={selected}
                    aria-controls={`linux-panel-${topic.id}`}
                    tabIndex={selected ? 0 : -1}
                    onClick={() => setActiveId(topic.id)}
                    onKeyDown={onTabKeyDown}
                    className={`group flex w-full items-center gap-4 border-b border-white/12 py-5 text-left transition-colors duration-300 sm:gap-6 sm:py-6 ${
                      selected ? "text-white" : "text-white/55 hover:text-white"
                    }`}
                  >
                    <span
                      className={`font-orbitron text-[0.625rem] font-semibold tracking-[0.16em] transition-colors duration-300 ${
                        selected ? "text-maze-lime" : "text-white/40"
                      }`}
                    >
                      {topic.index}
                    </span>
                    <span className="font-orbitron text-xl font-medium tracking-[-0.02em] sm:text-2xl">
                      {topic.title}
                    </span>
                    <span
                      className={`ml-auto flex h-8 w-8 shrink-0 items-center justify-center rounded-full border transition-all duration-300 ${
                        selected
                          ? "rotate-45 border-maze-lime bg-maze-lime text-edwins-dark"
                          : "border-white/15 text-white/40 group-hover:border-white/35 group-hover:text-white"
                      }`}
                      aria-hidden="true"
                    >
                      <span className="text-lg leading-none">+</span>
                    </span>
                  </button>
                );
              })}
            </div>

            <div
              role="tabpanel"
              id={`linux-panel-${activeTopic.id}`}
              aria-labelledby={`linux-tab-${activeTopic.id}`}
              className="mt-7 min-h-[104px] max-w-[46ch]"
            >
              <p
                key={activeTopic.id}
                className="tux-panel-copy text-[0.9375rem] leading-7 text-white/60 sm:text-base"
              >
                {activeTopic.summary}
              </p>
            </div>

            <div className="mt-8 inline-flex items-center gap-3 border-t border-white/12 pt-5 font-orbitron text-[0.625rem] font-medium uppercase tracking-[0.18em] text-white/48">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-maze-lime" />
              {linux.footerNote}
            </div>
          </div>

          <div data-reveal className="order-1 lg:order-2">
            <div ref={stageRef}>
              <div
                className="linux-stage relative mx-auto w-full max-w-[560px]"
                data-tux-tracking
              >
                <div className="linux-stage__halo" aria-hidden="true" />
                <div className="linux-stage__ring" aria-hidden="true" />
                <div className="linux-stage__ring linux-stage__ring--small" aria-hidden="true" />
                <div className="linux-stage__dots" aria-hidden="true" />

                <TuxArtwork
                  className="relative z-10 mx-auto w-[78%] max-w-[400px] drop-shadow-[0_30px_50px_rgba(0,0,0,0.28)]"
                  activePart={activeTopic.part}
                  interactive
                  title={`Classic Tux illustration, ${activeTopic.title.toLowerCase()} highlighted`}
                />

                <div className="linux-stage__label linux-stage__label--top">
                  <Sparkle className="h-3.5 w-3.5" />
                  Tux
                </div>
                <div className="linux-stage__label linux-stage__label--bottom">
                  <Terminal className="h-3.5 w-3.5" />
                  Terminal ready
                </div>

                <div className="linux-stage__prompt" aria-hidden="true">
                  <span className="text-maze-lime">$</span> learn --build --share
                  <span className="linux-caret" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <div
          data-reveal
          className="mt-16 flex flex-wrap items-center justify-between gap-4 border-t border-white/12 pt-6 font-orbitron text-[0.625rem] font-medium uppercase tracking-[0.18em] text-white/55 sm:mt-20"
        >
          <span>Linux / Open source / Practice</span>
          <a
            href="#testimonials"
            className="group inline-flex items-center gap-2 text-white/65 transition-colors hover:text-maze-lime"
          >
            Next: voices
            <ArrowUpRight className="h-3.5 w-3.5 transition-transform duration-300 group-hover:-translate-y-0.5 group-hover:translate-x-0.5" />
          </a>
        </div>
      </Container>
    </section>
  );
}