import { Container } from "../common";
import { exploreLab } from "../../content/exploreContent";
import { useReveal } from "../../hooks/useReveal";
import HowItWorksSlideshow from "./HowItWorksSlideshow";

export default function ExploreStory() {
  const revealRef = useReveal({ y: 30 });

  return (
    <section
      ref={revealRef}
      id="lab"
      className="explore-section explore-story"
      aria-labelledby="lab-title"
    >
      <div className="explore-section__glow" aria-hidden="true" />

      <Container>
        <div className="explore-story__head">
          <div>
            <p className="explore-eyebrow" data-reveal>
              {exploreLab.eyebrow}
            </p>
            <h2 id="lab-title" className="explore-display mt-7" data-reveal>
              {exploreLab.title}
            </h2>
          </div>

          <div className="explore-story__lead">
            <p className="explore-lead" data-reveal>
              {exploreLab.lead}
            </p>
            {exploreLab.paragraphs.map((paragraph) => (
              <p key={paragraph} className="explore-body" data-reveal>
                {paragraph}
              </p>
            ))}
          </div>
        </div>

        <div className="explore-story__grid">
          <HowItWorksSlideshow />

          <div className="explore-story__principles">
            <p className="explore-eyebrow" data-reveal>
              How the lab works
            </p>

            <div className="explore-principles">
              {exploreLab.principles.map((principle) => (
                <article
                  key={principle.index}
                  className="explore-principle"
                  data-reveal
                >
                  <span className="explore-principle__index">
                    {principle.index}
                  </span>
                  <div>
                    <h3 className="explore-principle__title">
                      {principle.title}
                    </h3>
                    <p className="explore-principle__copy">
                      {principle.description}
                    </p>
                  </div>
                </article>
              ))}
            </div>

            <blockquote className="explore-note" data-reveal>
              <span aria-hidden="true">/</span>
              {exploreLab.note}
            </blockquote>
          </div>
        </div>
      </Container>
    </section>
  );
}