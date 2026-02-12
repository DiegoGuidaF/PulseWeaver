import createClient from "openapi-fetch";
import type { paths, components } from "./schema";

export type ErrorResponse = components["schemas"]["ErrorResponse"];

export const api = createClient<paths>({
  baseUrl: "/api/v1",
  credentials: "include", // Ensure cookies are sent with all requests
});

// Custom error class that preserves HTTP status codes
export class ApiError extends Error {
  status?: number;
  constructor(message: string, status?: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// Helper to throw consistent errors with status preservation
export function toErrorMessage(err: unknown): string {
  if (typeof err === "object" && err && "error" in err)
    return String((err as any).error);
  if (err instanceof Error) return err.message;
  return "Unknown error";
}

// Helper to create an ApiError from an openapi-fetch error
export function toApiError(err: unknown): ApiError {
  const message = toErrorMessage(err);
  const status = (err as any)?.status;
  return new ApiError(message, status);
}
