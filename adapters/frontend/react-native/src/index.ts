export { RampartProvider, RampartNativeContext } from "./RampartProvider.js";
export { useAuth } from "./useAuth.js";

export type {
  RampartNativeProviderProps,
  RampartNativeContextValue,
  UseAuthReturn,
} from "./types.js";

// Re-export @rampart-auth/web types so consumers only need @rampart-auth/react-native
export { RampartClient } from "@rampart-auth/web";
export type {
  RampartClientConfig,
  RampartTokens,
  RampartUser,
  RampartError,
} from "@rampart-auth/web";
