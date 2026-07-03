import { useEffect } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  Button,
  Card,
  Group,
  PasswordInput,
  Stack,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useUpdateMe } from "@/features/auth/hooks/useUpdateMe";
import { useChangePassword } from "@/features/auth/hooks/useChangePassword";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import {
  zUpdateProfileRequest,
  zChangePasswordRequest,
} from "@/lib/api/zod.gen";

const profileSchema = zUpdateProfileRequest;

const passwordSchema = zChangePasswordRequest
  .extend({
    confirm_password: z.string().min(1, "Please confirm your new password"),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  });

interface AccountSettingsProps {
  onDirtyChange?: (dirty: boolean) => void;
}

export function AccountSettings({ onDirtyChange }: AccountSettingsProps) {
  const { user } = useAuth();
  const updateMe = useUpdateMe();
  const changePassword = useChangePassword();

  const profileForm = useForm<z.infer<typeof profileSchema>>({
    validateInputOnBlur: true,
    validate: schemaResolver(profileSchema),
    initialValues: {
      display_name: user?.display_name ?? "",
      username: user?.username ?? "",
      email: user?.email ?? "",
    },
  });

  const passwordForm = useForm<z.infer<typeof passwordSchema>>({
    validateInputOnBlur: true,
    validate: schemaResolver(passwordSchema),
    initialValues: {
      current_password: "",
      password: "",
      confirm_password: "",
    },
  });

  // Keep form in sync with server data. When useUpdateMe invalidates
  // getCurrentUser, fresh user data arrives here. setValues + resetDirty
  // updates both the displayed values and the dirty-check snapshot so
  // isDirty() returns false. Unlike initialize(), this works every time.
  useEffect(() => {
    if (user) {
      const values = {
        display_name: user.display_name ?? "",
        username: user.username ?? "",
        email: user.email ?? "",
      };
      profileForm.setValues(values);
      profileForm.resetDirty(values);
    }
  // profileForm methods are stable refs; only re-run when server data changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.display_name, user?.username, user?.email]);

  const passwordHasContent =
    passwordForm.values.current_password.length > 0 &&
    passwordForm.values.password.length > 0 &&
    passwordForm.values.confirm_password.length > 0;

  // Only the profile form contributes to the unsaved-changes guard.
  // The password form is a stateless "fill-and-submit" flow — partial
  // input there is not meaningful saved state worth guarding.
  const isDirty = profileForm.isDirty();

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  function submitProfile(values: z.infer<typeof profileSchema>) {
    const body: { display_name?: string; username?: string; email?: string } = {};
    const nextDisplayName = values.display_name?.trim() ?? "";
    const nextUsername = values.username?.trim() ?? "";
    const nextEmail = values.email?.trim() ?? "";

    if (nextDisplayName && nextDisplayName !== user?.display_name) {
      body.display_name = nextDisplayName;
    }
    if (nextUsername && nextUsername !== user?.username) {
      body.username = nextUsername;
    }
    if (nextEmail && nextEmail !== user?.email) {
      body.email = nextEmail;
    }

    updateMe.mutate(
      { body },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "Profile updated" });
        },
        onError: (err) => {
          const message =
            toApiError(err).status === 409
              ? "Username or email is already in use."
              : toErrorMessage(err);
          notifications.show({ color: "red", title: "Failed to update profile", message });
        },
      },
    );
  }

  function submitPassword(values: z.infer<typeof passwordSchema>) {
    changePassword.mutate(
      {
        body: {
          current_password: values.current_password,
          password: values.password,
        },
      },
      {
        onSuccess: () => {
          passwordForm.reset();
          notifications.show({ color: "green", message: "Password changed" });
        },
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to change password", message: toErrorMessage(err) }),
      },
    );
  }

  return (
    <Stack gap="xl">
      <Card withBorder>
        <Title order={2} mb="md">My profile</Title>
        <form onSubmit={profileForm.onSubmit(submitProfile)}>
          <Stack gap="md">
            <TextInput
              label="Display name"
              placeholder="Your display name"
              {...profileForm.getInputProps("display_name")}
            />
            <TextInput
              label="Username"
              placeholder="Username"
              {...profileForm.getInputProps("username")}
            />
            <TextInput
              label="Email"
              placeholder="you@example.com"
              {...profileForm.getInputProps("email")}
            />
            <Group gap="sm">
              {isDirty && (
                <>
                  <Button type="submit" loading={updateMe.isPending}>
                    Save profile
                  </Button>
                  <Button variant="outline" onClick={() => profileForm.reset()}>
                    Discard changes
                  </Button>
                </>
              )}
            </Group>
          </Stack>
        </form>
      </Card>

      <Card withBorder>
        <Title order={2} mb="md">Change password</Title>
        <form onSubmit={passwordForm.onSubmit(submitPassword)}>
          <Stack gap="md">
            {/* Hidden username field for password manager accessibility */}
            <input
              type="text"
              autoComplete="username"
              value={user?.username ?? ""}
              readOnly
              hidden
            />
            <PasswordInput
              label="Current password"
              autoComplete="current-password"
              withAsterisk
              {...passwordForm.getInputProps("current_password")}
            />
            <PasswordInput
              label="New password"
              autoComplete="new-password"
              withAsterisk
              {...passwordForm.getInputProps("password")}
            />
            <PasswordInput
              label="Confirm new password"
              autoComplete="new-password"
              withAsterisk
              {...passwordForm.getInputProps("confirm_password")}
            />
            <div>
              <Button
                type="submit"
                loading={changePassword.isPending}
                disabled={changePassword.isPending || !passwordHasContent}
              >
                Update password
              </Button>
            </div>
          </Stack>
        </form>
      </Card>
    </Stack>
  );
}
