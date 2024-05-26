import {useAsync} from "react-use";
import ReactDOM from "react-dom/client";
import React, {ChangeEvent, ReactNode, useCallback, useEffect, useMemo} from "react";
import {
    Button,
    ChakraProvider,
    Input,
    Modal,
    ModalBody,
    ModalContent,
    ModalFooter,
    ModalHeader,
    ModalOverlay
} from '@chakra-ui/react'

import "./tool/wasm_exec.js"
import gsa from "../gsa.wasm?init"
import {loadDataFromWasmResult} from "./tool/utils.ts";
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

const FileSelector = ({handler}: {
    handler: fileChangeHandler
}) => {
    const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
        if (event.target.files && event.target.files.length > 0) {
            const file = event.target.files[0]
            handler(file)
        }
    };

    return (
        <div>
            <Input type="file" onChange={handleFileChange} hidden id="file-upload"/>
            <label htmlFor="file-upload">
                <Button as="span">
                    Select a file
                </Button>
            </label>
        </div>
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
            setModalState({isOpen: true, title: "Error", content: loadError.message})
        } else if (loading) {
            setModalState({isOpen: true, title: "Loading", content: "Loading WebAssembly module..."})
        } else if (inst === undefined) {
            setModalState({isOpen: true, title: "Error", content: "Failed to load WebAssembly module"})
        } else if (file === null) {
            setModalState({
                isOpen: true,
                title: "Select a go binary",
                content: <FileSelector handler={(file) => {
                    setFile(file)
                }}/>
            })
        } else if (analyzing) {
            setModalState({isOpen: true, title: "Analyzing", content: "Analyzing binary..."})
        } else if (!analyzing && !jsonResult) {
            setModalState({
                isOpen: true,
                title: "Error",
                content: "Failed to analyze " + file.name + ", see console for more details"
            })
        } else {
            setModalState({isOpen: false})
        }
    }, [loadError, loading, file, jsonResult])

    const closeModal = useCallback(() => {
        setModalState({isOpen: false})
    }, [])

    return <>
        <Modal
            isOpen={modalState.isOpen}
            onClose={closeModal}
            closeOnOverlayClick={false}
            isCentered
        >
            <ModalOverlay/>
            <ModalContent>
                <ModalHeader>{modalState.isOpen && modalState.title}</ModalHeader>
                <ModalBody>
                    {modalState.isOpen && modalState.content}
                </ModalBody>
                <ModalFooter/>
            </ModalContent>
        </Modal>
        {entry && <TreeMap entry={entry}/>}
    </>
}

ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <ChakraProvider resetCSS>
            <App/>
        </ChakraProvider>
    </React.StrictMode>,
)
