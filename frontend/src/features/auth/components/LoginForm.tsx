import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Stack, TextInput } from "@mantine/core";
import { useLogin } from "@/features/auth/hooks/useLogin";
import { zAuthRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const loginSchema = zAuthRequest;

export function LoginForm() {
  const loginMutation = useLogin();

  const form = useForm<z.infer<typeof loginSchema>>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: "",
      password: "",
    },
  });

  function onSubmit(values: z.infer<typeof loginSchema>) {
    loginMutation.mutate({ body: values });
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <Stack gap="md">
        <TextInput
          label="Username"
          placeholder="Enter your username"
          error={form.formState.errors.username?.message}
          {...form.register("username")}
        />
        <TextInput
          label="Password"
          type="password"
          placeholder="Enter your password"
          error={form.formState.errors.password?.message}
          {...form.register("password")}
        />
        <Button
          type="submit"
          fullWidth
          disabled={loginMutation.isPending}
        >
          {loginMutation.isPending ? "Signing in..." : "Sign in"}
        </Button>
      </Stack>
    </form>
  );
}
