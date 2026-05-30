import {
    createTheme,
    darken,
    defaultVariantColorsResolver,
    parseThemeColor,
    type CSSVariablesResolver,
    type VariantColorsResolver,
} from '@mantine/core';

/*
 * PW-64 (a11y / color-contrast): the design is dark-mode-first and light mode was
 * never contrast-checked. Every override below is scoped to the LIGHT scheme; dark
 * mode renders exactly as before. See polish-findings.md for the per-lever node counts.
 */

// Lever 4 — light-variant badges/alerts (teal/green/indigo/orange/violet …) paint
// colored text on a pale tint that lands at 3.0–4.3:1, under WCAG AA (4.5:1). Darken
// the text for the `light` variant only, across every color, so it passes globally
// without per-component edits.
const variantColorResolver: VariantColorsResolver = (input) => {
    const resolved = defaultVariantColorsResolver(input);
    if (input.variant !== 'light') return resolved;

    const parsed = parseThemeColor({
        color: input.color || input.theme.primaryColor,
        theme: input.theme,
    });
    // Theme colors: start from the darkest palette shade (green/teal need it) and
    // nudge further. Custom hex colors aren't on a palette, so darken them harder.
    const base =
        parsed.isThemeColor && parsed.color ? input.theme.colors[parsed.color]?.[9] : parsed.value;
    if (!base) return resolved;

    return {
        ...resolved,
        color: darken(base, parsed.isThemeColor ? 0.2 : 0.45),
    };
};

// Levers 1 & 5 — token overrides that can't live on `theme` (dimmed is a CSS var,
// amber needs a per-scheme value). Wired onto <MantineProvider cssVariablesResolver>.
export const cssVariablesResolver: CSSVariablesResolver = (theme) => ({
    variables: {},
    light: {
        // gray-6 dimmed (#868e96 → 3.32:1 on white) is the single largest finding
        // (~205 nodes: every `c="dimmed"` + nav section labels). gray-7 (#495057)
        // clears AA comfortably (~8:1).
        '--mantine-color-dimmed': theme.colors.gray[7],
        // Contrast-safe amber for body text on light backgrounds (orange-4 fails AA).
        '--pw-amber-text': theme.colors.orange[8],
    },
    dark: {
        // Keep the vivid brand amber on dark, where it passes.
        '--pw-amber-text': theme.colors.orange[4],
    },
});

export const theme = createTheme({
    primaryColor: 'indigo',
    // Lever 2 — in light mode indigo-6 links/anchors land at 4.32:1 (just under AA).
    // Shade 7 (#4263eb, ~5.3:1) clears links, filled buttons, and light-variant text.
    // Dark keeps Mantine's default shade (8).
    primaryShade: { light: 7, dark: 8 },
    // Lever 3 — flip text to dark on filled badges/buttons whose background is light
    // enough (white-on-green "Allow", orange "bypass", …) instead of unreadable white.
    autoContrast: true,
    variantColorResolver,
    defaultRadius: 'md',
    fontFamily: 'Inter, system-ui, -apple-system, sans-serif',
    fontFamilyMonospace: 'ui-monospace, SFMono-Regular, Menlo, monospace',
});
