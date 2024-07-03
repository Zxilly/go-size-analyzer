import type { tags } from "typia"

export interface Section {
    name: string
    size: number & tags.Type<"uint64">
    file_size: number & tags.Type<"uint64">
    known_size: number & tags.Type<"uint64">
    offset: number & tags.Type<"uint64">
    end: number & tags.Type<"uint64">
    addr: number & tags.Type<"uint64">
    addr_end: number & tags.Type<"uint64">
    only_in_memory: boolean
    debug: boolean
}
export interface File {
    file_path: string
    size: number & tags.Type<"uint64">
    pcln_size: number & tags.Type<"uint64">
}
export interface FileSymbol {
    name: string
    addr: number & tags.Type<"uint64">
    size: number & tags.Type<"uint64">
    type: "unknown" | "text" | "data"
}
export interface Package {
    name: string
    type: "main" | "std" | "vendor" | "generated" | "unknown" | "cgo"
    subPackages: {
        [key: string]: Package
    }
    files: File[]
    symbols: FileSymbol[]
    size: number & tags.Type<"uint64">
}
export interface Result {
    name: string
    size: number & tags.Type<"uint64">
    packages: {
        [key: string]: Package
    }
    sections: Section[]
    analyzers: (("dwarf" | "disasm" | "symbol" | "pclntab")[]) | undefined
}
export function parseResult(input: any): import("typia").Primitive<Result> {
    const is = (input: any): input is Result => {
        const $io0 = (input: any): boolean => typeof input.name === "string" && (typeof input.size === "number" && (Math.floor(input.size) === input.size && input.size >= 0 && input.size <= 18446744073709552000)) && (typeof input.packages === "object" && input.packages !== null && Array.isArray(input.packages) === false && $io1(input.packages)) && (Array.isArray(input.sections) && input.sections.every((elem: any) => typeof elem === "object" && elem !== null && $io6(elem))) && (undefined === input.analyzers || Array.isArray(input.analyzers) && input.analyzers.every((elem: any) => elem === "symbol" || elem === "dwarf" || elem === "disasm" || elem === "pclntab"))
        const $io1 = (input: any): boolean => Object.keys(input).every((key: any) => {
            const value = input[key]
            if (undefined === value)
                return true
            return typeof value === "object" && value !== null && $io2(value)
        })
        const $io2 = (input: any): boolean => typeof input.name === "string" && (input.type === "main" || input.type === "std" || input.type === "vendor" || input.type === "generated" || input.type === "unknown" || input.type === "cgo") && (typeof input.subPackages === "object" && input.subPackages !== null && Array.isArray(input.subPackages) === false && $io3(input.subPackages)) && (Array.isArray(input.files) && input.files.every((elem: any) => typeof elem === "object" && elem !== null && $io4(elem))) && (Array.isArray(input.symbols) && input.symbols.every((elem: any) => typeof elem === "object" && elem !== null && $io5(elem))) && (typeof input.size === "number" && (Math.floor(input.size) === input.size && input.size >= 0 && input.size <= 18446744073709552000))
        const $io3 = (input: any): boolean => Object.keys(input).every((key: any) => {
            const value = input[key]
            if (undefined === value)
                return true
            return typeof value === "object" && value !== null && $io2(value)
        })
        const $io4 = (input: any): boolean => typeof input.file_path === "string" && (typeof input.size === "number" && (Math.floor(input.size) === input.size && input.size >= 0 && input.size <= 18446744073709552000)) && (typeof input.pcln_size === "number" && (Math.floor(input.pcln_size) === input.pcln_size && input.pcln_size >= 0 && input.pcln_size <= 18446744073709552000))
        const $io5 = (input: any): boolean => typeof input.name === "string" && (typeof input.addr === "number" && (Math.floor(input.addr) === input.addr && input.addr >= 0 && input.addr <= 18446744073709552000)) && (typeof input.size === "number" && (Math.floor(input.size) === input.size && input.size >= 0 && input.size <= 18446744073709552000)) && (input.type === "unknown" || input.type === "text" || input.type === "data")
        const $io6 = (input: any): boolean => typeof input.name === "string" && (typeof input.size === "number" && (Math.floor(input.size) === input.size && input.size >= 0 && input.size <= 18446744073709552000)) && (typeof input.file_size === "number" && (Math.floor(input.file_size) === input.file_size && input.file_size >= 0 && input.file_size <= 18446744073709552000)) && (typeof input.known_size === "number" && (Math.floor(input.known_size) === input.known_size && input.known_size >= 0 && input.known_size <= 18446744073709552000)) && (typeof input.offset === "number" && (Math.floor(input.offset) === input.offset && input.offset >= 0 && input.offset <= 18446744073709552000)) && (typeof input.end === "number" && (Math.floor(input.end) === input.end && input.end >= 0 && input.end <= 18446744073709552000)) && (typeof input.addr === "number" && (Math.floor(input.addr) === input.addr && input.addr >= 0 && input.addr <= 18446744073709552000)) && (typeof input.addr_end === "number" && (Math.floor(input.addr_end) === input.addr_end && input.addr_end >= 0 && input.addr_end <= 18446744073709552000)) && typeof input.only_in_memory === "boolean" && typeof input.debug === "boolean"
        return typeof input === "object" && input !== null && $io0(input)
    }; input = JSON.parse(input); return is(input) ? input as any : null
}
