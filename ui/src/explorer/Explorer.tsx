import type { ReactNode } from "react";
import React, { useEffect, useMemo } from "react";
import { useAsync } from "react-use";
import { Box, Dialog, DialogContent, DialogContentText, DialogTitle } from "@mui/material";
import gsa from "../../gsa.wasm?init";
import { createEntry } from "../tool/entry.ts";
import TreeMap from "../TreeMap.tsx";
import { FileSelector } from "./FileSelector.tsx";

import { resetCallback, setCallback } from "./fs.js";

import "../tool/wasm_exec.js";

type ModalState = {
  isOpen: false;
} | {
  isOpen: true;
  title: string;
  content: ReactNode;
};

declare function gsa_analyze(name: string, data: Uint8Array): import("../generated/schema.ts").Result;

export const Explorer: React.FC = () => {
  const go = useMemo(() => new Go(), []);

  const { value: inst, loading, error: loadError } = useAsync(async () => {
    return await gsa(go.importObject);
  });

  useAsync(async () => {
    if (loading || loadError || inst === undefined) {
      return;
    }

    return await go.run(inst);
  }, [inst]);

  const [file, setFile] = React.useState<File | null>(null);

  const [modalState, setModalState] = React.useState<ModalState>({ isOpen: false });

  const { value: result, loading: analyzing } = useAsync(async () => {
    if (!file) {
      return;
    }

    const bytes = await file.arrayBuffer();
    const uint8 = new Uint8Array(bytes);

    return gsa_analyze(file.name, uint8);
  }, [file]);

  const [log, setLog] = React.useState<string>("");

  const friendlyLog = useMemo(() => {
    if (log === "") {
      return "Waiting for log";
    }
    return log;
  }, [log]);

  useEffect(() => {
    setCallback((line) => {
      setLog(log => `${log + line}\n`);
    });

    return resetCallback;
  }, []);

  const entry = useMemo(() => {
    if (!result) {
      return null;
    }

    return createEntry(result);
  }, [result]);

  useEffect(() => {
    if (loadError) {
      setModalState({
        isOpen: true,
        title: "Error",
        content:
                    <DialogContentText>{loadError.message}</DialogContentText>,
      });
    }
    else if (loading) {
      setModalState({
        isOpen: true,
        title: "Loading",
        content:
          <DialogContentText>Loading WebAssembly module...</DialogContentText>,
      });
    }
    else if (!inst) {
      setModalState({
        isOpen: true,
        title: "Error",
        content: (
          <DialogContentText>
            Failed to load WebAssembly module
          </DialogContentText>
        ),
      });
    }
    else if (!file) {
      setModalState({
        isOpen: true,
        title: "Select a go binary",
        content: (
          <FileSelector handler={(file) => {
            setFile(file);
          }}
          />
        ),
      });
    }
    else if (analyzing) {
      setModalState({
        isOpen: true,
        title: `Analyzing ${file.name}`,
        content: (
          <Box fontFamily="monospace" component="pre">
            {friendlyLog}
          </Box>
        ),
      });
    }
    else if (!analyzing && !result && !entry) {
      setModalState({
        isOpen: true,
        title: `Failed to analyze ${file.name}`,
        content: (
          <Box fontFamily="monospace" component="pre">
            {friendlyLog}
          </Box>
        ),
      });
    }
    else {
      setModalState({ isOpen: false });
    }
  }, [loadError, loading, file, result, analyzing, inst, entry, friendlyLog]);

  return (
    <>
      <Dialog
        open={modalState.isOpen}
      >
        <DialogTitle>{modalState.isOpen && modalState.title}</DialogTitle>
        <DialogContent dividers>
          {modalState.isOpen && modalState.content}
        </DialogContent>
      </Dialog>
      {entry && <TreeMap entry={entry} />}
    </>
  );
};
