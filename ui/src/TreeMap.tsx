import {loadData} from "./utils.ts";
import {useEffect, useMemo} from "react";
import {Entry} from "./entry.ts";
import {useWindowSize} from "usehooks-ts";
import {hierarchy, treemap} from "d3-hierarchy";

function TreeMap() {
    const entry = useMemo(() => new Entry(loadData()), [])

    useEffect(() => {
        document.title = entry.getName()
    }, [entry])

    const {width, height} = useWindowSize()

    const rawHierarchy = useMemo(() => {
        return hierarchy(entry, (e) => e.getChildren())
    }, [entry])

    const layout = useMemo(() => {
        return treemap()
    })

    return (
        <>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox={`0 0 ${width} ${height}`}>
            </svg>
        </>
    )
}

export default TreeMap
