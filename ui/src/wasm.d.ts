// The Go WASM runtime's filesystem bridge is installed by runtime/fs.js.
// eslint-disable-next-line no-var -- Ambient var declarations describe properties of globalThis.
declare var fs: {
  writeSync: (fd: number, buffer: Uint8Array) => number;
};
