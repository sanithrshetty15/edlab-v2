import { Container, Button } from "../components/common";
import { Mail } from "../components/common/Icons";
import { useReveal } from "../hooks/useReveal";
import { navigation, site, footer } from "../content/labContent";
import wordmarkLight from "../assets/images/edwins-lab-wordmark-light.png";
import InteractiveContourField from "../components/shared/InteractiveContourField";

export default function Footer() {
  const revealRef = useReveal({ start: "top 90%" });
  const footerLinks = navigation.filter((item) => item.href !== "#contact");

  return (
    <footer
      ref={revealRef}
      id="contact"
      className="overflow-hidden bg-edwins-dark text-white"
      style={{ position: "relative" }}
    >
      <InteractiveContourField />
      <Container className="pb-9 pt-20 sm:pt-24 lg:pt-28">
        <div className="grid gap-12 border-b border-white/12 pb-14 sm:pb-20 lg:grid-cols-[1.18fr_0.82fr] lg:gap-16">
          <div data-reveal>
            <p className="mb-6 inline-flex items-center gap-3 font-orbitron text-[0.625rem] font-semibold uppercase tracking-[0.22em] text-white/55">
              <span className="h-1.5 w-1.5 rounded-full bg-maze-lime" />
              Contact / The lab
            </p>
            <h2 className="max-w-[14ch] text-balance font-orbitron text-[clamp(2.5rem,5.2vw,4.75rem)] font-semibold leading-[1.02] tracking-[-0.04em]">
              {footer.ctaTitle}
            </h2>
            <p className="mt-6 max-w-[46ch] text-[0.9375rem] leading-7 text-white/60 sm:text-base">
              {footer.ctaBody}
            </p>
            <Button
              href={`mailto:${site.email}`}
              variant="primary"
              size="lg"
              className="mt-9"
            >
              <Mail className="h-4 w-4" />
              Write to the lab
            </Button>
          </div>

          <div
            data-reveal
            className="flex flex-col justify-between gap-10"
          >
            <div className="flex justify-start items-start gap-3 lg:justify-end">
              <img
                src={wordmarkLight}
                alt="Edwins Lab"
                width="554"
                height="252"
                loading="lazy"
                className="h-auto w-36 sm:w-44"
              />
              <span className="mt-1 flex h-2 w-2 shrink-0 rounded-full bg-maze-lime" />
            </div>

            <div className="grid grid-cols-1 gap-8 min-[480px]:grid-cols-2">
              <nav aria-label="Footer">
                <p className="font-orbitron text-[0.5625rem] font-semibold uppercase tracking-[0.2em] text-white/55">
                  Explore
                </p>
                <ul className="mt-4 space-y-3">
                  {footerLinks.map((item) => (
                    <li key={item.label}>
                      <a
                        href={item.href}
                        className="footer-link text-sm text-white/70 transition-colors duration-300 hover:text-maze-lime"
                      >
                        {item.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>

              <div id="contact-details">
                <p className="font-orbitron text-[0.5625rem] font-semibold uppercase tracking-[0.2em] text-white/55">
                  Elsewhere
                </p>
                <ul className="mt-4 space-y-3">
                  <li>
                    <a
                      href={`mailto:${site.email}`}
                      className="footer-link text-sm text-white/70 transition-colors duration-300 hover:text-maze-lime"
                    >
                      {site.email}
                    </a>
                  </li>
                  <li>
                    <a
                      href="#top"
                      className="footer-link text-sm text-white/70 transition-colors duration-300 hover:text-maze-lime"
                    >
                      Back to top
                    </a>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </Container>
    </footer>
  );
}