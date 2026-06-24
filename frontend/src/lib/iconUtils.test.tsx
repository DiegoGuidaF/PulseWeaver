import { describe, expect, it } from "vitest";
import { IconCheck } from "@tabler/icons-react";
import { screen } from "@testing-library/react";
import {
    makeEmojiRenderer,
    makeTablerRenderer,
    makeUrlRenderer,
    validateIconWithMap,
} from "./iconUtils";
import { renderWithProviders } from "@/test/utils";

describe("iconUtils", () => {
    it("accepts empty, mapped, and emoji icon values", () => {
        const icons = new Map<string, unknown>([["server", IconCheck]]);

        expect(validateIconWithMap(icons, "   ")).toEqual({ ok: true });
        expect(validateIconWithMap(icons, " server ")).toEqual({ ok: true });
        expect(validateIconWithMap(icons, "🖥️")).toEqual({ ok: true });
    });

    it("rejects non-empty values that are neither mapped names nor a single emoji", () => {
        expect(validateIconWithMap(new Map(), "server")).toEqual({
            ok: false,
            reason: "Enter a single emoji or pick an icon from the suggestions.",
        });
        expect(validateIconWithMap(new Map(), "🖥️🖥️").ok).toBe(false);
    });

    it("renders emoji and URL icons with display attributes", () => {
        const Emoji = makeEmojiRenderer("🛡️");
        const UrlIcon = makeUrlRenderer("https://example.com/icon.png");

        renderWithProviders(
            <>
                <Emoji size={18} />
                <UrlIcon size={24} />
            </>,
        );

        expect(screen.getByText("🛡️")).toBeInTheDocument();
        const image = document.querySelector('img[src="https://example.com/icon.png"]');
        expect(image).toBeInTheDocument();
        expect(image).toHaveAttribute("src", "https://example.com/icon.png");
        expect(image).toHaveAttribute("width", "24");
        expect(image).toHaveAttribute("height", "24");
        expect(image).toHaveAttribute("alt", "");
    });

    it("lets caller styles override default Tabler renderer styles", () => {
        const Renderer = makeTablerRenderer(IconCheck, { color: "red" });

        const { container } = renderWithProviders(
            <Renderer size={20} style={{ color: "blue" }} />,
        );

        expect(container.querySelector("svg")).toHaveStyle({ color: "blue" });
    });
});
