import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { IconDevices } from "@tabler/icons-react";
import { renderWithProviders } from "@/test/utils";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
    it("renders the title and optional description", () => {
        renderWithProviders(
            <EmptyState icon={IconDevices} title="No devices yet" description="They will appear here." />,
        );

        expect(screen.getByText("No devices yet")).toBeInTheDocument();
        expect(screen.getByText("They will appear here.")).toBeInTheDocument();
    });

    it("renders an optional action when provided", () => {
        renderWithProviders(
            <EmptyState icon={IconDevices} title="No policies" action={<button>New policy</button>} />,
        );

        expect(screen.getByRole("button", { name: /new policy/i })).toBeInTheDocument();
    });

    it("renders no action element when omitted", () => {
        renderWithProviders(<EmptyState icon={IconDevices} title="No policies" />);

        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });
});
