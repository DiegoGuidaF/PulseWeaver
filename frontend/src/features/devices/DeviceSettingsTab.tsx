import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toErrorMessage } from "@/lib/api-client";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { usePutDeviceAddressLeaseRule } from "@/features/devices/hooks/usePutDeviceAddressLeaseRule";
import { useDisableDeviceAddressLeaseRule } from "@/features/devices/hooks/useDisableDeviceAddressLeaseRule";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";
import { toast } from "sonner";

const TTL_UNITS = ["seconds", "minutes", "days"] as const;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_DAY = 86400;

type TtlUnit = (typeof TTL_UNITS)[number];

function toSeconds(value: number, unit: TtlUnit): number {
  switch (unit) {
    case "seconds":
      return value;
    case "minutes":
      return value * SECONDS_PER_MINUTE;
    case "days":
      return value * SECONDS_PER_DAY;
    default: {
      const _exhaustive: never = unit;
      return _exhaustive;
    }
  }
}

function fromSeconds(ttlSeconds: number): { value: number; unit: TtlUnit } {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    return { value: ttlSeconds / SECONDS_PER_DAY, unit: "days" };
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    return { value: ttlSeconds / SECONDS_PER_MINUTE, unit: "minutes" };
  }
  return { value: ttlSeconds, unit: "seconds" };
}

function formatTtlLabel(ttlSeconds: number): string {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    const days = ttlSeconds / SECONDS_PER_DAY;
    return days === 1 ? "1 day" : `${days} days`;
  }

  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    const minutes = ttlSeconds / SECONDS_PER_MINUTE;
    if (minutes % 60 === 0) {
      const hours = minutes / 60;
      return hours === 1 ? "1 hour" : `${hours} hours`;
    }
    return minutes === 1 ? "1 minute" : `${minutes} minutes`;
  }

  return ttlSeconds === 1 ? "1 second" : `${ttlSeconds} seconds`;
}

const leaseRuleFormSchema = z.object({
  value: z
    .coerce.number()
    .int("Must be a whole number")
    .min(1, "Minimum is 1"),
  unit: z.enum(TTL_UNITS),
});

type LeaseRuleFormValues = z.infer<typeof leaseRuleFormSchema>;
type LeaseRuleFormInput = z.input<typeof leaseRuleFormSchema>;

interface DeviceSettingsTabProps {
  deviceId: number;
  device?: { name: string; api_key_prefix: string };
}

export function DeviceSettingsTab({ deviceId, device }: DeviceSettingsTabProps) {
  const {
    data: rule,
    isLoading,
    isError,
    error,
  } = useDeviceAddressLeaseRule(deviceId);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);
  const regenerateApiKey = useRegenerateApiKey();

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(null);

  const leaseRuleForm = useForm<LeaseRuleFormInput, unknown, LeaseRuleFormValues>({
    resolver: zodResolver(leaseRuleFormSchema),
    defaultValues: { value: 5, unit: "minutes" },
  });
  const { reset } = leaseRuleForm;
  const [editing, setEditing] = useState(false);

  const isOn = Boolean(rule && rule.enabled);

  async function handleCopyRegeneratedKey() {
    if (!regeneratedApiKey) return;

    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      toast.error("Copy to clipboard is not supported in this browser.");
      return;
    }

    try {
      await navigator.clipboard.writeText(regeneratedApiKey);
      toast.success("Copied to clipboard");
    } catch {
      toast.error("Failed to copy API key");
    }
  }

  function handleConfirmRegenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setRegeneratedApiKey(data.api_key);
        },
      },
    );
  }

  function handleLeaseRuleSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate({
      path: { device_id: deviceId },
      body: { ttl_seconds: toSeconds(values.value, values.unit) },
    });
    setEditing(false);
  }

  function handleStartEditing() {
    if (!rule) return;
    const initial = fromSeconds(rule.ttl_seconds);
    reset(initial);
    setEditing(true);
  }

  useEffect(() => {
    if (!rule || isOn) {
      return;
    }
    const initial = fromSeconds(rule.ttl_seconds);
    reset(initial);
  }, [isOn, reset, rule]);

  const ttlLabel =
    rule && rule.ttl_seconds ? formatTtlLabel(rule.ttl_seconds) : null;
  const submitButtonLabel = putRuleMutation.isPending
    ? "Saving..."
    : isOn
      ? "Save"
      : "Enable auto-expiry";

  return (
    <div className="space-y-6">
      {/* Settings section */}
      <div className="space-y-3">
        <h3 className="text-sm font-medium">Settings</h3>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between gap-4">
              <div className="space-y-1">
                <p className="text-sm font-medium">API Key</p>
                {device ? (
                  <p className="font-mono text-sm text-muted-foreground">
                    {device.api_key_prefix}&hellip;
                  </p>
                ) : (
                  <Skeleton className="h-4 w-32" />
                )}
              </div>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!device || regenerateApiKey.isPending}
                  >
                    Regenerate API key
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      Regenerate API key for &ldquo;{device?.name}&rdquo;?
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      The current key (
                      <span className="font-mono">{device?.api_key_prefix}&hellip;</span>
                      ) will stop working immediately. You will need to update
                      any scripts or services using this device.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={handleConfirmRegenerate}>
                      Regenerate
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Rules section */}
      <div className="space-y-3">
        <h3 className="text-sm font-medium">Rules</h3>
        <Card>
          <CardHeader>
            <CardTitle>Auto-expiry rule</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-4 w-64" />
              </div>
            ) : isError ? (
              <p className="text-sm text-red-500">
                Error loading rule: {toErrorMessage(error)}
              </p>
            ) : (
              <>
                {isOn && (
                  <div className="space-y-1">
                    <p className="text-sm">
                      Status:{" "}
                      <span className="font-medium">Enabled</span>
                    </p>
                    {ttlLabel && (
                      <p className="text-sm text-muted-foreground">
                        Addresses will automatically expire after{" "}
                        <span className="font-medium">{ttlLabel}</span>.
                      </p>
                    )}
                  </div>
                )}

                {!isOn && (
                  <p className="text-sm text-muted-foreground">
                    Auto-expiry is currently{" "}
                    <span className="font-medium text-foreground">disabled</span>
                    . Turn it on to automatically revoke stale addresses.
                  </p>
                )}

                {(!isOn || editing) && (
                  <Form {...leaseRuleForm}>
                    <form
                      onSubmit={leaseRuleForm.handleSubmit(handleLeaseRuleSubmit)}
                      className="flex flex-wrap items-end gap-4"
                    >
                      <FormField
                        control={leaseRuleForm.control}
                        name="value"
                        render={({ field }) => (
                          <FormItem className="w-32">
                            <FormLabel>Expires after</FormLabel>
                            <FormControl>
                              <Input
                                type="number"
                                min={1}
                                step={1}
                                placeholder="1"
                                name={field.name}
                                ref={field.ref}
                                onBlur={field.onBlur}
                                value={typeof field.value === "number" ? field.value : ""}
                                onChange={(e) =>
                                  field.onChange(
                                    e.target.value === ""
                                      ? undefined
                                      : Number(e.target.value),
                                  )
                                }
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={leaseRuleForm.control}
                        name="unit"
                        render={({ field }) => (
                          <FormItem className="w-32">
                            <FormLabel>Unit</FormLabel>
                            <FormControl>
                              <select
                                className="border-input focus-visible:ring-ring flex h-9 w-full rounded-md border bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1"
                                {...field}
                              >
                                {TTL_UNITS.map((unit) => (
                                  <option key={unit} value={unit}>
                                    {unit}
                                  </option>
                                ))}
                              </select>
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button
                        type="submit"
                        disabled={putRuleMutation.isPending}
                      >
                        {submitButtonLabel}
                      </Button>
                      {editing && (
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => setEditing(false)}
                        >
                          Cancel
                        </Button>
                      )}
                    </form>
                  </Form>
                )}

                {isOn && !editing && (
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={handleStartEditing}
                    >
                      Change TTL
                    </Button>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() =>
                        disableRuleMutation.mutate({
                          path: { device_id: deviceId },
                        })
                      }
                      disabled={disableRuleMutation.isPending}
                    >
                      Turn off auto-expiry
                    </Button>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* One-time key display dialog after successful regeneration */}
      <Dialog
        open={regeneratedApiKey !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRegeneratedApiKey(null);
          }
        }}
      >
        <DialogContent
          showCloseButton={false}
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>API key regenerated — save your new key</DialogTitle>
            <DialogDescription>
              This API key is shown only once. Copy it now and store it
              securely. The old key is no longer valid.
            </DialogDescription>
          </DialogHeader>
          {regeneratedApiKey && (
            <div className="space-y-4">
              <div className="space-y-2">
                <p className="text-sm font-medium">New API key</p>
                <div className="flex gap-2">
                  <Input
                    readOnly
                    value={regeneratedApiKey}
                    className="font-mono"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCopyRegeneratedKey}
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
            <Button type="button" onClick={() => setRegeneratedApiKey(null)}>
              I&apos;ve saved it
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
