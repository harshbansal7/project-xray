// A set of visually distinct colors for dynamic outcomes
const DYNAMIC_PALETTE = [
  '#8B5CF6', // Violet
  '#06B6D4', // Cyan
  '#F59E0B', // Amber
  '#10B981', // Emerald
  '#EC4899', // Pink
  '#6366F1', // Indigo
  '#14B8A6', // Teal
  '#F97316', // Orange
  '#84CC16', // Lime
  '#A855F7', // Purple
  '#EF4444', // Red
  '#3B82F6', // Blue
];

/**
 * Simple hash function for strings
 */
function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
}

/**
 * Get a consistent color for an outcome value
 * Generated from palette based on string hash
 */
export function getOutcomeColor(outcome: string): string {
  const normalized = outcome.toLowerCase().trim();
  const hash = hashString(normalized);
  return DYNAMIC_PALETTE[hash % DYNAMIC_PALETTE.length];
}

/**
 * Generate colors for a set of outcomes (for charts)
 * Returns a consistent mapping of outcome -> color
 */
export function generateOutcomeColors(outcomes: string[]): Record<string, string> {
  const colors: Record<string, string> = {};
  
  for (const outcome of outcomes) {
    colors[outcome] = getOutcomeColor(outcome);
  }
  
  return colors;
}


