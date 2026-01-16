/**
 * 6-7 meme overlay component
 * Displays a faded "6 7" in the background when triggered
 */
export function SixSevenOverlay({ show }: { show: boolean }) {
  if (!show) return null

  return (
    <div className="six-seven-overlay" aria-hidden="true">
      6 7
    </div>
  )
}
