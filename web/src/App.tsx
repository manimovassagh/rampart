import RegistrationForm from "./components/RegistrationForm";

export default function App() {
  return (
    <div className="flex min-h-screen flex-col bg-gradient-to-br from-slate-50 to-slate-100">
      {/* Header bar */}
      <header className="border-b border-slate-200 bg-white px-6 py-4 shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-slate-900">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="h-5 w-5 text-white"
            >
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
          </div>
          <span className="text-xl font-bold tracking-tight text-slate-900">
            Rampart
          </span>
        </div>
      </header>

      {/* Main content */}
      <main className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <RegistrationForm />
          <p className="mt-6 text-center text-xs text-slate-400">
            Powered by Rampart Identity Server
          </p>
        </div>
      </main>
    </div>
  );
}
