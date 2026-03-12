import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import { Button, Stack, TextInput } from "@mantine/core";
import { useLogin } from "@/features/auth/hooks/useLogin";
import { zAuthRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const loginSchema = zAuthRequest;

export function LoginForm() {
  const loginMutation = useLogin();

  const form = useForm<z.infer<typeof loginSchema>>({
    validate: zod4Resolver(loginSchema),
    initialValues: {
      username: "",
      password: "",
    },
  });

  function onSubmit(values: z.infer<typeof loginSchema>) {
    loginMutation.mutate({ body: values });
  }

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack gap="md">
        <TextInput
          label="Username"
          placeholder="Enter your username"
          {...form.getInputProps("username")}
        />
        <TextInput
          label="Password"
          type="password"
          placeholder="Enter your password"
          {...form.getInputProps("password")}
        />
        <Button type="submit" fullWidth disabled={loginMutation.isPending}>
          {loginMutation.isPending ? "Signing in..." : "Sign in"}
        </Button>
      </Stack>
    </form>
  );
}
