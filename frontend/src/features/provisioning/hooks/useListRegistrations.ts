import { useQuery } from "@tanstack/react-query";
import { listRegistrationsOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { ListRegistrationsData } from "@/lib/api";

type RegistrationQueryStatus = NonNullable<
  ListRegistrationsData["query"]
>["status"];

export function useListRegistrations(
  status: RegistrationQueryStatus = "pending",
) {
  return useQuery(listRegistrationsOptions({ query: { status } }));
}
