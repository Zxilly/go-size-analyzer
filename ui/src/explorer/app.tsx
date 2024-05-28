import React, {ReactNode, useEffect, useMemo} from "react";
import {useAsync} from "react-use";
import gsa from "../../gsa.wasm?init";
import {Entry} from "../tool/entry.ts";
import {loadDataFromWasmResult} from "../tool/utils.ts";
import {Dialog, DialogContent, DialogContentText, DialogTitle} from "@mui/material";
import {FileSelector} from "./file_selector.tsx";
import TreeMap from "../TreeMap.tsx";

type ModalState = {
    isOpen: false
} | {
    isOpen: true
    title: string
    content: ReactNode
}

export const App: React.FC = () => {
    const go = useMemo(() => new Go(), [])

    const {value: inst, loading, error: loadError} = useAsync(async () => {
        return await gsa(go.importObject)
    })

    useAsync(async () => {
        if (loading || loadError || inst === undefined) {
            return
        }

        return await go.run(inst)
    }, [inst])

    const [file, setFile] = React.useState<File | null>(null)

    const [modalState, setModalState] = React.useState<ModalState>({isOpen: false})

    const {value: jsonResult, loading: analyzing} = useAsync(async () => {
        if (!file) {
            return
        }

        const bytes = await file.arrayBuffer()
        const uint8 = new Uint8Array(bytes)

        return gsa_analyze(file.name, uint8)
    }, [file])

    const entry = useMemo(() => {
        if (!jsonResult) {
            return null
        }

        return new Entry(loadDataFromWasmResult(jsonResult))
    }, [jsonResult])

    useEffect(() => {
        if (loadError) {
            setModalState({
                isOpen: true, title: "Error", content:
                    <DialogContentText>{loadError.message}</DialogContentText>
            })
        } else if (loading) {
            setModalState({
                isOpen: true, title: "Loading", content:
                    <DialogContentText>Loading WebAssembly module...</DialogContentText>
            })
        } else if (inst === undefined) {
            setModalState({
                isOpen: true,
                title: "Error",
                content: <DialogContentText>Failed to load WebAssembly module</DialogContentText>
            })
        } else if (file === null) {
            setModalState({
                isOpen: true,
                title: "Select a go binary",
                content: <FileSelector handler={(file) => {
                    setFile(file)
                }}/>
            })
        } else if (analyzing) {
            setModalState({
                isOpen: true,
                title: "Analyzing",
                content: <DialogContentText>Analyzing binary...</DialogContentText>
            })
        } else if (!analyzing && !jsonResult && !entry) {
            setModalState({
                isOpen: true,
                title: "Error",
                content: <DialogContentText>
                    Failed to analyze {file.name}, see browser dev console for more details.
                </DialogContentText>
            })
        } else {
            setModalState({isOpen: false})
        }
    }, [loadError, loading, file, jsonResult, analyzing, inst, entry])

    return <>
        <Dialog
            open={modalState.isOpen}
        >
            <DialogTitle>{modalState.isOpen && modalState.title}</DialogTitle>
            <DialogContent>
                {modalState.isOpen && modalState.content}
            </DialogContent>
        </Dialog>
        {entry && <TreeMap entry={entry}/>}
    </>
}