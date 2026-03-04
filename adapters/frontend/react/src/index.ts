export { RampartProvider, RampartContext } from "./context.js";
export type { RampartProviderProps, RampartContextValue } from "./context.js";

export { useAuth } from "./use-auth.js";
export type { UseAuthReturn } from "./use-auth.js";

export { ProtectedRoute } from "./protected-route.js";
export type { ProtectedRouteProps } from "./protected-route.js";

// Re-export @rampart/web types so consumers only need @rampart/react
export { RampartClient } from "@rampart/web";
export type {
  RampartClientConfig,
  RampartTokens,
  RampartUser,
  RampartError,
} from "@rampart/web";
