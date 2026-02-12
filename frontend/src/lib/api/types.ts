import type { components } from "./schema";

// Request types
export type AuthRequest = components["schemas"]["AuthRequest"];
export type CreateDeviceRequest = components["schemas"]["CreateDeviceRequest"];
export type AddAddressRequest = components["schemas"]["AddAddressRequest"];
export type CreateUserRequest = components["schemas"]["CreateUserRequest"];

// Response types
export type User = components["schemas"]["User"];
export type Device = components["schemas"]["Device"];
export type Address = components["schemas"]["Address"];
export type ErrorResponse = components["schemas"]["ErrorResponse"];
