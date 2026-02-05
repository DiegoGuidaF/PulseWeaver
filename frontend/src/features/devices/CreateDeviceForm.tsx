import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { useState } from "react";

const formSchema = z.object({
    name: z.string().min(3, "Name must be at least 3 characters").max(50),
});

export function CreateDeviceForm() {
    const queryClient = useQueryClient();
    const [error, setError] = useState<string | null>(null);

    const form = useForm<z.infer<typeof formSchema>>({
        resolver: zodResolver(formSchema),
        defaultValues: { name: "" },
    });

    const mutation = useMutation({
        mutationFn: async (values: z.infer<typeof formSchema>) => {
            const { data, error } = await api.POST("/devices", {
                body: values,
            });
            if (error) throw new Error(toErrorMessage(error));
            return data;
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["devices"] });
            form.reset();
            setError(null);
        },
        onError: (err) => setError(err.message),
    });

    function onSubmit(values: z.infer<typeof formSchema>) {
        mutation.mutate(values);
    }

    return (
        <div className="space-y-4">
            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-4 items-end">
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
            {error && <p className="text-sm text-red-500">{error}</p>}
        </div>
    );
}
