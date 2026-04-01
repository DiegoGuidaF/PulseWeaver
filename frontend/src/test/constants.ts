/**
 * Timeout constants for async test utilities
 * Based on Testing Library defaults and common patterns
 */
export const TEST_TIMEOUTS = {
  /** Short timeout for fast operations (default: 1000ms) */
  SHORT: 2000,
  /** Medium timeout for typical async operations */
  MEDIUM: 4000,
  /** Long timeout for slow operations or CI environments */
  LONG: 7000,
} as const;
