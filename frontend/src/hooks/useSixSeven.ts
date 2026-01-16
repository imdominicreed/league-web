import { useState, useEffect, useCallback, useRef } from 'react'

/**
 * Easter egg hook for the 6-7 meme
 *
 * Triggers a screen shake effect when:
 * 1. User types "67" anywhere on the page
 * 2. Draft transitions from phase 6 to phase 7
 * 3. Room code contains "67"
 *
 * Reference: https://en.wikipedia.org/wiki/6-7_meme
 * The 6-7 meme originated from Skrilla's song "Doot Doot (6 7)"
 * and became viral through basketball culture. Google added a
 * similar screen shake easter egg in December 2025.
 */
export function useSixSeven(currentPhase: number, roomCode?: string) {
  const [isShaking, setIsShaking] = useState(false)
  const [, setKeystrokeBuffer] = useState('')
  const previousPhase = useRef(currentPhase)
  const bufferTimeoutRef = useRef<ReturnType<typeof setTimeout>>()

  // Check if room code contains 67
  const hasSpecialCode = roomCode?.includes('67') ?? false

  // Trigger shake animation
  const triggerShake = useCallback(() => {
    setIsShaking(true)
    setTimeout(() => setIsShaking(false), 600)
  }, [])

  // Handle keystroke detection for "67"
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore if user is typing in an input
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return
      }

      // Only track number keys
      if (e.key === '6' || e.key === '7') {
        setKeystrokeBuffer(prev => {
          const newBuffer = prev + e.key

          // Check for "67" sequence
          if (newBuffer.includes('67')) {
            triggerShake()
            return ''
          }

          // Keep only last 2 characters
          return newBuffer.slice(-2)
        })

        // Clear buffer after 1 second of no input
        if (bufferTimeoutRef.current) {
          clearTimeout(bufferTimeoutRef.current)
        }
        bufferTimeoutRef.current = setTimeout(() => {
          setKeystrokeBuffer('')
        }, 1000)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      if (bufferTimeoutRef.current) {
        clearTimeout(bufferTimeoutRef.current)
      }
    }
  }, [triggerShake])

  // Handle phase 6 -> 7 transition
  useEffect(() => {
    if (previousPhase.current === 6 && currentPhase === 7) {
      triggerShake()
    }
    previousPhase.current = currentPhase
  }, [currentPhase, triggerShake])

  // Trigger shake on initial load if room has special code
  useEffect(() => {
    if (hasSpecialCode) {
      // Small delay so the effect is noticeable after page load
      const timeout = setTimeout(triggerShake, 500)
      return () => clearTimeout(timeout)
    }
  }, [hasSpecialCode, triggerShake])

  return {
    isShaking,
    hasSpecialCode,
    triggerShake,
  }
}
