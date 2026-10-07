import { MAX_LOG_LENGTH } from "./logLimit.ts";
const truncationMarker = "[Earlier log output truncated]\n";
let outputBuf = "";
let truncated = false;
let progressMessages = 0;
const decoder = new TextDecoder("utf-8");
function enosys() {
  const err = new Error("not implemented");
  err.code = "ENOSYS";
  return err;
}

const defaultFSCallback = line => console.log(line);

let fsCallback = defaultFSCallback;

export function setCallback(callback) {
  fsCallback = callback;
}

export function resetLog() {
  outputBuf = "";
  truncated = false;
  progressMessages = 0;
}

// Keep at most four progress messages, then one bounded tail before the result.
export function flushLog() {
  if (outputBuf) {
    fsCallback((truncated ? truncationMarker : "") + outputBuf);
    outputBuf = "";
  }
}

export function resetCallback() {
  fsCallback = defaultFSCallback;
  resetLog();
}

globalThis.fs = {
  constants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1 }, // unused
  writeSync(fd, buf) {
    // Decode at most the retained tail, even for a single enormous write.
    const limit = MAX_LOG_LENGTH - truncationMarker.length;
    const chunk = buf.length > limit ? buf.subarray(buf.length - limit) : buf;
    const text = decoder.decode(chunk);
    if (buf.length > limit || outputBuf.length + text.length > limit) {
      truncated = true;
    }
    outputBuf = (outputBuf + text).slice(-limit);
    if (progressMessages < 4 && outputBuf.length >= 16 * 1024) {
      fsCallback(outputBuf.slice(0, 16 * 1024));
      outputBuf = outputBuf.slice(16 * 1024);
      progressMessages++;
    }
    return buf.length;
  },
  write(fd, buf, offset, length, position, callback) {
    if (offset !== 0 || length !== buf.length || position !== null) {
      callback(enosys());
      return;
    }
    const n = this.writeSync(fd, buf);
    callback(null, n);
  },
  chmod(path, mode, callback) { callback(enosys()); },
  chown(path, uid, gid, callback) { callback(enosys()); },
  close(fd, callback) { callback(enosys()); },
  fchmod(fd, mode, callback) { callback(enosys()); },
  fchown(fd, uid, gid, callback) { callback(enosys()); },
  fstat(fd, callback) { callback(enosys()); },
  fsync(fd, callback) { callback(null); },
  ftruncate(fd, length, callback) { callback(enosys()); },
  lchown(path, uid, gid, callback) { callback(enosys()); },
  link(path, link, callback) { callback(enosys()); },
  lstat(path, callback) { callback(enosys()); },
  mkdir(path, perm, callback) { callback(enosys()); },
  open(path, flags, mode, callback) { callback(enosys()); },
  read(fd, buffer, offset, length, position, callback) { callback(enosys()); },
  readdir(path, callback) { callback(enosys()); },
  readlink(path, callback) { callback(enosys()); },
  rename(from, to, callback) { callback(enosys()); },
  rmdir(path, callback) { callback(enosys()); },
  stat(path, callback) { callback(enosys()); },
  symlink(path, link, callback) { callback(enosys()); },
  truncate(path, length, callback) { callback(enosys()); },
  unlink(path, callback) { callback(enosys()); },
  utimes(path, atime, mtime, callback) { callback(enosys()); },
};
