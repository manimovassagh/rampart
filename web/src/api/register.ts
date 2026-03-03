import type {
  RegistrationRequest,
  UserResponse,
  ValidationErrorResponse,
  ApiErrorResponse,
} from "../types";

export type RegisterResult =
  | { ok: true; user: UserResponse }
  | { ok: false; validation?: ValidationErrorResponse; error?: ApiErrorResponse };

export async function registerUser(data: RegistrationRequest): Promise<RegisterResult> {
  const res = await fetch("/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  let body: unknown;
  try {
    body = await res.json();
  } catch {
    return {
      ok: false,
      error: {
        error: "network_error",
        error_description: "Unable to reach the server. Please try again.",
        status: res.status,
      },
    };
  }

  if (res.ok) {
    return { ok: true, user: body as UserResponse };
  }

  const err = body as Record<string, unknown>;
  if (err.fields) {
    return { ok: false, validation: err as unknown as ValidationErrorResponse };
  }

  return { ok: false, error: err as unknown as ApiErrorResponse };
}
