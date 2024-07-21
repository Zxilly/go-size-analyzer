import React from "react";
import ReactDOM from "react-dom/client";

import TreeMap from "./treemap/TreeMap.tsx";
import { loadDataFromEmbed } from "./tool/utils.ts";
import { createEntry } from "./tool/entry.ts";

import "./base.scss";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <TreeMap entry={createEntry(loadDataFromEmbed())} />
  </React.StrictMode>,
);
