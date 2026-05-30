import {
    createTheme,
    darken,
    defaultVariantColorsResolver,
    lighten,
    parseThemeColor,
    type CSSVariablesResolver,
    type VariantColorsResolver,
} from '@mantine/core';

/*
 * PW-64 (a11y / color-contrast): the design is dark-mode-first, but light mode was
 * never contrast-checked. The first pass fixed light; re-auditing dark (which had
 * also never been machine-checked) surfaced its own failures — some of which the
 * light-variant darkening below was itself causing in dark. Every override is now
 * scheme-scoped: light gets its darkened text / gray-7 dimmed, dark keeps Mantine's
 * (passing) defaults plus a lighter dimmed. See polish-findings.md for node counts.
 */

// Lever 4 — light-variant badges/alerts (teal/green/indigo/orange/violet …) paint
// colored text on a pale tint. In LIGHT the tint is pale and the text lands at
// 3.0–4.3:1 (under AA 4.5:1), so the text must go DARKER; in DARK the tint is dark
// and the same text (indigo especially, ~2.85:1) must go LIGHTER. We compute both and
// pick per-scheme with light-dark(), so one resolver fixes the `light` variant across
// every color and both schemes without per-component edits. (Lightening on the dark
// tint only ever raises contrast, so colors already passing in dark stay passing.)
const variantColorResolver: VariantColorsResolver = (input) => {
    const resolved = defaultVariantColorsResolver(input);
    if (input.variant !== 'light') return resolved;

    const parsed = parseThemeColor({
        color: input.color || input.theme.primaryColor,
        theme: input.theme,
    });
    if (parsed.isThemeColor && parsed.color) {
        // On a palette: darkest shade darkened for light, a pale shade for dark.
        const palette = input.theme.colors[parsed.color];
        if (!palette) return resolved;
        return {
            ...resolved,
            color: `light-dark(${darken(palette[9], 0.2)}, ${palette[2]})`,
        };
    }
    // Custom hex (off-palette): no shades to pick from, so darken/lighten harder.
    if (!parsed.value) return resolved;
    return {
        ...resolved,
        color: `light-dark(${darken(parsed.value, 0.45)}, ${lighten(parsed.value, 0.6)})`,
    };
};

// Levers 1 & 5 — token overrides that can't live on `theme` (dimmed is a CSS var,
// amber needs a per-scheme value). Wired onto <MantineProvider cssVariablesResolver>.
export const cssVariablesResolver: CSSVariablesResolver = (theme) => ({
    variables: {},
    light: {
        // gray-6 dimmed (#868e96 → 3.32:1 on white) is the single largest LIGHT finding
        // (~205 nodes: every `c="dimmed"` + nav section labels). gray-7 (#495057)
        // clears AA comfortably (~8:1).
        '--mantine-color-dimmed': theme.colors.gray[7],
        // Contrast-safe amber for body text on light backgrounds (orange-4 fails AA).
        '--pw-amber-text': theme.colors.orange[8],
    },
    dark: {
        // Mantine's default dark dimmed is dark-2 (#828282 → 4.03:1 on the dark-7 body,
        // under AA) — the largest DARK finding (~200 nodes: nav section labels + every
        // `c="dimmed"`). gray-5 (#adb5bd, ~6:1) clears AA and still reads as dimmed.
        '--mantine-color-dimmed': theme.colors.gray[5],
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
