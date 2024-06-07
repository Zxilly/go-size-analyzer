import React, {useCallback, useState} from "react";
import {Box, Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, Link} from "@mui/material";
import {formatBytes} from "../tool/utils.ts";

const SizeLimit = 1024 * 1024 * 30

type fileChangeHandler = (file: File) => void

export const FileSelector = ({handler}: {
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

    }, [handler])

    const handleClose = useCallback(() => {
        setOpen(false)
    }, [])

    const handleContinue = useCallback(() => {
        if (pendingFile) {
            handler(pendingFile)
            setOpen(false)
        }
    }, [handler, pendingFile])

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
                        It is not recommended to use the wasm version for binary files larger than 30 MB.
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={handleClose}>Cancel</Button>
                    <Button onClick={handleContinue}>Continue</Button>
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