import React from 'react'
import ReactDOM from 'react-dom/client'

import TreeMap from './TreeMap.tsx'
import {loadDataFromEmbed} from "./tool/utils.ts";
import {createEntry} from "./tool/entry.ts";

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <TreeMap entry={createEntry(loadDataFromEmbed())}/>
    </React.StrictMode>,
)
