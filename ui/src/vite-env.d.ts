/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly PACKAGE_DATA: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}

declare const PACKAGE_DATA: string;
