import { Container, SectionHeading, Button } from "../components/common";
import { Mail, ArrowUpRight } from "../components/common/Icons";
import { useReveal } from "../hooks/useReveal";
import { testimonials, site } from "../content/labContent";

function QuoteMark({ className = "" }) {
  return (
    <svg
      viewBox="0 0 64 48"
      className={className}
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M0 48V29.8C0 10.4 9.1 1.1 27.3 0v10.8c-8.1.8-12.2 5.9-12.2 15.3h12.2V48H0Zm36.7 0V29.8C36.7 10.4 45.8 1.1 64 0v10.8c-8.1.8-12.2 5.9-12.2 15.3H64V48H36.7Z" />
    </svg>
  );
}

function Initials({ name }) {
  const initials = name
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("");

  return (
    <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-edwins-dark font-orbitron text-[0.625rem] font-semibold tracking-[0.08em] text-maze-lime">
      {initials}
    </span>
  );
}

function TestimonialCard({ item, index }) {
  const styles = [
    "bg-white text-edwins-dark border-edwins-dark/10",
    "bg-maze-lime text-edwins-dark border-edwins-dark/15",
    "bg-edwins-dark text-white border-white/10",
  ];
  const isDark = index % 3 === 2;

  return (
    <figure
      className={`testimonial-card ${styles[index % styles.length]} flex h-full w-[min(82vw,390px)] shrink-0 flex-col justify-between rounded-[22px] border p-6 sm:w-[420px] sm:p-8`}
    >
      <div>
        <QuoteMark
          className={`h-7 w-9 ${isDark ? "text-maze-lime" : "text-edwins-dark/25"}`}
        />
        <blockquote className="mt-5 text-pretty text-[0.9375rem] leading-7 sm:text-base sm:leading-8">
          “{item.quote}”
        </blockquote>
      </div>
      <figcaption className={`mt-8 flex items-center gap-3 border-t pt-5 ${isDark ? "border-white/10" : "border-edwins-dark/10"}`}>
        <Initials name={item.name} />
        <div>
          <p className="text-sm font-semibold">{item.name}</p>
          {item.role && (
            <p className={`mt-0.5 text-xs ${isDark ? "text-white/60" : "text-edwins-dark/65"}`}>
              {item.role}
            </p>
          )}
        </div>
      </figcaption>
    </figure>
  );
}

function TestimonialsMarquee() {
  const items = testimonials.items;
  const groups = [items, items];

  return (
    <div className="testimonial-marquee relative mt-14 sm:mt-16">
      <div className="testimonial-track flex w-max gap-5 py-2">
        {groups.map((group, groupIndex) => (
          <div
            key={groupIndex}
            className="flex gap-5"
            aria-hidden={groupIndex === 1 ? "true" : undefined}
          >
            {group.map((item, index) => (
              <TestimonialCard
                key={`${groupIndex}-${index}-${item.name}`}
                item={item}
                index={index}
              />
            ))}
          </div>
        ))}
      </div>
      <p className="mt-6 text-center text-sm text-edwins-dark/45">
        Approved member stories. Hover to pause the loop.
      </p>
    </div>
  );
}

function EmptyVoices() {
  const prompts = [
    "What did you learn?",
    "What did you build?",
    "What did you share?",
  ];

  return (
    <div className="mt-14 overflow-hidden rounded-[26px] border border-edwins-dark/10 bg-white sm:mt-16">
      <div className="grid lg:grid-cols-[1.08fr_0.92fr]">
        <div className="p-7 sm:p-10 lg:p-12">
          <QuoteMark className="h-10 w-14 text-maze-lime" />
          <h3 className="mt-6 max-w-[16ch] text-balance font-orbitron text-[clamp(1.75rem,3vw,2.65rem)] font-semibold leading-[1.1] tracking-[-0.035em] text-edwins-dark">
            {testimonials.emptyTitle}
          </h3>
          <p className="mt-5 max-w-[48ch] text-[0.9375rem] leading-7 text-edwins-dark/65">
            {testimonials.emptyBody}
          </p>
          <Button
            href={`mailto:${site.email}?subject=My%20Edwins%20Lab%20story`}
            variant="dark"
            size="md"
            className="mt-8"
          >
            <Mail className="h-4 w-4" />
            Share your story
            <ArrowUpRight className="h-4 w-4" />
          </Button>
        </div>

        <div className="border-t border-edwins-dark/10 bg-maze-lime p-7 text-edwins-dark sm:p-10 lg:border-l lg:border-t-0 lg:p-12">
          <div className="flex items-center justify-between gap-4">
            <span className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.2em] text-edwins-dark/65">
              Story prompts
            </span>
            <span className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.2em] text-edwins-dark/65">
              01 — 03
            </span>
          </div>

          <ul className="mt-8">
            {prompts.map((prompt, index) => (
              <li
                key={prompt}
                className="flex items-baseline gap-4 border-b border-edwins-dark/15 py-5"
              >
                <span className="font-orbitron text-[0.625rem] font-semibold text-edwins-dark/75">
                  0{index + 1}
                </span>
                <span className="font-orbitron text-lg font-medium tracking-[-0.02em] sm:text-xl">
                  {prompt}
                </span>
              </li>
            ))}
          </ul>

          <p className="mt-8 max-w-[30ch] text-sm leading-6 text-edwins-dark/75">
            Real words from real members. Approved stories are added here as
            they arrive.
          </p>
        </div>
      </div>
    </div>
  );
}

export default function Testimonials() {
  const revealRef = useReveal();
  const hasItems = testimonials.items.length > 0;

  return (
    <section
      ref={revealRef}
      id="testimonials"
      aria-labelledby="testimonials-title"
      className="bg-edwins-neutral py-24 sm:py-28 lg:py-36"
    >
      <Container>
        <div className="grid gap-8 lg:grid-cols-[1fr_0.72fr] lg:items-end lg:gap-16">
          <SectionHeading
            eyebrow={testimonials.eyebrow}
            title={testimonials.title}
            id="testimonials-title"
          />
          <p
            data-reveal
            className="max-w-[46ch] text-pretty text-[0.9375rem] leading-7 text-edwins-dark/65 sm:text-base"
          >
            {testimonials.intro}
          </p>
        </div>

        <div data-reveal>{hasItems ? <TestimonialsMarquee /> : <EmptyVoices />}</div>
      </Container>
    </section>
  );
}