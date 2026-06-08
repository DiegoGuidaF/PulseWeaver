import { z } from "zod";

/**
 * Accepts both IPv4 and IPv6 CIDR ranges using Zod v4's native CIDR primitives
 * (e.g. `192.168.1.0/24` or `2001:db8::/32`). Verified against zod@4.4.3.
 */
const cidrSchema = z.cidrv4().or(z.cidrv6());

/** Example shown in CIDR field placeholders and validation copy. */
export const CIDR_EXAMPLE = "192.168.1.0/24 or 2001:db8::/32";

/** Validation message for an invalid CIDR range. */
export const CIDR_ERROR = `Enter a valid CIDR range, e.g. ${CIDR_EXAMPLE}`;

/** True when `value` is a valid IPv4 or IPv6 CIDR range. */
export function isValidCidr(value: string): boolean {
    return cidrSchema.safeParse(value).success;
}
