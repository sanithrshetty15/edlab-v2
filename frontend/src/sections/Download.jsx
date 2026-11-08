import { Button, Container, SectionHeading } from "../components/common";
import {
  Android,
  ArrowUpRight,
  Download as DownloadIcon,
  Globe,
  Mail,
} from "../components/common/Icons";
import TuxArtwork from "../components/tux/TuxArtwork";
import { useReveal } from "../hooks/useReveal";
import { useParallax } from "../hooks/useParallax";
import { downloads } from "../content/labContent";
import portrait from "../assets/images/edwins-portrait.png";
import wordmark from "../assets/images/edwins-lab-wordmark.png";

function DownloadOption({ icon, title, description, note, href, label }) {
  return (
    <article className="flex flex-col rounded-[18px] border border-white/12 bg-white/[0.035] p-5 transition-colors duration-300 hover:border-white/22 hover:bg-white/[0.055] sm:p-6">
      <div className="flex items-start justify-between gap-4">
        <span className="flex h-11 w-11 items-center justify-center rounded-full bg-maze-lime text-edwins-dark">
          {icon}
        </span>
        <span className="rounded-full border border-white/12 px-3 py-1 font-orbitron text-[0.5625rem] font-semibold uppercase tracking-[0.16em] text-white/50">
          {href ? "Available" : "In preparation"}
        </span>
      </div>
      <h3 className="mt-6 font-orbitron text-lg font-medium tracking-[-0.02em] text-white sm:text-xl">
        {title}
      </h3>
      <p className="mt-2 max-w-[36ch] text-sm leading-6 text-white/60">
        {description}
      </p>
      <div className="mt-6 flex items-center gap-3">
        {href ? (
          <Button href={href} variant="primary" size="sm">
            <DownloadIcon className="h-4 w-4" />
            {label}
          </Button>
        ) : (
          <span className="inline-flex items-center gap-2 rounded-full border border-white/12 px-4 py-2.5 font-poppins text-xs font-medium text-white/55">
            <span className="h-1.5 w-1.5 rounded-full bg-maze-lime/60" />
            {note}
          </span>
        )}
      </div>
    </article>
  );
}

function PhonePreview() {
  return (
    <div className="relative mx-auto w-full max-w-[330px]" aria-hidden="true">
      <div className="phone-frame">
        <div className="phone-screen">
          <div className="flex items-center justify-between px-1">
            <span className="font-orbitron text-[0.5rem] font-semibold uppercase tracking-[0.16em] text-edwins-dark/35">
              Edwins Lab
            </span>
            <span className="flex gap-1">
              <span className="h-1 w-1 rounded-full bg-edwins-dark/25" />
              <span className="h-1 w-1 rounded-full bg-edwins-dark/25" />
              <span className="h-1 w-1 rounded-full bg-edwins-dark/25" />
            </span>
          </div>

          <div className="mt-5 flex items-center gap-2.5">
            <img
              src={portrait}
              alt=""
              width="219"
              height="219"
              loading="lazy"
              className="h-9 w-9 rounded-full border border-edwins-dark/15 object-cover"
            />
            <img
              src={wordmark}
              alt=""
              width="554"
              height="252"
              loading="lazy"
              className="h-4 w-auto"
            />
          </div>

          <div className="phone-hero mt-5">
            <p className="font-orbitron text-[0.5rem] font-semibold uppercase tracking-[0.18em] text-edwins-dark/45">
              Welcome back
            </p>
            <p className="mt-2 font-orbitron text-lg font-semibold leading-[1.12] tracking-[-0.035em] text-edwins-dark">
              Learn.
              <br />
              Build.
              <br />
              Innovate.
            </p>
            <TuxArtwork
              className="phone-tux absolute -bottom-6 -right-6 w-[62%]"
              title=""
            />
          </div>

          <div className="mt-6 space-y-2">
            {["About the lab", "Linux practice", "Download app"].map((label) => (
              <div
                key={label}
                className="flex items-center justify-between rounded-xl border border-edwins-dark/8 bg-white/70 px-3.5 py-2.5"
              >
                <span className="text-[0.625rem] font-medium text-edwins-dark/70">
                  {label}
                </span>
                <ArrowUpRight className="h-3 w-3 text-edwins-dark/35" />
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="absolute -right-5 top-10 rounded-full border border-edwins-dark/10 bg-white px-3.5 py-2 font-orbitron text-[0.5625rem] font-semibold uppercase tracking-[0.14em] text-edwins-dark shadow-[0_12px_30px_rgba(28,28,28,0.12)] sm:-right-8">
        Coming soon
      </div>
      <div className="absolute -bottom-5 -left-4 flex h-16 w-16 items-center justify-center rounded-full border border-edwins-dark/10 bg-maze-lime text-edwins-dark shadow-[0_18px_40px_rgba(28,28,28,0.16)] sm:-left-8">
        <DownloadIcon className="h-5 w-5" />
      </div>
    </div>
  );
}

export default function Download() {
  const revealRef = useReveal();
  const phoneRef = useParallax({ distance: 18, triggerRef: revealRef });

  return (
    <section
      ref={revealRef}
      id="download"
      aria-labelledby="download-title"
      className="overflow-hidden bg-white py-24 sm:py-28 lg:py-36"
    >
      <Container>
        <div className="grid gap-8 lg:grid-cols-[1fr_0.8fr] lg:items-end lg:gap-16">
          <SectionHeading
            eyebrow={downloads.eyebrow}
            title={downloads.title}
            id="download-title"
          />
          <p
            data-reveal
            className="max-w-[46ch] text-pretty text-[0.9375rem] leading-7 text-edwins-dark/65 sm:text-base"
          >
            {downloads.intro}
          </p>
        </div>

        <div
          data-reveal
          className="download-panel relative mt-14 overflow-hidden rounded-[28px] bg-edwins-dark p-6 text-white sm:mt-16 sm:p-9 lg:p-12"
        >
          <div className="download-panel__grid" aria-hidden="true" />
          <div className="download-panel__glow" aria-hidden="true" />

          <div className="relative z-10 grid gap-12 lg:grid-cols-[1.12fr_0.88fr] lg:items-center lg:gap-10">
            <div>
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-white/12 pb-5">
                <span className="inline-flex items-center gap-2.5 font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.18em] text-white/55">
                  <span className="h-1.5 w-1.5 rounded-full bg-maze-lime" />
                  Release status / In preparation
                </span>
                <span className="font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.18em] text-white/55">
                  Edwins Lab / Mobile
                </span>
              </div>

              <p className="mt-7 max-w-[34ch] text-pretty font-orbitron text-[clamp(1.5rem,2.6vw,2.15rem)] font-medium leading-[1.18] tracking-[-0.03em]">
                The lab, wherever you are.
              </p>
              <p className="mt-4 max-w-[48ch] text-sm leading-7 text-white/60">
                Two ways to carry the lab with you. Both are being prepared and
                will be linked here the moment they are approved.
              </p>

              <div className="mt-8 grid gap-4 sm:grid-cols-2">
                <DownloadOption
                  icon=<Android className="h-5 w-5" />
                  title={downloads.android.title}
                  description={downloads.android.description}
                  note={downloads.android.note}
                  label="Download APK"
                  href={downloads.apkUrl}
                />
                <DownloadOption
                  icon=<Globe className="h-5 w-5" />
                  title={downloads.pwa.title}
                  description={downloads.pwa.description}
                  note={downloads.pwa.note}
                  label="Install web app"
                  href={downloads.pwaUrl}
                />
              </div>

              <div className="mt-7 flex flex-wrap items-center gap-4">
                <Button
                  href={downloads.notifyHref}
                  variant="primary"
                  size="md"
                >
                  <Mail className="h-4 w-4" />
                  {downloads.notifyLabel}
                </Button>
                <span className="text-xs leading-5 text-white/50">
                  No release dates are promised until the builds are ready.
                </span>
              </div>
            </div>

            <div ref={phoneRef} className="relative pb-5 lg:pb-0">
              <PhonePreview />
            </div>
          </div>
        </div>
      </Container>
    </section>
  );
}