import React from "react";
import ReactDOM from "react-dom/client";

import { createEntry } from "./tool/entry.ts";
import { loadDataFromEmbed } from "./tool/utils.ts";
import TreeMap from "./TreeMap.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <TreeMap entry={createEntry(loadDataFromEmbed())} />
  </React.StrictMode>,
);
