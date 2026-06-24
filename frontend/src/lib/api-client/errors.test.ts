import { describe, expect, it } from "vitest";
import { ApiError, toApiError, toErrorMessage } from "./errors";

describe("api-client errors", () => {
    it("preserves ApiError message, name, and status", () => {
        const err = new ApiError("not allowed", 403);

        expect(err).toBeInstanceOf(Error);
        expect(err.message).toBe("not allowed");
        expect(err.name).toBe("ApiError");
        expect(err.status).toBe(403);
    });

    it("extracts display messages from known error shapes", () => {
        expect(toErrorMessage(new ApiError("conflict", 409))).toBe("conflict");
        expect(toErrorMessage(new Error("native failure"))).toBe("native failure");
        expect(toErrorMessage({ error: "invalid input" })).toBe("invalid input");
    });

    it("falls back for unknown error shapes", () => {
        expect(toErrorMessage("plain string")).toBe("Unknown error");
        expect(toErrorMessage(null)).toBe("Unknown error");
    });

    it("extracts status from supported SDK error fields", () => {
        expect(toApiError({ error: "bad request", status: 400 })).toMatchObject({
            message: "bad request",
            status: 400,
        });
        expect(toApiError({ error: "rate limited", statusCode: 429 })).toMatchObject({
            message: "rate limited",
            status: 429,
        });
        expect(toApiError({ error: "missing", response: { status: 404 } })).toMatchObject({
            message: "missing",
            status: 404,
        });
    });
});
