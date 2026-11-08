import { useEffect } from "react";
import "../explore.css";
import ExploreHeader from "../components/explore/ExploreHeader";
import ExploreHero from "../components/explore/ExploreHero";
import ExploreStory from "../components/explore/ExploreStory";
import GallerySection from "../components/explore/GallerySection";
import ExploreExperience from "../components/explore/ExploreExperience";
import ExploreFooter from "../components/explore/ExploreFooter";

export default function ExplorePage() {
  useEffect(() => {
    document.body.classList.add("explore-mode");
    window.scrollTo({ top: 0, left: 0, behavior: "auto" });

    return () => document.body.classList.remove("explore-mode");
  }, []);

  return (
    <div id="explore-top" className="explore-page">
      <a href="#explore-main" className="skip-link">
        Skip to content
      </a>

      <ExploreHeader />

      <main id="explore-main" tabIndex={-1}>
        <ExploreHero />
        <ExploreStory />
        <GallerySection />
        <ExploreExperience />
      </main>

      <ExploreFooter />
    </div>
  );
}
