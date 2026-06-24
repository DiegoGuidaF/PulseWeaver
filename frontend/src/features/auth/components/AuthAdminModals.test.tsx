import { afterEach, describe, expect, it, vi } from "vitest";
import { notifications } from "@mantine/notifications";
import { screen, waitFor } from "@testing-library/react";
import { http } from "msw";
import { useState } from "react";
import { Route, Routes } from "react-router-dom";
import { CreateUserModal } from "./CreateUserModal";
import { DeleteUserModal } from "./DeleteUserModal";
import { RoleChangeModal, type PendingRole } from "./RoleChangeModal";
import { UserRole } from "@/lib/api";
import { createMockUser } from "@/test/mocks/data";
import { endpoints, responses } from "@/test/mocks/handlers";
import { server } from "@/test/setup";
import { TEST_TIMEOUTS } from "@/test/constants";
import { renderWithProviders, setupUser } from "@/test/utils";

function renderCreateUserModal(onClose = vi.fn()) {
    renderWithProviders(<CreateUserModal opened onClose={onClose} />);
    return { onClose };
}

function RoleChangeHarness({ initialRole }: { initialRole: PendingRole }) {
    const [pendingRole, setPendingRole] = useState<PendingRole | null>(initialRole);

    return (
        <>
            <button type="button" onClick={() => setPendingRole(initialRole)}>
                Open role modal
            </button>
            <RoleChangeModal pendingRole={pendingRole} onClose={() => setPendingRole(null)} />
        </>
    );
}

describe("CreateUserModal", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("validates username length and allowed characters", async () => {
        const user = setupUser();
        renderCreateUserModal();

        await user.type(screen.getByLabelText(/username/i), "ab");
        await user.type(screen.getByLabelText(/display name/i), "Alice User");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        expect(await screen.findByText("Username must be at least 3 characters")).toBeInTheDocument();

        await user.clear(screen.getByLabelText(/username/i));
        await user.type(screen.getByLabelText(/username/i), "Alice User");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        expect(await screen.findByText("Use only lowercase letters, numbers, hyphens, and underscores")).toBeInTheDocument();

        await user.clear(screen.getByLabelText(/username/i));
        await user.type(screen.getByLabelText(/username/i), "a".repeat(33));
        await user.click(screen.getByRole("button", { name: /create user/i }));

        expect(await screen.findByText("Username must be 32 characters or fewer")).toBeInTheDocument();
    });

    it("requires a display name", async () => {
        const user = setupUser();
        renderCreateUserModal();

        await user.type(screen.getByLabelText(/username/i), "alice");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        expect(await screen.findByText("Display name is required")).toBeInTheDocument();
    });

    it("validates email format", async () => {
        const user = setupUser();
        renderCreateUserModal();

        await user.type(screen.getByLabelText(/username/i), "alice");
        await user.type(screen.getByLabelText(/display name/i), "Alice User");
        await user.type(screen.getByLabelText(/email/i), "not-an-email");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        expect(await screen.findByText("Enter a valid email address")).toBeInTheDocument();
    });

    it("creates a user with a trimmed email, closes, notifies, and navigates to detail", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");
        const createCall = vi.fn();

        server.use(
            http.post(endpoints.adminUsers, async ({ request }) => {
                createCall(await request.json());
                return responses.created(
                    createMockUser({
                        id: 42,
                        username: "alice",
                        display_name: "Alice User",
                        email: "alice@example.com",
                    }),
                );
            }),
        );

        function TestApp() {
            const [opened, setOpened] = useState(true);
            return (
                <Routes>
                    <Route
                        path="/"
                        element={<CreateUserModal opened={opened} onClose={() => setOpened(false)} />}
                    />
                    <Route path="/access/users/:id" element={<div>User detail route</div>} />
                </Routes>
            );
        }

        renderWithProviders(<TestApp />);

        await user.type(screen.getByLabelText(/username/i), "alice");
        await user.type(screen.getByLabelText(/display name/i), "Alice User");
        await user.type(screen.getByLabelText(/email/i), "  alice@example.com  ");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        await waitFor(
            () => {
                expect(createCall).toHaveBeenCalledWith({
                    username: "alice",
                    display_name: "Alice User",
                    email: "alice@example.com",
                });
                expect(show).toHaveBeenCalledWith({ color: "green", message: "User created" });
                expect(screen.getByText("User detail route")).toBeInTheDocument();
                expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("shows a friendly duplicate-username message for conflicts", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");

        server.use(
            http.post(endpoints.adminUsers, () =>
                responses.custom({ error: "username already exists" }, 409),
            ),
        );

        renderCreateUserModal();

        await user.type(screen.getByLabelText(/username/i), "alice");
        await user.type(screen.getByLabelText(/display name/i), "Alice User");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        await waitFor(
            () =>
                expect(show).toHaveBeenCalledWith({
                    color: "red",
                    title: "Failed to create user",
                    message: "A user with this username already exists.",
                }),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("uses the API error message for non-conflict failures", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");

        server.use(
            http.post(endpoints.adminUsers, () =>
                responses.serverError({ error: "database unavailable" }),
            ),
        );

        renderCreateUserModal();

        await user.type(screen.getByLabelText(/username/i), "alice");
        await user.type(screen.getByLabelText(/display name/i), "Alice User");
        await user.click(screen.getByRole("button", { name: /create user/i }));

        await waitFor(
            () =>
                expect(show).toHaveBeenCalledWith({
                    color: "red",
                    title: "Failed to create user",
                    message: "database unavailable",
                }),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });
});

describe("DeleteUserModal", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders the target user copy", () => {
        renderWithProviders(
            <DeleteUserModal
                deleteTarget={{ id: 7, username: "alice" }}
                onClose={vi.fn()}
            />,
        );

        expect(screen.getByText(/Are you sure you want to delete/i)).toBeInTheDocument();
        expect(screen.getByText("alice")).toBeInTheDocument();
        expect(screen.getByText(/This action cannot be undone/i)).toBeInTheDocument();
    });

    it("deletes the target user and shows success feedback", async () => {
        const user = setupUser();
        const onClose = vi.fn();
        const show = vi.spyOn(notifications, "show");
        const deleteCall = vi.fn();

        server.use(
            http.delete(endpoints.adminUserById, ({ params }) => {
                deleteCall(params.userId);
                return responses.noContent();
            }),
        );

        renderWithProviders(
            <DeleteUserModal
                deleteTarget={{ id: 7, username: "alice" }}
                onClose={onClose}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Delete" }));

        await waitFor(
            () => {
                expect(deleteCall).toHaveBeenCalledWith("7");
                expect(show).toHaveBeenCalledWith({ color: "green", message: "User deleted" });
                expect(onClose).toHaveBeenCalled();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("shows error feedback when delete fails", async () => {
        const user = setupUser();
        const onClose = vi.fn();
        const show = vi.spyOn(notifications, "show");

        server.use(
            http.delete(endpoints.adminUserById, () =>
                responses.forbidden({ error: "cannot delete self" }),
            ),
        );

        renderWithProviders(
            <DeleteUserModal
                deleteTarget={{ id: 7, username: "alice" }}
                onClose={onClose}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Delete" }));

        await waitFor(
            () => {
                expect(show).toHaveBeenCalledWith({
                    color: "red",
                    title: "Failed to delete user",
                    message: "cannot delete self",
                });
                expect(onClose).toHaveBeenCalled();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("does not mutate when cancelled", async () => {
        const user = setupUser();
        const onClose = vi.fn();
        const deleteCall = vi.fn();

        server.use(
            http.delete(endpoints.adminUserById, () => {
                deleteCall();
                return responses.noContent();
            }),
        );

        renderWithProviders(
            <DeleteUserModal
                deleteTarget={{ id: 7, username: "alice" }}
                onClose={onClose}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Cancel" }));

        expect(onClose).toHaveBeenCalled();
        expect(deleteCall).not.toHaveBeenCalled();
    });
});

describe("RoleChangeModal", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    const promoteRole: PendingRole = {
        userId: 7,
        username: "alice",
        targetRole: "admin",
    };

    const demoteRole: PendingRole = {
        userId: 7,
        username: "alice",
        targetRole: "user",
    };

    it("requires a valid initial password and resets state on close", async () => {
        const user = setupUser();
        const promoteCall = vi.fn();

        server.use(
            http.post(endpoints.promoteUser, () => {
                promoteCall();
                return responses.ok(createMockUser({ role: UserRole.ADMIN }));
            }),
        );

        renderWithProviders(<RoleChangeHarness initialRole={promoteRole} />);

        await user.type(screen.getByLabelText(/initial password/i), "short");
        await user.click(screen.getByRole("button", { name: "Confirm" }));

        expect(await screen.findByText("Password must be at least 8 characters.")).toBeInTheDocument();
        expect(promoteCall).not.toHaveBeenCalled();

        await user.click(screen.getByRole("button", { name: "Cancel" }));
        await user.click(screen.getByRole("button", { name: /open role modal/i }));

        const passwordInput = screen.getByLabelText(/initial password/i);
        expect(passwordInput).toHaveValue("");
        expect(screen.queryByText("Password must be at least 8 characters.")).not.toBeInTheDocument();
    });

    it("promotes with the password payload and shows success feedback", async () => {
        const user = setupUser();
        const onClose = vi.fn();
        const show = vi.spyOn(notifications, "show");
        const promoteCall = vi.fn();

        server.use(
            http.post(endpoints.promoteUser, async ({ params, request }) => {
                promoteCall({ userId: params.userId, body: await request.json() });
                return responses.ok(createMockUser({ id: 7, role: UserRole.ADMIN }));
            }),
        );

        renderWithProviders(<RoleChangeModal pendingRole={promoteRole} onClose={onClose} />);

        await user.type(screen.getByLabelText(/initial password/i), "Initial123");
        await user.click(screen.getByRole("button", { name: "Confirm" }));

        await waitFor(
            () => {
                expect(promoteCall).toHaveBeenCalledWith({
                    userId: "7",
                    body: { password: "Initial123" },
                });
                expect(show).toHaveBeenCalledWith({ color: "green", message: "User promoted to admin" });
                expect(onClose).toHaveBeenCalled();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("demotes without a password body and shows success feedback", async () => {
        const user = setupUser();
        const onClose = vi.fn();
        const show = vi.spyOn(notifications, "show");
        const demoteCall = vi.fn();

        server.use(
            http.post(endpoints.demoteUser, async ({ params, request }) => {
                demoteCall({ userId: params.userId, body: await request.text() });
                return responses.ok(createMockUser({ id: 7, role: UserRole.USER }));
            }),
        );

        renderWithProviders(<RoleChangeModal pendingRole={demoteRole} onClose={onClose} />);

        await user.click(screen.getByRole("button", { name: "Confirm" }));

        await waitFor(
            () => {
                expect(demoteCall).toHaveBeenCalledWith({ userId: "7", body: "" });
                expect(show).toHaveBeenCalledWith({ color: "green", message: "User demoted" });
                expect(onClose).toHaveBeenCalled();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("shows promote error feedback", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");

        server.use(
            http.post(endpoints.promoteUser, () =>
                responses.forbidden({ error: "role changes are disabled" }),
            ),
        );

        renderWithProviders(<RoleChangeModal pendingRole={promoteRole} onClose={vi.fn()} />);

        await user.type(screen.getByLabelText(/initial password/i), "Initial123");
        await user.click(screen.getByRole("button", { name: "Confirm" }));

        await waitFor(
            () =>
                expect(show).toHaveBeenCalledWith({
                    color: "red",
                    title: "Failed to promote user",
                    message: "role changes are disabled",
                }),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });

    it("shows demote error feedback", async () => {
        const user = setupUser();
        const show = vi.spyOn(notifications, "show");

        server.use(
            http.post(endpoints.demoteUser, () =>
                responses.forbidden({ error: "cannot demote superadmin" }),
            ),
        );

        renderWithProviders(<RoleChangeModal pendingRole={demoteRole} onClose={vi.fn()} />);

        await user.click(screen.getByRole("button", { name: "Confirm" }));

        await waitFor(
            () =>
                expect(show).toHaveBeenCalledWith({
                    color: "red",
                    title: "Failed to demote user",
                    message: "cannot demote superadmin",
                }),
            { timeout: TEST_TIMEOUTS.MEDIUM },
        );
    });
});
