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

  if (res.ok) {
    const user: UserResponse = await res.json();
    return { ok: true, user };
  }

  const body = await res.json();

  if (body.fields) {
    return { ok: false, validation: body as ValidationErrorResponse };
  }

  return { ok: false, error: body as ApiErrorResponse };
}
