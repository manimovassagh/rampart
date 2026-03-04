import { jsx as _jsx } from "react/jsx-runtime";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { RampartProvider } from "@rampart/react";
import { App } from "./App";
import "./app.css";
createRoot(document.getElementById("root")).render(_jsx(StrictMode, { children: _jsx(RampartProvider, { issuer: "http://localhost:8080", clientId: "sample-react-app", redirectUri: "http://localhost:3002/callback", children: _jsx(BrowserRouter, { children: _jsx(App, {}) }) }) }));
