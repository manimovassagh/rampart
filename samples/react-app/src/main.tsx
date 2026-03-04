import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { RampartProvider } from "@rampart/react";
import { App } from "./App";
import "./app.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RampartProvider
      issuer="http://localhost:8080"
      clientId="sample-react-app"
      redirectUri="http://localhost:3002/callback"
    >
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </RampartProvider>
  </StrictMode>
);
