import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useCreateDevice } from "@/features/devices/hooks/useCreateDevice";
import type { CreateDeviceResponse } from "@/lib/api";
import { zCreateDeviceRequest } from "@/lib/api/zod.gen";
import { toast } from "sonner";
import type { z } from "zod";

const formSchema = zCreateDeviceRequest;

export function CreateDeviceForm() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "" },
  });

  const [createdResult, setCreatedResult] =
    useState<CreateDeviceResponse | null>(null);

  const mutation = useCreateDevice({
    onSuccess: (data) => {
      setCreatedResult(data);
      form.reset();
    },
  });

  async function handleCopyApiKey() {
    if (!createdResult) return;

    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      toast.error("Copy to clipboard is not supported in this browser.");
      return;
    }

    try {
      await navigator.clipboard.writeText(createdResult.api_key);
      toast.success("Copied to clipboard");
    } catch {
      toast.error("Failed to copy API key");
    }
  }

  function onSubmit(values: z.infer<typeof formSchema>) {
    mutation.mutate({ body: values });
  }

  return (
    <div className="space-y-4">
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex items-end gap-4"
        >
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormLabel>New Device Name</FormLabel>
                <FormControl>
                  <Input placeholder="e.g. Office Printer" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Creating..." : "Add Device"}
          </Button>
        </form>
      </Form>
      <Dialog
        open={createdResult !== null}
        onOpenChange={(open) => {
          if (!open) {
            setCreatedResult(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Device created — save your API key</DialogTitle>
            <DialogDescription>
              This API key is shown only once. Copy it now and store it
              securely.
            </DialogDescription>
          </DialogHeader>
          {createdResult && (
            <div className="space-y-4">
              <div className="text-sm">
                <span className="font-medium">Device:</span>{" "}
                <span>{createdResult.device.name}</span>
              </div>
              <div className="space-y-2">
                <p className="text-sm font-medium">API key</p>
                <div className="flex gap-2">
                  <Input
                    readOnly
                    value={createdResult.api_key}
                    className="font-mono"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCopyApiKey}
                  >
                    Copy
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground">
                You will not be able to see this full API key again. Make sure
                you have stored it securely.
              </p>
            </div>
          )}
          <DialogFooter>
            <Button type="button" onClick={() => setCreatedResult(null)}>
              I&apos;ve saved it
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
