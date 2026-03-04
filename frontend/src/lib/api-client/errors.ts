// Custom error class that preserves HTTP status codes
export class ApiError extends Error {
  status?: number;
  constructor(message: string, status?: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

// Helper to extract error message from various error types
export function toErrorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  if (err instanceof Error) return err.message;
  if (typeof err === 'object' && err && 'error' in err) {
    return String((err as { error: unknown }).error);
  }
  return 'Unknown error';
}

// Helper to create an ApiError from SDK errors
export function toApiError(err: unknown): ApiError {
  const message = toErrorMessage(err);
  
  // Try to extract status code from error object
  let status: number | undefined;
  if (err && typeof err === 'object') {
    if ('status' in err) {
      status = Number((err as { status: unknown }).status);
    } else if ('statusCode' in err) {
      status = Number((err as { statusCode: unknown }).statusCode);
    } else if ('response' in err) {
      const response = (err as { response: unknown }).response;
      if (response && typeof response === 'object' && 'status' in response) {
        status = Number((response as { status: unknown }).status);
      }
    }
  }
  
  return new ApiError(message, status);
}
