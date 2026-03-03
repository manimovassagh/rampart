import { useState, useEffect, useCallback } from "react";

export interface ToastMessage {
  id: number;
  type: "success" | "error";
  text: string;
}

let nextId = 0;
let addToastFn: ((type: "success" | "error", text: string) => void) | null =
  null;

export function toast(type: "success" | "error", text: string) {
  addToastFn?.(type, text);
}

export default function ToastContainer() {
  const [messages, setMessages] = useState<ToastMessage[]>([]);

  const addToast = useCallback((type: "success" | "error", text: string) => {
    const id = nextId++;
    setMessages((prev) => [...prev, { id, type, text }]);
    setTimeout(() => {
      setMessages((prev) => prev.filter((m) => m.id !== id));
    }, 4000);
  }, []);

  useEffect(() => {
    addToastFn = addToast;
    return () => {
      addToastFn = null;
    };
  }, [addToast]);

  if (messages.length === 0) return null;

  return (
    <div className="fixed right-4 top-4 z-50 flex flex-col gap-2">
      {messages.map((m) => (
        <div
          key={m.id}
          className={`rounded-lg px-4 py-3 text-sm font-medium text-white shadow-lg transition-all ${
            m.type === "success" ? "bg-emerald-600" : "bg-red-600"
          }`}
        >
          {m.text}
        </div>
      ))}
    </div>
  );
}
