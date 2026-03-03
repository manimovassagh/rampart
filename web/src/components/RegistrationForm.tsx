import { useState } from "react";
import type { FormEvent } from "react";
import { registerUser } from "../api/register";
import type { FieldError, UserResponse } from "../types";

interface Props {
  onNavigateLogin: () => void;
}

export default function RegistrationForm({ onNavigateLogin }: Props) {
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
      <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-lg">
        <div className="mb-4 flex justify-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-7 w-7 text-emerald-600"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </div>
        </div>
        <h2 className="mb-2 text-center text-xl font-semibold text-slate-900">
          Account Created
        </h2>
        <p className="text-center text-sm text-slate-500">
          Welcome, <span className="font-semibold text-slate-700">{success.username}</span>.
          Your account is ready.
        </p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-lg">
      {/* Card header */}
      <div className="border-b border-slate-100 px-8 pt-8 pb-6">
        <h1 className="text-center text-xl font-semibold text-slate-900">
          Create your account
        </h1>
        <p className="mt-1 text-center text-sm text-slate-500">
          Register to get started with Rampart
        </p>
      </div>

      {/* Card body */}
      <form onSubmit={handleSubmit} className="space-y-5 px-8 pt-6 pb-8">
        {generalError && (
          <div className="flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <svg
              viewBox="0 0 20 20"
              fill="currentColor"
              className="mt-0.5 h-4 w-4 shrink-0 text-red-500"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z"
                clipRule="evenodd"
              />
            </svg>
            <span className="text-sm text-red-700">{generalError}</span>
          </div>
        )}

        {/* Name row */}
        <div className="grid grid-cols-2 gap-4">
          <Field
            label="First name"
            value={givenName}
            onChange={setGivenName}
            autoComplete="given-name"
            placeholder="John"
          />
          <Field
            label="Last name"
            value={familyName}
            onChange={setFamilyName}
            autoComplete="family-name"
            placeholder="Doe"
          />
        </div>

        <Field
          label="Username"
          value={username}
          onChange={setUsername}
          error={getFieldError("username")}
          autoComplete="username"
          placeholder="johndoe"
        />
        <Field
          label="Email address"
          type="email"
          value={email}
          onChange={setEmail}
          error={getFieldError("email")}
          autoComplete="email"
          placeholder="john@example.com"
        />
        <Field
          label="Password"
          type="password"
          value={password}
          onChange={setPassword}
          error={getFieldError("password")}
          autoComplete="new-password"
          placeholder="Min. 8 characters"
        />

        <button
          type="submit"
          disabled={loading}
          className="mt-2 flex w-full items-center justify-center gap-2 rounded-lg bg-slate-900 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-slate-800 focus:ring-2 focus:ring-slate-900 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading && (
            <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="3"
                className="opacity-25"
              />
              <path
                d="M4 12a8 8 0 018-8"
                stroke="currentColor"
                strokeWidth="3"
                strokeLinecap="round"
                className="opacity-75"
              />
            </svg>
          )}
          {loading ? "Creating account..." : "Register"}
        </button>

        <p className="text-center text-xs text-slate-400">
          Already have an account?{" "}
          <button
            type="button"
            onClick={onNavigateLogin}
            className="font-medium text-slate-600 hover:text-slate-900"
          >
            Sign in
          </button>
        </p>
      </form>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  error,
  type = "text",
  autoComplete,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  error?: string;
  type?: string;
  autoComplete?: string;
  placeholder?: string;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-slate-700">
        {label}
      </label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoComplete={autoComplete}
        placeholder={placeholder}
        className={`block w-full rounded-lg border bg-white px-3.5 py-2.5 text-sm text-slate-900 shadow-sm transition-colors placeholder:text-slate-400 focus:ring-2 focus:outline-none ${
          error
            ? "border-red-300 focus:border-red-400 focus:ring-red-100"
            : "border-slate-300 focus:border-slate-400 focus:ring-slate-100"
        }`}
      />
      {error && (
        <p className="mt-1.5 text-xs text-red-600">{error}</p>
      )}
    </div>
  );
}
