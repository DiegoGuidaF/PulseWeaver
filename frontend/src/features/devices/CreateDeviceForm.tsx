import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { useCreateDevice } from "@/features/devices/hooks/useCreateDevice";
import { zCreateDeviceRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const formSchema = zCreateDeviceRequest;

export function CreateDeviceForm() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "" },
  });

  const mutation = useCreateDevice({
    onSuccess: () => form.reset(),
  });

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
    </div>
  );
}
