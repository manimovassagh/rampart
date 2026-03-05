// Types
export type {
  RampartClaims,
  RampartMiddlewareConfig,
  ServerAuth,
} from "./types.js";

// Middleware (Edge Runtime)
export { withRampartAuth } from "./middleware.js";

// Server utilities
export { getServerAuth, validateToken } from "./server.js";

// Client hooks and components
export {
  useRampartSession,
  RampartProvider,
  RampartContext,
  useAuth,
  ProtectedRoute,
} from "./client.js";

export type {
  RampartProviderProps,
  RampartContextValue,
  UseAuthReturn,
  ProtectedRouteProps,
} from "./client.js";
