// The Go WASM runtime's filesystem bridge is installed by runtime/fs.js.
declare namespace fs {
  function writeSync(fd: number, buffer: Uint8Array): number;
}
