import { afterEach, beforeAll } from "vitest";
import { cleanup } from "@testing-library/react";

// Clean up DOM between tests (React 19 + testing-library quirk)
afterEach(() => {
  cleanup();
});

// Node 25+ ships a bare `localStorage` global that conflicts with jsdom's
// Web Storage implementation. Replace it with a spec-compliant in-memory mock.
const store = new Map<string, string>();

const storage: Storage = {
  get length() {
    return store.size;
  },
  clear() {
    store.clear();
  },
  getItem(key: string) {
    return store.get(key) ?? null;
  },
  key(index: number) {
    return [...store.keys()][index] ?? null;
  },
  removeItem(key: string) {
    store.delete(key);
  },
  setItem(key: string, value: string) {
    store.set(key, value);
  },
};

beforeAll(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: storage,
    writable: true,
    configurable: true,
  });

  Object.defineProperty(window, "localStorage", {
    value: storage,
    writable: true,
    configurable: true,
  });
});
