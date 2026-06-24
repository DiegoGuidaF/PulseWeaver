import { describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import { PairingCreationForm } from "@/features/device-pairing/PairingCreationForm";
import type { CreatePairingRequest } from "@/lib/api";
import { TEST_TIMEOUTS } from "@/test/constants";
import { createMockDevicePairing } from "@/test/mocks/data";
import { devicePairingHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";

const SERVER_URL_KEY = "pw.pair.serverUrl";

describe("PairingCreationForm", () => {
  it("defaults to the saved server URL when one exists", () => {
    window.localStorage.setItem(SERVER_URL_KEY, "https://devices.example.com");

    renderWithProviders(<PairingCreationForm deviceId={1} onSuccess={vi.fn()} />);

    expect(screen.getByRole("textbox", { name: /device server url/i })).toHaveValue(
      "https://devices.example.com",
    );
  });

  it("defaults to window.location.origin without a saved server URL", () => {
    renderWithProviders(<PairingCreationForm deviceId={1} onSuccess={vi.fn()} />);

    expect(screen.getByRole("textbox", { name: /device server url/i })).toHaveValue(
      window.location.origin,
    );
  });

  it("submits changed config, persists the server URL, and calls onSuccess", async () => {
    const user = setupUser();
    const onSuccess = vi.fn();
    let body: CreatePairingRequest | undefined;
    server.use(
      http.post(endpoints.devicePairings, async ({ request }) => {
        body = (await request.json()) as CreatePairingRequest;
        return HttpResponse.json(createMockDevicePairing({ id: 42 }), { status: 201 });
      }),
    );

    renderWithProviders(<PairingCreationForm deviceId={1} onSuccess={onSuccess} />);

    await user.clear(screen.getByRole("textbox", { name: /device server url/i }));
    await user.type(
      screen.getByRole("textbox", { name: /device server url/i }),
      "https://heartbeat.example.com",
    );
    await user.click(screen.getByRole("radio", { name: "15 min" }));
    await user.click(screen.getByText("Biometric unlock"));
    await user.click(screen.getByText("Lock app settings"));
    await user.click(screen.getByRole("radio", { name: "7 days" }));
    await user.click(screen.getByRole("button", { name: "Generate code →" }));

    await waitFor(
      () => {
        expect(body).toEqual({
          heartbeat_server_url: "https://heartbeat.example.com",
          interval_seconds: 900,
          app_biometric_enabled: false,
          app_settings_locked: true,
          expires_in_hours: 168,
        });
        expect(window.localStorage.getItem(SERVER_URL_KEY)).toBe("https://heartbeat.example.com");
        expect(onSuccess).toHaveBeenCalledWith(expect.objectContaining({ id: 42 }));
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("shows an error notification when pairing creation fails", async () => {
    const user = setupUser();
    server.use(
      http.post(endpoints.devicePairings, () =>
        responses.serverError({ error: "Pairing service unavailable" }),
      ),
    );

    renderWithProviders(<PairingCreationForm deviceId={1} onSuccess={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Generate code →" }));

    expect(await screen.findByText("Failed to create pairing code")).toBeInTheDocument();
    expect(await screen.findByText("Pairing service unavailable")).toBeInTheDocument();
  });

  it("can use the shared success handler without per-test API mocking", async () => {
    const user = setupUser();
    const onSuccess = vi.fn();
    server.use(devicePairingHandlers.create.success({ id: 7 }));

    renderWithProviders(<PairingCreationForm deviceId={1} onSuccess={onSuccess} />);

    await user.click(screen.getByRole("button", { name: "Generate code →" }));

    await waitFor(
      () => {
        expect(onSuccess).toHaveBeenCalledWith(expect.objectContaining({ id: 7 }));
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });
});
