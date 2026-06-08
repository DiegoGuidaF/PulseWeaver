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

interface ParsedCidr {
    /** IPv4 (or 4-in-6 mapped) ranges measure host bits on the 32-bit scale. */
    isV4: boolean;
    /** Prefix length, normalized to the IPv4 scale for mapped addresses. */
    bits: number;
}

function parseCidr(value: string): ParsedCidr | null {
    if (!isValidCidr(value)) return null;

    const slash = value.lastIndexOf("/");
    const addr = value.slice(0, slash);
    let bits = Number(value.slice(slash + 1));

    // 4-in-6 (::ffff:a.b.c.d) is measured on the IPv4 scale, like the backend.
    const mapped = MAPPED_V4_RE.test(addr);
    const isV4 = !addr.includes(":") || mapped;
    if (mapped) bits -= 96;

    return { isV4, bits };
}

/** Abbreviates a large count to a short K/M/B form, exact below 1000. */
function abbreviateCount(n: number): string {
    if (n < 1_000) return n.toLocaleString();
    const trim = (x: number) => x.toFixed(1).replace(/\.0$/, "");
    if (n < 1_000_000) return `~${trim(n / 1_000)}K`;
    if (n < 1_000_000_000) return `~${trim(n / 1_000_000)}M`;
    return `~${trim(n / 1_000_000_000)}B`;
}

/**
 * Short, human address-space size for a CIDR — e.g. "256 addresses",
 * "~16.8M addresses", or "2^96 addresses" for IPv6 (whose counts are
 * astronomically large). Null when the range is invalid.
 */
export function formatAddressCount(value: string): string | null {
    const parsed = parseCidr(value);
    if (!parsed) return null;

    if (parsed.isV4) {
        const count = 2 ** (32 - parsed.bits);
        return `${abbreviateCount(count)} ${count === 1 ? "address" : "addresses"}`;
    }
    return `2^${128 - parsed.bits} addresses`;
}

/**
 * Classifies how much address space a CIDR covers, mirroring the backend.
 * Returns "normal" for anything invalid so callers can rely on isValidCidr
 * separately for the malformed case.
 */
export function classifyCidr(value: string): CidrBand {
    const parsed = parseCidr(value);
    if (!parsed) return "normal";

    const rejectMax = parsed.isV4 ? REJECT_MAX_BITS_V4 : REJECT_MAX_BITS_V6;
    const warnMax = parsed.isV4 ? WARN_MAX_BITS_V4 : WARN_MAX_BITS_V6;

    if (parsed.bits <= rejectMax) return "reject";
    if (parsed.bits <= warnMax) return "warn";
    return "normal";
}

/**
 * Returns a blast-radius warning for a large-but-allowed CIDR, or null when the
 * range is narrow, invalid, or too broad (the latter blocks submit instead).
 */
export function broadCidrWarning(value: string): string | null {
    if (classifyCidr(value) !== "warn") return null;
    return `Covers ${formatAddressCount(value)} — everyone in it will match this policy once it is granted hosts or bypass.`;
}

/**
 * Error shown when a CIDR is broad enough to be rejected by the server. Quantifies
 * the range so its voice matches the warn-band copy.
 */
export function cidrTooBroadError(value: string): string {
    const count = formatAddressCount(value);
    const size = count ? `covers ${count}` : "covers an entire network operator's address space";
    return `Too broad: ${size}. The broadest allowed prefix is /9 (IPv4) or /33 (IPv6).`;
}

/**
 * Informational size note for a narrow (normal-band) CIDR — e.g. "Covers 256
 * addresses." Null when the range is invalid, broad, or too broad (those surface
 * their own warning/error instead).
 */
export function normalCidrNote(value: string): string | null {
    if (classifyCidr(value) !== "normal") return null;
    const count = formatAddressCount(value);
    return count ? `Covers ${count}.` : null;
}
