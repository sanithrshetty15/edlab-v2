import { useEffect, useRef } from 'react'
import { gsap } from 'gsap'
import logo from '../../assets/images/edwins-lab-logo.png'
import { prefersReducedMotion } from '../../utils/reducedMotion'

const LOGO_WIDTH = 'w-[min(90vw,52rem,138vh)]'
const TEXT_ONLY_CLIP = { clipPath: 'inset(0 0 0 33%)' }

function LogoImage({ textOnly = false, alt = '' }) {
  return (
    <img
      src={logo}
      alt={alt}
      draggable="false"
      decoding="async"
      className="block h-auto w-full select-none"
      style={textOnly ? TEXT_ONLY_CLIP : undefined}
      aria-hidden={textOnly ? 'true' : undefined}
    />
  )
}

export default function IntroAnimation({ onComplete, onRevealStart }) {
  const overlayRef = useRef(null)
  const fullLogoRef = useRef(null)
  const portraitMaskRef = useRef(null)
  const topPanelRef = useRef(null)
  const bottomPanelRef = useRef(null)
  const topMarkRef = useRef(null)
  const bottomMarkRef = useRef(null)

  useEffect(() => {
    if (!overlayRef.current) return undefined

    if (prefersReducedMotion()) {
      onComplete?.()
      return undefined
    }

    let ctx

    try {
      ctx = gsap.context(() => {
        const tl = gsap.timeline({
          defaults: { ease: 'power3.out' },
          onComplete: () => onComplete?.(),
        })

        tl.fromTo(
          fullLogoRef.current,
          { autoAlpha: 0, scale: 0.96, y: 12 },
          { autoAlpha: 1, scale: 1, y: 0, duration: 1, ease: 'power2.out' },
          0.2,
        )
          .to(
            portraitMaskRef.current,
            { autoAlpha: 1, duration: 0.8, ease: 'power2.inOut' },
            '+=0.3',
          )
          .addLabel('split', '+=0.3')
          .call(() => onRevealStart?.(), [], 'split')
          .set(fullLogoRef.current, { autoAlpha: 0 }, 'split')
          .set([topMarkRef.current, bottomMarkRef.current], { autoAlpha: 1 }, 'split')
          .to(
            topPanelRef.current,
            { yPercent: -100, duration: 1.15, ease: 'power4.inOut' },
            'split',
          )
          .to(
            bottomPanelRef.current,
            { yPercent: 100, duration: 1.15, ease: 'power4.inOut' },
            'split',
          )
      }, overlayRef)
    } catch {
      onComplete?.()
      return undefined
    }

    return () => ctx.revert()
  }, [onComplete, onRevealStart])

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-[100] overflow-hidden"
      role="presentation"
    >
      <div
        ref={topPanelRef}
        className="absolute inset-x-0 top-0 h-1/2 overflow-hidden bg-white"
      >
        <div
          className={`absolute left-1/2 top-full -translate-x-1/2 -translate-y-1/2 ${LOGO_WIDTH}`}
        >
          <div ref={topMarkRef} className="opacity-0">
            <LogoImage textOnly />
          </div>
        </div>
      </div>

      <div
        ref={bottomPanelRef}
        className="absolute inset-x-0 bottom-0 h-1/2 overflow-hidden bg-white"
      >
        <div
          className={`absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/2 ${LOGO_WIDTH}`}
        >
          <div ref={bottomMarkRef} className="opacity-0">
            <LogoImage textOnly />
          </div>
        </div>
      </div>

      <div className="pointer-events-none absolute inset-0 grid place-items-center">
        <div ref={fullLogoRef} className={`relative opacity-0 ${LOGO_WIDTH}`}>
          <LogoImage alt="Edwins Lab" />
          <div
            ref={portraitMaskRef}
            className="absolute inset-y-0 left-0 w-[33%] bg-white opacity-0"
            aria-hidden="true"
          />
        </div>
      </div>
    </div>
  )
}
