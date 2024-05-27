import ReactDOM from "react-dom/client";
import React from "react";

import "./tool/wasm_exec.js"
import {App} from "./explorer/app.tsx";

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <App/>
    </React.StrictMode>,
)
