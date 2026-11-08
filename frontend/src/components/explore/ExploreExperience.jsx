import { Container } from "../common";
import { exploreExperience } from "../../content/exploreContent";
import { labGallery } from "../../content/exploreGallery";
import { useReveal } from "../../hooks/useReveal";
import RevealFrame from "./RevealFrame";

export default function ExploreExperience() {
  const revealRef = useReveal({ y: 28 });

  return (
    <section
      ref={revealRef}
      id="experience"
      className="explore-section explore-experience"
      aria-labelledby="experience-title"
    >
      <div className="explore-section__glow explore-section__glow--cyan" aria-hidden="true" />

      <Container>
        <div className="explore-experience__head">
          <div>
            <p className="explore-eyebrow" data-reveal>
              {exploreExperience.eyebrow}
            </p>
            <h2
              id="experience-title"
              className="explore-display mt-7"
              data-reveal
            >
              {exploreExperience.title}
            </h2>
          </div>
          <p className="explore-lead" data-reveal>
            {exploreExperience.intro}
          </p>
        </div>

        <div className="explore-experience__grid">
          {exploreExperience.items.map((item, position) => {
            const image =
              labGallery[item.imageIndex % labGallery.length];

            return (
              <article
                key={item.title}
                className="explore-experience-card"
                data-reveal
              >
                <RevealFrame
                  item={image}
                  index={item.imageIndex}
                  ratio="4 / 3"
                  className="explore-experience-card__media"
                  overlay={
                    <span className="explore-experience-card__frame">
                      Frame {String(position + 1).padStart(2, "0")}
                    </span>
                  }
                />

                <div className="explore-experience-card__copy">
                  <span className="explore-experience-card__index">
                    / {item.index}
                  </span>
                  <h3>{item.title}</h3>
                  <p>{item.description}</p>
                </div>
              </article>
            );
          })}
        </div>

        <div className="explore-experience__closing" data-reveal>
          <span className="explore-eyebrow">
            {exploreExperience.closingLabel}
          </span>
          <p>{exploreExperience.closing}</p>
        </div>
      </Container>
    </section>
  );
}