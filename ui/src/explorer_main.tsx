import {useAsync} from "react-use";
import ReactDOM from "react-dom/client";
import React, {ReactNode, useCallback, useEffect, useMemo, useState} from "react";
import {
    Box,
    Button,
    CssBaseline,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Link
} from "@mui/material";

import "./tool/wasm_exec.js"

import gsa from "../gsa.wasm?init"

import {formatBytes, loadDataFromWasmResult} from "./tool/utils.ts";
import {Entry} from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";


type ModalState = {
    isOpen: false
} | {
    isOpen: true
    title: string
    content: ReactNode
}


type fileChangeHandler = (file: File) => void

const SizeLimit = 1024 * 1024 * 30

const FileSelector = ({handler}: {
    value?: File | null,
    handler: fileChangeHandler
}) => {
    const [open, setOpen] = useState(false)
    const [pendingFile, setPendingFile] = useState<File | null>(null)

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        if (!e.target.files || e.target.files.length === 0) {
            return
        }

        const f = e.target.files[0]

        if (f.size > SizeLimit) {
            setOpen(true)
            setPendingFile(f)
            return
        } else {
            handler(f)
        }

    }, [])

    const handleClose = useCallback(() => {
        setOpen(false)
    }, [])

    return (
        <>
            <Dialog
                open={open}
            >
                <DialogTitle>
                    Binary too large
                </DialogTitle>
                <DialogContent>
                    <DialogContentText>
                        The selected binary {pendingFile?.name} has a size of {formatBytes(pendingFile?.size || 0)}.
                        It is not recommended to use the wasm version for binary files larger than 30MB.
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={handleClose}>Cancel</Button>
                    <Button onClick={() => {
                        handler(pendingFile!)
                        setOpen(false)
                    }}>Continue</Button>
                </DialogActions>
            </Dialog>
            <Box
                display="flex"
                justifyContent="center"
                alignItems="center"
                height="100%"
            >
                <Button
                    variant="outlined"
                    component="label"
                >
                    Select file
                    <input
                        type="file"
                        multiple={false}
                        onChange={handleChange}
                        hidden
                    />
                </Button>
            </Box>
            <DialogContentText marginTop={2} style={{
                verticalAlign: "middle",
            }}>
                For full features, see
                <Link
                    href="https://github.com/Zxilly/go-size-analyzer"
                    target="_blank"
                    style={{
                        marginLeft: "0.3em",
                    }}
                >go-size-analyzer
                    <img alt="GitHub Repo stars"
                         style={{
                             marginLeft: "0.3em",
                         }}
                         src="https://img.shields.io/github/stars/Zxilly/go-size-analyzer"/>
                </Link>
            </DialogContentText>
        </>
    );
};

const App: React.FC = () => {
    const go = useMemo(() => new Go(), [])

    const {value: inst, loading, error: loadError} = useAsync(async () => {
        return await gsa(go.importObject)
    })

    useAsync(async () => {
        if (loading || loadError || inst === undefined) {
            return
        }

        return await go.run(inst!)
    }, [inst])

    const [file, setFile] = React.useState<File | null>(null)

    const [modalState, setModalState] = React.useState<ModalState>({isOpen: false})

    const {value: jsonResult, loading: analyzing} = useAsync(async () => {
        if (!file) {
            return
        }

        const bytes = await file.arrayBuffer()
        const uint8 = new Uint8Array(bytes)
        const result = gsa_analyze(file.name, uint8)

        const decoder = new TextDecoder()
        return decoder.decode(result)
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
                    <DialogContentText>"Loading WebAssembly module...</DialogContentText>
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
    }, [loadError, loading, file, jsonResult, analyzing])

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

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <CssBaseline/>
        <App/>
    </React.StrictMode>,
)
