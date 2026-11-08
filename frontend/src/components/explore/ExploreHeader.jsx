import { ArrowLeft } from "../common/Icons";
import { navigate } from "../../utils/router";

export default function ExploreHeader() {
  const handleBack = (event) => {
    if (
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return;
    }

    event.preventDefault();
    navigate("/");
  };

  return (
    <header className="explore-header">
      <a href="/" onClick={handleBack} className="explore-back group">
        <span className="explore-back__icon">
          <ArrowLeft />
        </span>
        <span>Back to Home</span>
      </a>

      <div className="explore-header__mark" aria-hidden="true">
        <span className="explore-header__rule" />
        <span>Edwin&apos;s Lab</span>
        <span className="explore-header__dot" />
      </div>
    </header>
  );
}
