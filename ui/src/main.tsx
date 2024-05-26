import React from 'react'
import ReactDOM from 'react-dom/client'
import TreeMap from './TreeMap.tsx'

import {loadDataFromEmbed} from "./tool/utils.ts";
import {Entry} from "./tool/entry.ts";



ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <TreeMap entry={new Entry(loadDataFromEmbed())}/>
    </React.StrictMode>,
)
