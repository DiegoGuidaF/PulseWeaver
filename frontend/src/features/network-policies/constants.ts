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

// ── Broadness bands ─────────────────────────────────────────────────────────
// Mirror of the backend guard in internal/networkpolicies/network_policy.go.
// IPv4 reasons by host count; IPv6 by allocation structure. The server is
// authoritative for the reject band — these exist for live UX only and MUST
// stay in sync with the Go thresholds.

const REJECT_MAX_BITS_V4 = 8; // /0../8  (>= a class-A) is rejected
const WARN_MAX_BITS_V4 = 16; // /9../16 warns
const REJECT_MAX_BITS_V6 = 32; // /0../32 (an ISP allocation or broader) is rejected
const WARN_MAX_BITS_V6 = 47; // /33../47 warns

const MAPPED_V4_RE = /^::ffff:\d{1,3}(\.\d{1,3}){3}$/i;

export type CidrBand = "normal" | "warn" | "reject";

/** Error shown when a CIDR is broad enough to be rejected by the server. */
export const CIDR_TOO_BROAD_ERROR =
    "This range is too broad — it covers an entire network operator's address space. " +
    "The broadest allowed prefix is /9 (IPv4) or /33 (IPv6).";

/**
 * Classifies how much address space a CIDR covers, mirroring the backend.
 * Returns "normal" for anything invalid so callers can rely on isValidCidr
 * separately for the malformed case.
 */
export function classifyCidr(value: string): CidrBand {
    if (!isValidCidr(value)) return "normal";

    const slash = value.lastIndexOf("/");
    const addr = value.slice(0, slash);
    let bits = Number(value.slice(slash + 1));

    // 4-in-6 (::ffff:a.b.c.d) is measured on the IPv4 scale, like the backend.
    const mapped = MAPPED_V4_RE.test(addr);
    const isV4 = !addr.includes(":") || mapped;
    if (mapped) bits -= 96;

    const rejectMax = isV4 ? REJECT_MAX_BITS_V4 : REJECT_MAX_BITS_V6;
    const warnMax = isV4 ? WARN_MAX_BITS_V4 : WARN_MAX_BITS_V6;

    if (bits <= rejectMax) return "reject";
    if (bits <= warnMax) return "warn";
    return "normal";
}

/**
 * Returns a blast-radius warning for a large-but-allowed CIDR, or null when the
 * range is narrow, invalid, or too broad (the latter blocks submit instead).
 */
export function broadCidrWarning(value: string): string | null {
    if (classifyCidr(value) !== "warn") return null;

    const slash = value.lastIndexOf("/");
    const addr = value.slice(0, slash);
    const mapped = MAPPED_V4_RE.test(addr);
    const isV4 = !addr.includes(":") || mapped;
    const tail = " Everyone in it will match this policy once it is granted hosts or bypass.";

    if (isV4) {
        const bits = Number(value.slice(slash + 1)) - (mapped ? 96 : 0);
        const count = 2 ** (32 - bits);
        return `This range covers ~${count.toLocaleString()} addresses.${tail}`;
    }
    return `This range spans many subnets — well beyond a single site.${tail}`;
}
