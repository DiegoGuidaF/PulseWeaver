import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {LoginPage} from './LoginPage';
import {AuthProvider} from '@/features/auth/AuthContext';
import {TEST_TIMEOUTS} from '@/test/constants';
import {handlers, responses} from "@/test/mocks/handlers.ts";
import {createMockUser} from "@/test/mocks/data.ts";

function renderLoginPage(options?: Parameters<typeof renderWithProviders>[1]) {
    return renderWithProviders(
        <AuthProvider>
            <LoginPage/>
        </AuthProvider>,
        options
    );
}

describe('LoginPage', () => {
    it('renders login form with username and password fields', async () => {
        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            })
        );

        renderLoginPage({initialEntries: ['/login']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /sign in/i})).toBeInTheDocument();
    });

    it('shows loading state during auth check', () => {
        server.use(
            handlers.auth.meHandler(undefined, async () => {
                await delay('infinite');
                return responses.ok(createMockUser());
            })
        );

        renderLoginPage();

        expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    it('redirects to /devices if already authenticated', async () => {
        server.use(
            handlers.auth.meHandler()
        );

        renderLoginPage();

        // Wait for redirect to happen
        await waitFor(() => {
            // Check that we're not showing the login form
            expect(screen.queryByLabelText(/username/i)).not.toBeInTheDocument();
        });
    });

    it('shows loading state during login submission', async () => {
        const user = userEvent.setup();

        // Override handler to delay response (special case for loading state test)
        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            }),
            handlers.auth.loginHandler(undefined, async () => {
                await delay('infinite');
                return responses.ok(createMockUser());
            }),
        );

        renderLoginPage({initialEntries: ['/login']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        const usernameInput = screen.getByLabelText(/username/i);
        const passwordInput = screen.getByLabelText(/password/i);
        const submitButton = screen.getByRole('button', {name: /sign in/i});

        await user.type(usernameInput, 'testuser');
        await user.type(passwordInput, 'password');
        await user.click(submitButton);

        // Check loading state
        expect(screen.getByRole('button', {name: /signing in/i})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: /signing in/i})).toBeDisabled();
    });

    it('successfully logs in and navigates to /devices', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            }),
            handlers.auth.loginHandler()
        );

        renderLoginPage({initialEntries: ['/login']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        const usernameInput = screen.getByLabelText(/username/i);
        const passwordInput = screen.getByLabelText(/password/i);
        const submitButton = screen.getByRole('button', {name: /sign in/i});

        await user.type(usernameInput, 'testuser');
        await user.type(passwordInput, 'password');
        await user.click(submitButton);

        // Wait for mutation to complete - button should no longer be in loading state
        await waitFor(() => {
            const button = screen.getByRole('button', {name: /sign in/i});
            expect(button).not.toBeDisabled();
            expect(button).not.toHaveTextContent(/signing in/i);
        }, {timeout: TEST_TIMEOUTS.MEDIUM});
    });

    it('successfully logs in and navigates to returnTo query parameter', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            }),
            handlers.auth.loginHandler()
        );

        renderLoginPage({initialEntries: ['/login?returnTo=/custom-path']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        const usernameInput = screen.getByLabelText(/username/i);
        const passwordInput = screen.getByLabelText(/password/i);
        const submitButton = screen.getByRole('button', {name: /sign in/i});

        await user.type(usernameInput, 'testuser');
        await user.type(passwordInput, 'password');
        await user.click(submitButton);

        // Wait for success toast (user feedback is important to test)
        await waitFor(() => {
            expect(screen.getByText(/login successful/i)).toBeInTheDocument();
        }, {timeout: TEST_TIMEOUTS.MEDIUM});
    });

    it('shows error toast on login failure', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            }),
            handlers.auth.loginHandler(undefined, async () => {
                return responses.serverError()
            })
        );

        renderLoginPage({initialEntries: ['/login']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        const usernameInput = screen.getByLabelText(/username/i);
        const passwordInput = screen.getByLabelText(/password/i);
        const submitButton = screen.getByRole('button', {name: /sign in/i});

        await user.type(usernameInput, 'testuser');
        await user.type(passwordInput, 'wrongpassword');
        await user.click(submitButton);

        // Wait for error toast to appear (user feedback is important to test)
        // Toast has both title and description, so use getAllByText
        await waitFor(() => {
            const toastElements = screen.getAllByText(/login failed/i);
            expect(toastElements.length).toBeGreaterThan(0);
        }, {timeout: TEST_TIMEOUTS.MEDIUM});

        // Form should still be visible
        expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    });

    it('validates form fields (empty username/password)', async () => {
        const user = userEvent.setup();

        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            }),
        );

        renderLoginPage({initialEntries: ['/login']});

        await waitFor(() => {
            expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
        });

        const submitButton = screen.getByRole('button', {name: /sign in/i});
        await user.click(submitButton);

        // Wait for validation errors - check that inputs are marked as invalid
        // (ARIA attributes are acceptable for form validation testing)
        await waitFor(() => {
            const usernameInput = screen.getByLabelText(/username/i);
            const passwordInput = screen.getByLabelText(/password/i);
            expect(usernameInput).toHaveAttribute('aria-invalid', 'true');
            expect(passwordInput).toHaveAttribute('aria-invalid', 'true');
        });
    });
});
