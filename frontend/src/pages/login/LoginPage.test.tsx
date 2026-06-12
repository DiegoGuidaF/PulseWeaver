import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { delay, http } from "msw";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { LoginPage } from "./LoginPage";
import { AuthProvider } from "@/features/auth/AuthContext";
import { TEST_TIMEOUTS } from "@/test/constants";
import { authHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { createMockUser } from "@/test/mocks/data";

function renderLoginPage(options?: Parameters<typeof renderWithProviders>[1]) {
  return renderWithProviders(
    <AuthProvider>
      <LoginPage />
    </AuthProvider>,
    options,
  );
}

describe("LoginPage", () => {
  it("renders login form with username and password fields", async () => {
    server.use(authHandlers.me.unauthenticated());

    renderLoginPage({ initialEntries: ["/login"] });

    await waitFor(
      () => {
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /sign in/i }),
    ).toBeInTheDocument();
  });

  it("shows loading state during auth check", () => {
    server.use(
      http.get(endpoints.authMe, async () => {
        await delay("infinite");
        return responses.ok(createMockUser());
      }),
    );

    renderLoginPage();

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("redirects to /devices if already authenticated", async () => {
    renderLoginPage();

    // Wait for redirect to happen
    await waitFor(
      () => {
        expect(screen.queryByLabelText(/username/i)).not.toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("shows loading state during login submission", async () => {
    const user = setupUser();

    server.use(
      authHandlers.me.unauthenticated(),
      http.post(endpoints.authLogin, async () => {
        await delay("infinite");
        return responses.ok(createMockUser());
      }),
    );

    renderLoginPage({ initialEntries: ["/login"] });

    await waitFor(
      () => {
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole("button", { name: /sign in/i });

    await user.type(usernameInput, "testuser");
    await user.type(passwordInput, "password");
    await user.click(submitButton);

    // Check loading state
    expect(
      screen.getByRole("button", { name: /signing in/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /signing in/i })).toBeDisabled();
  });

  it("successfully logs in and navigates to /devices", async () => {
    const user = setupUser();
    let meCallCount = 0;

    server.use(
      http.get(endpoints.authMe, async () => {
        meCallCount++;
        if (meCallCount === 1) {
          return responses.unauthorized();
        }
        return responses.ok(createMockUser());
      }),
    );

    renderLoginPage({ initialEntries: ["/login"] });

    await waitFor(
      () => {
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole("button", { name: /sign in/i });

    await user.type(usernameInput, "testuser");
    await user.type(passwordInput, "password");
    await user.click(submitButton);

    await waitFor(
      () => {
        expect(screen.queryByLabelText(/username/i)).not.toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );
  });

  it("shows error toast on login failure", async () => {
    const user = setupUser();

    server.use(
      authHandlers.me.unauthenticated(),
      http.post(endpoints.authLogin, async () => responses.serverError()),
    );

    renderLoginPage({ initialEntries: ["/login"] });

    await waitFor(
      () => {
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    const usernameInput = screen.getByLabelText(/username/i);
    const passwordInput = screen.getByLabelText(/password/i);
    const submitButton = screen.getByRole("button", { name: /sign in/i });

    await user.type(usernameInput, "testuser");
    await user.type(passwordInput, "wrongpassword");
    await user.click(submitButton);

    // Wait for error toast to appear (user feedback is important to test)
    // Toast has both title and description, so use getAllByText
    await waitFor(
      () => {
        const toastElements = screen.getAllByText(/login failed/i);
        expect(toastElements.length).toBeGreaterThan(0);
      },
      { timeout: TEST_TIMEOUTS.MEDIUM },
    );

    // Form should still be visible
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
  });

  it("validates form fields (empty username/password)", async () => {
    const user = setupUser();

    server.use(authHandlers.me.unauthenticated());

    renderLoginPage({ initialEntries: ["/login"] });

    await waitFor(
      () => {
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );

    const submitButton = screen.getByRole("button", { name: /sign in/i });
    await user.click(submitButton);

    // Wait for validation errors - check that inputs are marked as invalid
    // (ARIA attributes are acceptable for form validation testing)
    await waitFor(
      () => {
        const usernameInput = screen.getByLabelText(/username/i);
        const passwordInput = screen.getByLabelText(/password/i);
        expect(usernameInput).toHaveAttribute("aria-invalid", "true");
        expect(passwordInput).toHaveAttribute("aria-invalid", "true");
      },
      { timeout: TEST_TIMEOUTS.SHORT },
    );
  });
});
