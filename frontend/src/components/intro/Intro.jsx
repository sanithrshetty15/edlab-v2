import { useCallback, useEffect, useRef, useState } from 'react'
import IntroAnimation from './IntroAnimation'
import { SITE_REVEAL_EVENT } from '../../utils/animations'
import { prefersReducedMotion } from '../../utils/reducedMotion'

const INTRO_FALLBACK_MS = 9000
let introPlayed = false

export default function Intro({ children, onFinished }) {
  const [introVisible, setIntroVisible] = useState(
    () => !introPlayed && !prefersReducedMotion(),
  )
  const [revealStarted, setRevealStarted] = useState(() => !introVisible)
  const finishedRef = useRef(false)
  const announcedRef = useRef(false)
  const onFinishedRef = useRef(onFinished)

  useEffect(() => {
    onFinishedRef.current = onFinished
  }, [onFinished])

  const announceFinished = useCallback(() => {
    if (announcedRef.current) return

    announcedRef.current = true
    onFinishedRef.current?.()
  }, [])

  const finish = useCallback(() => {
    if (finishedRef.current) return

    finishedRef.current = true
    introPlayed = true
    setRevealStarted(true)
    setIntroVisible(false)
    announceFinished()
  }, [announceFinished])

  useEffect(() => {
    if (!introVisible) announceFinished()
  }, [introVisible, announceFinished])

  const startReveal = useCallback(() => {
    setRevealStarted(true)
    window.dispatchEvent(new Event(SITE_REVEAL_EVENT))
  }, [])

  useEffect(() => {
    if (!introVisible) return undefined

    const fallback = window.setTimeout(finish, INTRO_FALLBACK_MS)
    return () => window.clearTimeout(fallback)
  }, [introVisible, finish])

  return (
    <>
      {introVisible && (
        <IntroAnimation onComplete={finish} onRevealStart={startReveal} />
      )}
      <div
        className={`transition-opacity duration-[850ms] ease-out ${
          revealStarted ? 'opacity-100' : 'opacity-0'
        }`}
        inert={introVisible ? true : undefined}
      >
        {children}
      </div>
    </>
  )
}
