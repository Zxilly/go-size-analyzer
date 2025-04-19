import Troubleshoot from "@mui/icons-material/Troubleshoot";
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Link,
  Typography,
} from "@mui/material";
import React, { memo, useCallback, useState } from "react";
import { useDropzone } from "react-dropzone";
import { formatBytes } from "../tool/utils.ts";

const SizeLimit = 1024 * 1024 * 30;

type FileChangeHandler = (file: File) => void;

interface FileSelectorProps {
  handler: FileChangeHandler;
}

export const FileSelector: React.FC<FileSelectorProps> = memo(({ handler }) => {
  const [dialogState, setDialogState] = useState<{ open: boolean; file: File | null }>({ open: false, file: null });

  const handleFile = useCallback((file: File) => {
    if (file.size > SizeLimit) {
      setDialogState({ open: true, file });
    }
    else {
      handler(file);
    }
  }, [handler]);

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles && acceptedFiles.length > 0) {
      handleFile(acceptedFiles[0]);
    }
  }, [handleFile]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({ onDrop });

  const handleClose = useCallback(() => setDialogState(prev => ({ ...prev, open: false })), []);

  const handleContinue = useCallback(() => {
    if (dialogState.file) {
      handler(dialogState.file);
      setDialogState({ open: false, file: null });
    }
  }, [handler, dialogState.file]);

  return (
    <>
      <Dialog open={dialogState.open} onClose={handleClose}>
        <DialogTitle>Binary too large</DialogTitle>
        <DialogContent>
          <DialogContentText>
            The selected binary
            {" "}
            {dialogState.file?.name}
            {" "}
            has a size of
            {" "}
            {formatBytes(dialogState.file?.size || 0)}
            .
            It is not recommended to use the WebAssembly version for binary files larger than 30 MB.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button onClick={handleContinue}>Continue</Button>
        </DialogActions>
      </Dialog>
      <Box display="flex" flexDirection="column" alignItems="center" height="100%">
        <Box
          {...getRootProps()}
          sx={{
            "border": "2px dashed #cccccc",
            "borderRadius": "4px",
            "padding": "20px",
            "textAlign": "center",
            "cursor": "pointer",
            "height": "120px",
            "display": "flex",
            "flexDirection": "column",
            "justifyContent": "center",
            "alignItems": "center",
            "transition": "all 0.3s ease",
            "backgroundColor": isDragActive ? "#e8f0fe" : "transparent",
            "&:hover": {
              backgroundColor: "#f0f0f0",
            },
            "minWidth": "20vw",
            "width": "100%",
          }}
        >
          <input {...getInputProps()} data-testid="file-selector" />
          <Troubleshoot sx={{ fontSize: 40, mb: 1, color: isDragActive ? "#1976d2" : "#757575" }} />
          <Typography
            variant="body1"
            sx={{
              color: isDragActive ? "#1976d2" : "inherit", // Change color on drag, if desired
            }}
          >
            {isDragActive ? "Drop the file here" : "Click or drag file to analyze"}
          </Typography>
        </Box>
        <DialogContentText sx={{ mt: 2, width: "100%" }}>
          For full features, see
          <Link
            href="https://github.com/Zxilly/go-size-analyzer"
            target="_blank"
            rel="noreferrer noopener"
            sx={{ ml: 0.5 }}
          >
            go-size-analyzer
          </Link>
        </DialogContentText>
      </Box>
    </>
  );
});
