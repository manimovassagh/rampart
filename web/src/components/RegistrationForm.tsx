import { useState } from "react";
import type { FormEvent } from "react";
import { registerUser } from "../api/register";
import type { FieldError, UserResponse } from "../types";

export default function RegistrationForm() {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [givenName, setGivenName] = useState("");
  const [familyName, setFamilyName] = useState("");

  const [fieldErrors, setFieldErrors] = useState<FieldError[]>([]);
  const [generalError, setGeneralError] = useState("");
  const [success, setSuccess] = useState<UserResponse | null>(null);
  const [loading, setLoading] = useState(false);

  function getFieldError(field: string): string | undefined {
    return fieldErrors.find((e) => e.field === field)?.message;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFieldErrors([]);
    setGeneralError("");
    setSuccess(null);
    setLoading(true);

    const result = await registerUser({
      username,
      email,
      password,
      given_name: givenName,
      family_name: familyName,
    });

    setLoading(false);

    if (result.ok) {
      setSuccess(result.user);
      return;
    }

    if (result.validation) {
      setFieldErrors(result.validation.fields);
    } else if (result.error) {
      setGeneralError(result.error.error_description);
    }
  }

  if (success) {
    return (
      <div className="mx-auto max-w-md rounded-lg bg-green-50 p-6 text-center">
        <h2 className="mb-2 text-xl font-semibold text-green-800">
          Registration Successful
        </h2>
        <p className="text-green-700">
          Welcome, <span className="font-medium">{success.username}</span>! Your
          account has been created.
        </p>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="mx-auto max-w-md space-y-4 rounded-lg bg-white p-6 shadow-md"
    >
      <h1 className="text-center text-2xl font-bold text-gray-900">
        Create Account
      </h1>

      {generalError && (
        <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">
          {generalError}
        </div>
      )}

      <Field
        label="Username"
        value={username}
        onChange={setUsername}
        error={getFieldError("username")}
        autoComplete="username"
      />
      <Field
        label="Email"
        type="email"
        value={email}
        onChange={setEmail}
        error={getFieldError("email")}
        autoComplete="email"
      />
      <Field
        label="Password"
        type="password"
        value={password}
        onChange={setPassword}
        error={getFieldError("password")}
        autoComplete="new-password"
      />
      <Field
        label="First Name"
        value={givenName}
        onChange={setGivenName}
        autoComplete="given-name"
      />
      <Field
        label="Last Name"
        value={familyName}
        onChange={setFamilyName}
        autoComplete="family-name"
      />

      <button
        type="submit"
        disabled={loading}
        className="w-full rounded-md bg-indigo-600 px-4 py-2 font-medium text-white hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:opacity-50"
      >
        {loading ? "Creating account..." : "Register"}
      </button>
    </form>
  );
}

function Field({
  label,
  value,
  onChange,
  error,
  type = "text",
  autoComplete,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  error?: string;
  type?: string;
  autoComplete?: string;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoComplete={autoComplete}
        className={`mt-1 block w-full rounded-md border px-3 py-2 shadow-sm focus:ring-2 focus:outline-none ${
          error
            ? "border-red-300 focus:ring-red-500"
            : "border-gray-300 focus:ring-indigo-500"
        }`}
      />
      {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
    </div>
  );
}
