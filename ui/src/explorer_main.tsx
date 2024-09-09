import React from "react";
import ReactDOM from "react-dom/client";

import { Explorer } from "./explorer/Explorer.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Explorer />
  </React.StrictMode>,
);
