import ReactDOM from "react-dom/client";
import React from "react";

import "./tool/wasm_exec.js";
import { Explorer } from "./explorer/Explorer.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Explorer />
  </React.StrictMode>,
);
