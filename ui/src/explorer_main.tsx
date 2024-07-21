import ReactDOM from "react-dom/client";
import React from "react";

import { Explorer } from "./explorer/Explorer.tsx";

import "./base.scss";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Explorer />
  </React.StrictMode>,
);
