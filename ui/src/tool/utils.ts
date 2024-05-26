import {Result} from "../schema/schema.ts";
import {parseResult} from "../generated/schema.ts";
import {useCallback, useRef} from 'react';

export function loadDataFromEmbed(): Result {
    const doc = document.querySelector("#data")!;
    const ret = parseResult(doc.textContent!);
    if (ret === null) {
        throw new Error("Failed to parse data");
    }
    return ret;
}

export function loadDataFromWasmResult(data: string): Result {
    const ret = parseResult(data);
    if (ret === null) {
        throw new Error("Failed to parse data");
    }
    return ret;
}


export function formatBytes(bytes: number) {
    if (bytes == 0) return '0 B';
    const k = 1024,
        dm = 2,
        sizes = ['B', 'KB', 'MB', 'GB'],
        i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function title(s: string): string {
    return s[0].toUpperCase() + s.slice(1);
}

export function trimPrefix(str: string, prefix: string) {
    if (str.startsWith(prefix)) {
        return str.slice(prefix.length)
    } else {
        return str
    }
}

export function useThrottle<T extends (...args: Parameters<T>) => ReturnType<T>>(func: T, delay: number): (...args: Parameters<T>) => void {
    const lastCall = useRef<number>(0);
    const lastFunc = useRef<T>(func);

    lastFunc.current = func;

    return useCallback((...args: Parameters<T>) => {
        const now = Date.now();

        if (now - lastCall.current >= delay) {
            lastCall.current = now;
            lastFunc.current(...args);
        }
    }, [delay]);
}
