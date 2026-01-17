// Lane types and utilities for champion filtering/sorting

export type Lane = 'top' | 'jungle' | 'mid' | 'bot' | 'support'

export const LANE_ORDER: Lane[] = ['top', 'jungle', 'mid', 'bot', 'support']

export const LANE_DISPLAY_NAMES: Record<Lane, string> = {
  top: 'Top',
  jungle: 'Jungle',
  mid: 'Mid',
  bot: 'Bot',
  support: 'Support',
}

// Get the primary lane for a champion (first in the array)
export function getPrimaryLane(lanes: string[] | undefined): Lane {
  if (!lanes || lanes.length === 0) return 'mid'
  return (lanes[0] as Lane) || 'mid'
}

// Check if a champion plays a specific lane
export function playsLane(lanes: string[] | undefined, lane: Lane): boolean {
  if (!lanes) return false
  return lanes.includes(lane)
}

// Get lane index for sorting (0-4)
export function getLaneIndex(lane: Lane): number {
  return LANE_ORDER.indexOf(lane)
}
