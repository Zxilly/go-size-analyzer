import React, { memo, useCallback, useState } from "react";
import { Box, Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, Link } from "@mui/material";
import { formatBytes } from "../tool/utils.ts";

const SizeLimit = 1024 * 1024 * 30;

type FileChangeHandler = (file: File) => void;

interface FileSelectorProps {
  handler: FileChangeHandler;
}

export const FileSelector: React.FC<FileSelectorProps> = memo(({ handler }) => {
  const [dialogState, setDialogState] = useState<{ open: boolean; file: File | null }>({ open: false, file: null });

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file)
      return;

    if (file.size > SizeLimit) {
      setDialogState({ open: true, file });
    }
    else {
      handler(file);
    }
  }, [handler]);

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
            It is not recommended to use the wasm version for binary files larger than 30 MB.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancel</Button>
          <Button onClick={handleContinue}>Continue</Button>
        </DialogActions>
      </Dialog>
      <Box display="flex" flexDirection="column" alignItems="center" height="100%">
        <Button variant="outlined" component="label">
          Select file
          <input
            type="file"
            onChange={handleChange}
            data-testid="file-selector"
            hidden
          />
        </Button>
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
