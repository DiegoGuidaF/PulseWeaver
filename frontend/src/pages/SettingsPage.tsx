import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useAuth } from "@/features/auth/AuthContext";
import { useAdminUpdateUser } from "@/features/auth/hooks/useAdminUpdateUser";
import { useChangePassword } from "@/features/auth/hooks/useChangePassword";
import { useDeleteUser } from "@/features/auth/hooks/useDeleteUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useUpdateMe } from "@/features/auth/hooks/useUpdateMe";
import { UserRole } from "@/lib/api";

const profileSchema = z.object({
  display_name: z.string().trim().min(1).max(50).optional(),
  username: z.string().trim().min(3).max(32).regex(/^[a-zA-Z0-9_-]+$/).optional(),
  email: z.string().email().optional().or(z.literal("")),
});

const passwordSchema = z.object({
  current_password: z.string().min(1, "Current password is required"),
  password: z.string().min(8).max(72),
});

export function SettingsPage() {
  const { user } = useAuth();
  const updateMe = useUpdateMe();
  const changePassword = useChangePassword();
  const listUsers = useListUsers({ enabled: user?.role === UserRole.ADMIN });
  const adminUpdateUser = useAdminUpdateUser();
  const deleteUser = useDeleteUser();
  const [displayNameEdits, setDisplayNameEdits] = useState<Record<number, string>>({});
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null);

  const profileForm = useForm<z.infer<typeof profileSchema>>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      display_name: user?.display_name ?? "",
      username: user?.username ?? "",
      email: user?.email ?? "",
    },
  });

  const passwordForm = useForm<z.infer<typeof passwordSchema>>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      current_password: "",
      password: "",
    },
  });

  const adminUsers = useMemo(() => listUsers.data ?? [], [listUsers.data]);

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

    updateMe.mutate({ body });
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
        },
      }
    );
  }

  function handleRoleToggle(targetUserId: number, currentRole: string) {
    const nextRole = currentRole === UserRole.ADMIN ? UserRole.USER : UserRole.ADMIN;
    adminUpdateUser.mutate({
      path: { user_id: targetUserId },
      body: { role: nextRole },
    });
  }

  function handleAdminDisplayNameSave(targetUserId: number) {
    const displayName = displayNameEdits[targetUserId]?.trim();
    if (!displayName) return;

    adminUpdateUser.mutate({
      path: { user_id: targetUserId },
      body: { display_name: displayName },
    });
  }

  function handleDeleteUser(targetUserId: number, username: string) {
    setDeleteTarget({ id: targetUserId, username });
  }

  function confirmDeleteUser() {
    if (!deleteTarget) return;
    deleteUser.mutate(
      { path: { user_id: deleteTarget.id } },
      { onSettled: () => setDeleteTarget(null) },
    );
  }

  return (
    <>
    <AlertDialog open={deleteTarget !== null} onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete user</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete{" "}
            <span className="font-semibold">{deleteTarget?.username}</span>?
            This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => setDeleteTarget(null)} />
          <AlertDialogAction
            onClick={confirmDeleteUser}
            disabled={deleteUser.isPending}
          >
            {deleteUser.isPending ? "Deleting..." : "Delete"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    <div className="w-full max-w-5xl space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">
          Manage your profile, password, and users.
        </p>
      </div>

      {user?.must_change_password && (
        <Card className="border-amber-500/30">
          <CardHeader>
            <CardTitle>Password change required</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            You must set a new password before using the rest of the application.
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>My profile</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...profileForm}>
            <form
              onSubmit={profileForm.handleSubmit(submitProfile)}
              className="space-y-4"
            >
              <FormField
                control={profileForm.control}
                name="display_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Display name</FormLabel>
                    <FormControl>
                      <Input placeholder="Your display name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={profileForm.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <Input placeholder="Username" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={profileForm.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input placeholder="you@example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button type="submit" disabled={updateMe.isPending}>
                {updateMe.isPending ? "Saving..." : "Save profile"}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Change password</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...passwordForm}>
            <form
              onSubmit={passwordForm.handleSubmit(submitPassword)}
              className="space-y-4"
            >
              <FormField
                control={passwordForm.control}
                name="current_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Current password</FormLabel>
                    <FormControl>
                      <Input type="password" autoComplete="current-password" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={passwordForm.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>New password</FormLabel>
                    <FormControl>
                      <Input type="password" autoComplete="new-password" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button type="submit" disabled={changePassword.isPending}>
                {changePassword.isPending ? "Updating..." : "Update password"}
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>

      {user?.role === UserRole.ADMIN && !user.must_change_password && (
        <Card>
          <CardHeader>
            <CardTitle>Users (admin)</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Display name</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {adminUsers.map((adminUser) => {
                  const isSelf = adminUser.id === user.id;
                  return (
                    <TableRow key={adminUser.id}>
                      <TableCell className="font-medium">{adminUser.username}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Input
                            value={displayNameEdits[adminUser.id] ?? adminUser.display_name}
                            onChange={(event) =>
                              setDisplayNameEdits((prev) => ({
                                ...prev,
                                [adminUser.id]: event.target.value,
                              }))
                            }
                            disabled={isSelf || adminUpdateUser.isPending}
                          />
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={isSelf || adminUpdateUser.isPending}
                            onClick={() => handleAdminDisplayNameSave(adminUser.id)}
                          >
                            Save
                          </Button>
                        </div>
                      </TableCell>
                      <TableCell>{adminUser.role}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={isSelf || adminUpdateUser.isPending}
                            onClick={() => handleRoleToggle(adminUser.id, adminUser.role)}
                          >
                            {adminUser.role === UserRole.ADMIN ? "Demote" : "Promote"}
                          </Button>
                          <Button
                            type="button"
                            variant="destructive"
                            size="sm"
                            disabled={isSelf || deleteUser.isPending}
                            onClick={() => handleDeleteUser(adminUser.id, adminUser.username)}
                          >
                            Delete
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
    </>
  );
}
