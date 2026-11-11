import { useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import logo from '../../assets/images/edwins-lab-logo.png'
import { prefersReducedMotion } from '../../utils/reducedMotion'

const LOGO_WIDTH = 'w-[min(90vw,52rem,138vh)]'

function LogoImage({ textOnly = false, alt = '' }) {
  return (
    <img
      src={logo}
      alt={alt}
      draggable="false"
      decoding="async"
      className="block h-auto w-full select-none"
      style={textOnly ? { clipPath: 'inset(0 0 0 33%)' } : undefined}
      aria-hidden={textOnly ? 'true' : undefined}
    />
  )
}

const STAIRCASE_DURATION = 1800
const TOP_STAGGER = 55
const BOTTOM_STAGGER = 48

export default function IntroAnimation({ onComplete, onRevealStart }) {
  const containerRef = useRef(null)

  useEffect(() => {
    if (!containerRef.current) return undefined

    if (prefersReducedMotion()) {
      onComplete?.()
      return undefined
    }

    const handleRevealStart = () => onRevealStart?.()
    window.addEventListener('edwins:site-reveal', handleRevealStart)

    return () => window.removeEventListener('edwins:site-reveal', handleRevealStart)
  }, [onComplete, onRevealStart])

  return (
    <AnimatePresence>
      <div
        ref={containerRef}
        className="fixed inset-0 z-[100] overflow-hidden bg-edwins-dark"
        role="presentation"
      >
        {/* Centered logo + text - visible initially, fades when reveal starts */}
        <motion.div
          className="pointer-events-none absolute inset-0 grid place-items-center"
          initial={{ opacity: 1 }}
          animate={{ opacity: 0 }}
          transition={{ duration: 0.4, ease: 'easeOut' }}
          onAnimationComplete={() => onRevealStart?.()}
        >
          <div className={`relative opacity-0 ${LOGO_WIDTH}`}>
            <LogoImage alt="Edwins Lab" />
          </div>
        </motion.div>

        {/* Top staircase panels - fall downward */}
        <div className="absolute inset-x-0 top-0 overflow-hidden">
          {Array.from({ length: 14 }).map((_, i) => {
            const delay = i * TOP_STAGGER
            const panelHeight = 180 + (i % 3) * 40
            const isWide = i % 2 === 0
            return (
              <motion.div
                key={`top-${i}`}
                className="absolute inset-x-0 bg-edwins-dark"
                style={{ height: `${panelHeight}px` }}
                initial={{
                  y: '-100%',
                  opacity: 0.92,
                }}
                animate={{
                  y: '100%',
                  opacity: 0,
                }}
                transition={{
                  duration: STAIRCASE_DURATION / 1000,
                  delay: delay / 1000,
                  ease: 'easeOut',
                  opacity: { duration: 0.3, delay: (delay + 300) / 1000 },
                }}
                style={{
                  ...((isWide ? { width: '100%' } : { maxWidth: '70%', left: '15%' })),
                }}
              />
            )
          })}
        </div>

        {/* Bottom staircase panels - rise upward */}
        <div className="absolute inset-x-0 bottom-0 overflow-hidden">
          {Array.from({ length: 12 }).map((_, i) => {
            const delay = i * BOTTOM_STAGGER + 200
            const panelHeight = 160 + (i % 4) * 30
            const isWide = i % 3 === 0
            return (
              <motion.div
                key={`bottom-${i}`}
                className="absolute inset-x-0 bg-edwins-dark"
                style={{ height: `${panelHeight}px` }}
                initial={{
                  y: '100%',
                  opacity: 0.92,
                }}
                animate={{
                  y: '-100%',
                  opacity: 0,
                }}
                transition={{
                  duration: STAIRCASE_DURATION / 1000,
                  delay: delay / 1000,
                  ease: 'easeOut',
                  opacity: { duration: 0.3, delay: (delay + 300) / 1000 },
                }}
                style={{
                  ...(isWide
                    ? { width: '100%' }
                    : { maxWidth: '65%', right: '17%' }),
                }}
              />
            )
          })}
        </div>

        {/* Complete the transition */}
        <motion.div
          className="absolute inset-0"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{
            duration: 0.1,
            delay: (14 * TOP_STAGGER + 12 * BOTTOM_STAGGER + 400) / 1000,
          }}
          onAnimationComplete={() => onComplete?.()}
        />
      </div>
    </AnimatePresence>
  )
}
