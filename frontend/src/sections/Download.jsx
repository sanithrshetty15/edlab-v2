import { useCallback, useEffect, useState } from "react";
import { Capacitor } from "@capacitor/core";
import { Button, Container, SectionHeading } from "../components/common";
import {
  Android,
  ArrowUpRight,
  Check,
  Download as DownloadIcon,
  Globe,
  Mail,
  Share,
  Smartphone,
} from "../components/common/Icons";
import TuxArtwork from "../components/tux/TuxArtwork";
import { useReveal } from "../hooks/useReveal";
import { useParallax } from "../hooks/useParallax";
import { usePwaInstall } from "../hooks/usePwaInstall";
import { useApkRelease } from "../hooks/useApkRelease";
import { downloads } from "../content/labContent";
import portrait from "../assets/images/edwins-portrait.png";
import wordmark from "../assets/images/edwins-lab-wordmark.png";

const ANDROID_VERSIONS = {
  21: "5.0",
  22: "5.1",
  23: "6.0",
  24: "7.0",
  25: "7.1",
  26: "8.0",
  27: "8.1",
  28: "9",
  29: "10",
  30: "11",
  31: "12",
  32: "12L",
  33: "13",
  34: "14",
  35: "15",
};

function androidVersionLabel(minSdk) {
  const version = ANDROID_VERSIONS[minSdk];
  return version ? `Android ${version}+` : "Android";
}

const WEB_STATUS_LABELS = {
  available: "Available",
  "prompt-open": "Prompt open",
  accepted: "Accepted",
  dismissed: "Dismissed",
  installed: "Installed",
  ios: "iPhone & iPad",
  unavailable: "Browser menu",
};

function StatusBadge({ children, tone = "default" }) {
  return (
    <span className={`download-card__badge download-card__badge--${tone}`}>
      {children}
    </span>
  );
}

function StatusPanel({ tone = "default", icon, title, children, live = false }) {
  return (
    <div className={`download-status download-status--${tone}`} role={live ? "status" : undefined} aria-live={live ? "polite" : undefined}>
      <span className="download-status__icon" aria-hidden="true">
        {icon}
      </span>
      <div>
        <p className="download-status__title">{title}</p>
        <div className="download-status__body">{children}</div>
      </div>
    </div>
  );
}

function InstallAppCard() {
  const install = usePwaInstall();
  const { status, isIos, isAndroid, browser, promptInstall } = install;

  return (
    <article className="download-card" data-reveal>
      <header className="download-card__head">
        <span className="download-card__icon" aria-hidden="true">
          <Globe className="h-5 w-5" />
        </span>
        <StatusBadge tone={status === "installed" ? "success" : "lime"}>
          {WEB_STATUS_LABELS[status] || "Web app"}
        </StatusBadge>
      </header>

      <h3 className="download-card__title">{downloads.pwa.title}</h3>
      <p className="download-card__description">{downloads.pwa.description}</p>

      <div className="download-card__body">
        {status === "installed" && (
          <StatusPanel
            tone="success"
            icon={<Check className="h-4 w-4" />}
            title="Installed on this device"
          >
            <p>Open Edwins Lab from your home screen, dock or app launcher.</p>
          </StatusPanel>
        )}

        {status === "available" && (
          <div className="download-card__action">
            <Button
              variant="primary"
              size="lg"
              className="w-full sm:w-auto"
              onClick={promptInstall}
            >
              <DownloadIcon className="h-4 w-4" />
              Install App
            </Button>
            <p className="download-card__hint">
              Your browser will ask you to confirm before anything is installed.
            </p>
          </div>
        )}

        {status === "prompt-open" && (
          <StatusPanel
            tone="lime"
            live
            icon={<Smartphone className="h-4 w-4" />}
            title="Finish the browser prompt"
          >
            <p>The installation dialog is open in your browser.</p>
          </StatusPanel>
        )}

        {status === "accepted" && (
          <StatusPanel
            tone="success"
            live
            icon={<Check className="h-4 w-4" />}
            title="Installation accepted"
          >
            <p>
              The browser accepted the request. If the app does not appear on
              your device yet, check your home screen or app list.
            </p>
          </StatusPanel>
        )}

        {status === "dismissed" && (
          <StatusPanel
            tone="muted"
            live
            icon={<Globe className="h-4 w-4" />}
            title="Installation dismissed"
          >
            <p>
              Nothing was installed. You can install later from your browser
              menu.
            </p>
          </StatusPanel>
        )}

        {status === "ios" && (
          <div className="download-guide" role="note">
            <p className="download-guide__lead">
              On iPhone and iPad, installation happens through Safari:
            </p>
            <ol className="download-guide__steps">
              <li>
                <Share className="h-3.5 w-3.5" aria-hidden="true" />
                Open this page in Safari and tap the Share button.
              </li>
              <li>Choose “Add to Home Screen”.</li>
              <li>Tap “Add” in the top corner.</li>
            </ol>
            <p className="download-guide__note">
              iOS does not tell the website when the icon is added, so this
              guide stays visible after installation.
            </p>
          </div>
        )}

        {status === "unavailable" && (
          <div className="download-guide" role="note">
            <p className="download-guide__lead">
              This browser has not offered an in-page install prompt. Install
              manually:
            </p>
            <ul className="download-guide__steps">
              {browser === "edge" && (
                <li>Open the “…” menu → Apps → “Install this site as an app”.</li>
              )}
              {browser === "samsung" && (
                <li>Open the browser menu and choose “Add page to” → Home screen.</li>
              )}
              {browser === "chrome" && isAndroid && (
                <li>Open the ⋮ menu and choose “Install app” or “Add to Home screen”.</li>
              )}
              {browser === "chrome" && !isAndroid && (
                <li>
                  Use the install icon in the address bar, or open ⋮ → “Cast,
                  save and share” → “Install page as app”.
                </li>
              )}
              {browser === "safari" && !isIos && (
                <li>In Safari, choose File → “Add to Dock”.</li>
              )}
              {browser === "other" && (
                <li>
                  Use the browser menu and look for “Install app”, “Add to Home
                  screen” or “Install this site as an app”.
                </li>
              )}
            </ul>
            <p className="download-guide__note">
              If your browser has no install option, open Edwins Lab in Chrome,
              Edge or Safari to install it.
            </p>
          </div>
        )}
      </div>

      <footer className="download-card__foot">
        <span className="download-card__meta">
          <span className="download-card__dot" aria-hidden="true" />
          Runs in standalone mode · No app store required
        </span>
      </footer>
    </article>
  );
}

function ApkCard() {
  const { status, release } = useApkRelease();
  const [downloadState, setDownloadState] = useState("idle");
  const [progress, setProgress] = useState(0);

  const handleDownload = useCallback(async () => {
    if (!release) return;

    setDownloadState("downloading");
    setProgress(0);

    try {
      const response = await fetch(release.file, { cache: "no-store" });
      if (!response.ok) throw new Error("APK could not be downloaded.");

      const total =
        Number(response.headers.get("content-length")) || release.sizeBytes || 0;
      const reader = response.body?.getReader?.();
      let blob;

      if (reader) {
        const chunks = [];
        let received = 0;

        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          chunks.push(value);
          received += value.length;
          if (total > 0) {
            setProgress(Math.min(99, Math.round((received / total) * 100)));
          }
        }

        blob = new Blob(chunks, {
          type: "application/vnd.android.package-archive",
        });

        if (total > 0 && blob.size !== total) {
          throw new Error("APK download was incomplete.");
        }
      } else {
        blob = await response.blob();
      }

      const objectUrl = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = objectUrl;
      anchor.download = release.file.split("/").pop() || "EdwinsLab.apk";
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.setTimeout(() => URL.revokeObjectURL(objectUrl), 120000);

      setDownloadState("downloaded");
      setProgress(100);
    } catch {
      setDownloadState("error");
      setProgress(0);
    }
  }, [release]);

  return (
    <article className="download-card" data-reveal>
      <header className="download-card__head">
        <span className="download-card__icon" aria-hidden="true">
          <Android className="h-5 w-5" />
        </span>
        <StatusBadge
          tone={
            status === "available"
              ? "lime"
              : status === "unavailable"
                ? "muted"
                : "default"
          }
        >
          {status === "available"
            ? `v${release.versionName}`
            : status === "unavailable"
              ? "In preparation"
              : "Checking release"}
        </StatusBadge>
      </header>

      <h3 className="download-card__title">{downloads.android.title}</h3>
      <p className="download-card__description">{downloads.android.description}</p>

      <div className="download-card__body">
        {status === "loading" && (
          <StatusPanel tone="muted" icon={<Android className="h-4 w-4" />} title="Checking the latest release">
            <p>Looking for a signed build of Edwins Lab for Android.</p>
          </StatusPanel>
        )}

        {status === "unavailable" && (
          <StatusPanel
            tone="muted"
            icon={<Android className="h-4 w-4" />}
            title="The APK is being prepared"
          >
            <p>
              A signed Android build will be published here once it passes
              testing. No download link is shown until the file exists.
            </p>
          </StatusPanel>
        )}

        {status === "available" && release && (
          <>
            <dl className="download-meta">
              <div>
                <dt>Version</dt>
                <dd>
                  {release.versionName}
                  {release.versionCode ? ` (build ${release.versionCode})` : ""}
                </dd>
              </div>
              <div>
                <dt>File size</dt>
                <dd>{release.sizeLabel || "Verified on the server"}</dd>
              </div>
              <div>
                <dt>Package</dt>
                <dd>{release.packageId || "com.edwinslab.app"}</dd>
              </div>
            </dl>

            {downloadState === "downloading" && (
              <div className="download-progress" role="status" aria-live="polite">
                <div className="download-progress__track">
                  <span
                    className="download-progress__bar"
                    style={{ width: `${progress}%` }}
                  />
                </div>
                <p className="download-progress__label">
                  Downloading… {progress}%
                </p>
              </div>
            )}

            {downloadState === "downloaded" && (
              <StatusPanel
                tone="success"
                live
                icon={<Check className="h-4 w-4" />}
                title="APK download complete"
              >
                <p>
                  Check your browser downloads, then open the file to install
                  Edwins Lab.
                </p>
              </StatusPanel>
            )}

            {downloadState === "error" && (
              <StatusPanel
                tone="muted"
                live
                icon={<Android className="h-4 w-4" />}
                title="Download did not finish"
              >
                <p>Check your connection and try again.</p>
              </StatusPanel>
            )}

            <div className="download-card__action">
              <Button
                variant={downloadState === "downloaded" ? "light" : "primary"}
                size="lg"
                className="w-full sm:w-auto"
                onClick={handleDownload}
                disabled={downloadState === "downloading"}
              >
                <DownloadIcon className="h-4 w-4" />
                {downloadState === "downloading"
                  ? "Downloading APK…"
                  : downloadState === "downloaded"
                    ? "Download again"
                    : "Download APK"}
              </Button>
              <p className="download-card__hint">
                Android may ask you to allow installs from this browser.
                Enable “Install unknown apps”, then open the downloaded file.
              </p>
            </div>
          </>
        )}
      </div>

      <footer className="download-card__foot">
        <span className="download-card__meta">
          <span className="download-card__dot" aria-hidden="true" />
          {status === "available"
            ? `${androidVersionLabel(release.minSdk)} · Signed release build`
            : "Android build · Signed release pending"}
        </span>
      </footer>
    </article>
  );
}

function NativeAppCard() {
  const [info, setInfo] = useState(null);

  useEffect(() => {
    let active = true;

    import("@capacitor/app")
      .then(({ App }) => App.getInfo())
      .then((value) => {
        if (active) setInfo(value);
      })
      .catch(() => {});

    return () => {
      active = false;
    };
  }, []);

  return (
    <article className="download-card" data-reveal>
      <header className="download-card__head">
        <span className="download-card__icon" aria-hidden="true">
          <Android className="h-5 w-5" />
        </span>
        <StatusBadge tone="success">Installed</StatusBadge>
      </header>

      <h3 className="download-card__title">Edwins Lab for Android</h3>
      <p className="download-card__description">
        You are using the Android build installed on this device. Open the web
        app or download the APK from a browser when you need them elsewhere.
      </p>

      <div className="download-card__body">
        <StatusPanel
          tone="success"
          icon={<Check className="h-4 w-4" />}
          title="Android app installed"
        >
          <p>
            {info
              ? `Version ${info.version} (build ${info.build}) · ${info.id}`
              : "Signed release build"}
          </p>
        </StatusPanel>
      </div>

      <footer className="download-card__foot">
        <span className="download-card__meta">
          <span className="download-card__dot" aria-hidden="true" />
          Signed release build · No app store required
        </span>
      </footer>
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
        Installable
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
  const isNative = Capacitor.isNativePlatform();

  return (
    <section
      ref={revealRef}
      id="download"
      aria-labelledby="download-title"
      className="download-section relative overflow-hidden bg-edwins-dark py-24 text-white sm:py-28 lg:py-36"
    >
      <div className="download-section__grid" aria-hidden="true" />
      <div className="download-section__glow" aria-hidden="true" />

      <Container className="relative z-10">
        <div className="grid gap-12 lg:grid-cols-[1.05fr_0.75fr] lg:items-center lg:gap-16">
          <div>
            <SectionHeading
              dark
              eyebrow={downloads.eyebrow}
              title={downloads.title}
              id="download-title"
            />
            <p
              data-reveal
              className="mt-6 max-w-[48ch] text-pretty text-[0.9375rem] leading-7 text-white/60 sm:text-base"
            >
              {downloads.intro}
            </p>
            <div data-reveal className="mt-8 flex flex-wrap gap-3">
              <span className="download-chip">
                <Smartphone className="h-4 w-4" aria-hidden="true" />
                Installable web app
              </span>
              <span className="download-chip">
                <Android className="h-4 w-4" aria-hidden="true" />
                Android APK
              </span>
            </div>
          </div>

          <div ref={phoneRef} className="relative pb-5 lg:pb-0">
            <PhonePreview />
          </div>
        </div>

        <div className="mt-14 grid gap-4 md:grid-cols-2 lg:mt-16 lg:gap-6">
          {isNative ? (
            <NativeAppCard />
          ) : (
            <>
              <InstallAppCard />
              <ApkCard />
            </>
          )}
        </div>

        <div
          data-reveal
          className="mt-6 flex flex-col gap-4 rounded-[20px] border border-white/10 bg-white/[0.035] p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6"
        >
          <div>
            <p className="font-orbitron text-sm font-medium tracking-[-0.01em] text-white">
              Release updates by email
            </p>
            <p className="mt-1 text-xs leading-5 text-white/50">
              Ask the lab to email you when a new APK or web app release is
              published. No release dates are promised until the builds are
              ready.
            </p>
          </div>
          <Button
            href={downloads.notifyHref}
            variant="primary"
            size="md"
            className="shrink-0"
          >
            <Mail className="h-4 w-4" />
            {downloads.notifyLabel}
          </Button>
        </div>
      </Container>
    </section>
  );
}
