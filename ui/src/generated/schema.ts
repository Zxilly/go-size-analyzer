import { tags } from "typia";
export interface Section {
    name: string;
    size: number & tags.Type<'uint64'>;
    file_size: number & tags.Type<'uint64'>;
    known_size: number & tags.Type<'uint64'>;
    offset: number & tags.Type<'uint64'>;
    end: number & tags.Type<'uint64'>;
    addr: number & tags.Type<'uint64'>;
    addr_end: number & tags.Type<'uint64'>;
    only_in_memory: boolean;
    debug: boolean;
}
export interface File {
    file_path: string;
    size: number & tags.Type<'uint64'>;
    pcln_size: number & tags.Type<'uint64'>;
}
export interface FileSymbol {
    name: string;
    addr: number & tags.Type<'uint64'>;
    size: number & tags.Type<'uint64'>;
    type: "unknown" | "text" | "data";
}
export interface Package {
    name: string;
    type: 'main' | 'std' | 'vendor' | 'generated' | 'unknown' | 'cgo';
    subPackages: {
        [key: string]: Package;
    };
    files: File[];
    symbols: FileSymbol[];
    size: number & tags.Type<'uint64'>;
}
export interface Result {
    name: string;
    size: number & tags.Type<'uint64'>;
    packages: {
        [key: string]: Package;
    };
    sections: Section[];
}
export const parseResult = (input: any): import("typia").Primitive<Result> => { const is = (input: any): input is Result => {
    const $io0 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.size && (Math.floor(input.size) === input.size && 0 <= input.size && input.size <= 18446744073709552000)) && ("object" === typeof input.packages && null !== input.packages && false === Array.isArray(input.packages) && $io1(input.packages)) && (Array.isArray(input.sections) && input.sections.every((elem: any) => "object" === typeof elem && null !== elem && $io6(elem)));
    const $io1 = (input: any): boolean => Object.keys(input).every((key: any) => {
        const value = input[key];
        if (undefined === value)
            return true;
        return "object" === typeof value && null !== value && $io2(value);
    });
    const $io2 = (input: any): boolean => "string" === typeof input.name && ("main" === input.type || "std" === input.type || "vendor" === input.type || "generated" === input.type || "unknown" === input.type || "cgo" === input.type) && ("object" === typeof input.subPackages && null !== input.subPackages && false === Array.isArray(input.subPackages) && $io3(input.subPackages)) && (Array.isArray(input.files) && input.files.every((elem: any) => "object" === typeof elem && null !== elem && $io4(elem))) && (Array.isArray(input.symbols) && input.symbols.every((elem: any) => "object" === typeof elem && null !== elem && $io5(elem))) && ("number" === typeof input.size && (Math.floor(input.size) === input.size && 0 <= input.size && input.size <= 18446744073709552000));
    const $io3 = (input: any): boolean => Object.keys(input).every((key: any) => {
        const value = input[key];
        if (undefined === value)
            return true;
        return "object" === typeof value && null !== value && $io2(value);
    });
    const $io4 = (input: any): boolean => "string" === typeof input.file_path && ("number" === typeof input.size && (Math.floor(input.size) === input.size && 0 <= input.size && input.size <= 18446744073709552000)) && ("number" === typeof input.pcln_size && (Math.floor(input.pcln_size) === input.pcln_size && 0 <= input.pcln_size && input.pcln_size <= 18446744073709552000));
    const $io5 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.addr && (Math.floor(input.addr) === input.addr && 0 <= input.addr && input.addr <= 18446744073709552000)) && ("number" === typeof input.size && (Math.floor(input.size) === input.size && 0 <= input.size && input.size <= 18446744073709552000)) && ("unknown" === input.type || "text" === input.type || "data" === input.type);
    const $io6 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.size && (Math.floor(input.size) === input.size && 0 <= input.size && input.size <= 18446744073709552000)) && ("number" === typeof input.file_size && (Math.floor(input.file_size) === input.file_size && 0 <= input.file_size && input.file_size <= 18446744073709552000)) && ("number" === typeof input.known_size && (Math.floor(input.known_size) === input.known_size && 0 <= input.known_size && input.known_size <= 18446744073709552000)) && ("number" === typeof input.offset && (Math.floor(input.offset) === input.offset && 0 <= input.offset && input.offset <= 18446744073709552000)) && ("number" === typeof input.end && (Math.floor(input.end) === input.end && 0 <= input.end && input.end <= 18446744073709552000)) && ("number" === typeof input.addr && (Math.floor(input.addr) === input.addr && 0 <= input.addr && input.addr <= 18446744073709552000)) && ("number" === typeof input.addr_end && (Math.floor(input.addr_end) === input.addr_end && 0 <= input.addr_end && input.addr_end <= 18446744073709552000)) && "boolean" === typeof input.only_in_memory && "boolean" === typeof input.debug;
    return "object" === typeof input && null !== input && $io0(input);
}; input = JSON.parse(input); return is(input) ? input as any : null; };
