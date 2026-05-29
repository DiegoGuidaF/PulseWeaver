import { useForm, schemaResolver } from "@mantine/form";
import { Button, Stack, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useLogin } from "@/features/auth/hooks/useLogin";
import { toErrorMessage } from "@/lib/api-client";
import { zAuthRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const loginSchema = zAuthRequest;

export function LoginForm() {
  const loginMutation = useLogin();

  const form = useForm<z.infer<typeof loginSchema>>({
    validateInputOnBlur: true,
    validate: schemaResolver(loginSchema),
    initialValues: {
      username: "",
      password: "",
    },
  });

  function onSubmit(values: z.infer<typeof loginSchema>) {
    loginMutation.mutate(
      { body: values },
      {
        onError: (err) =>
          notifications.show({ color: "red", title: "Login failed", message: toErrorMessage(err) }),
      },
    );
  }

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack gap="md">
        <TextInput
          label="Username"
          placeholder="Enter your username"
          withAsterisk
          {...form.getInputProps("username")}
        />
        <TextInput
          label="Password"
          type="password"
          placeholder="Enter your password"
          withAsterisk
          {...form.getInputProps("password")}
        />
        <Button type="submit" fullWidth disabled={loginMutation.isPending}>
          {loginMutation.isPending ? "Signing in..." : "Sign in"}
        </Button>
      </Stack>
    </form>
  );
}
