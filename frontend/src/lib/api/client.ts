import createClient from "openapi-fetch";
import type { paths, components } from "./schema";

export type ErrorResponse = components["schemas"]["ErrorResponse"];

export const api = createClient<paths>({
  baseUrl: "/api/v1",
});

// Helper to throw consistent errors
export function toErrorMessage(err: unknown): string {
  if (typeof err === "object" && err && "error" in err)
    return String((err as any).error);
  if (err instanceof Error) return err.message;
  return "Unknown error";
}
