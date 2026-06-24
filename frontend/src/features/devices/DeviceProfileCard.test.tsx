import { describe, expect, it } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import { DeviceProfileCard } from "@/features/devices/DeviceProfileCard";
import { TEST_TIMEOUTS } from "@/test/constants";
import { createMockDevice } from "@/test/mocks/data";
import { deviceHandlers, endpoints } from "@/test/mocks/handlers";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";

describe("DeviceProfileCard", () => {
  it("shows the dirty-state banner only after edits and reset restores server values", async () => {
    const user = setupUser();

    renderWithProviders(
      <DeviceProfileCard
        deviceId={1}
        device={{ name: "Router", description: "Closet AP", icon: "📡" }}
      />,
    );

    expect(screen.queryByText("Unsaved changes")).not.toBeInTheDocument();

    await user.clear(screen.getByRole("textbox", { name: /name/i }));
    await user.type(screen.getByRole("textbox", { name: /name/i }), "Garage router");

    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Reset" }));

    expect(screen.queryByText("Unsaved changes")).not.toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /name/i })).toHaveValue("Router");
    expect(screen.getByRole("textbox", { name: /description/i })).toHaveValue("Closet AP");
    expect(screen.getByRole("button", { name: "Clear icon override" })).toBeInTheDocument();
  });

  it("submits only changed fields", async () => {
    const user = setupUser();
    let body: unknown;
    server.use(
      http.patch(endpoints.deviceById, async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(createMockDevice({ name: "Garage router" }));
      }),
    );

    renderWithProviders(
      <DeviceProfileCard
        deviceId={1}
        device={{ name: "Router", description: "Closet AP", icon: "📡" }}
      />,
    );

    await user.clear(screen.getByRole("textbox", { name: /name/i }));
    await user.type(screen.getByRole("textbox", { name: /name/i }), "Garage router");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(
      () => {
        expect(body).toEqual({ name: "Garage router" });
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("normalizes empty description and icon values to null", async () => {
    const user = setupUser();
    let body: unknown;
    server.use(
      http.patch(endpoints.deviceById, async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(createMockDevice({ description: null, icon: null }));
      }),
    );

    renderWithProviders(
      <DeviceProfileCard
        deviceId={1}
        device={{ name: "Router", description: "Closet AP", icon: "📡" }}
      />,
    );

    await user.clear(screen.getByRole("textbox", { name: /description/i }));
    await user.click(screen.getByRole("button", { name: "Clear icon override" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(
      () => {
        expect(body).toEqual({ description: null, icon: null });
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("sets the name field error for conflict responses", async () => {
    const user = setupUser();
    server.use(deviceHandlers.update.conflict());

    renderWithProviders(<DeviceProfileCard deviceId={1} device={{ name: "Router" }} />);

    await user.clear(screen.getByRole("textbox", { name: /name/i }));
    await user.type(screen.getByRole("textbox", { name: /name/i }), "Duplicate router");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Name already in use")).toBeInTheDocument();
  });
});
