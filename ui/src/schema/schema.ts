import typia, {tags} from "typia";

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
    name: string
    addr: number & tags.Type<'uint64'>;
    size: number & tags.Type<'uint64'>;
    type: "unknown" | "text" | "data"
}

export interface Package {
    name: string;
    type: 'main' | 'std' | 'vendor' | 'generated' | 'unknown' | 'cgo';
    subPackages: { [key: string]: Package };
    files: File[];
    symbols: FileSymbol[];
    size: number & tags.Type<'uint64'>;
}

export interface Result {
    name: string;
    size: number & tags.Type<'uint64'>;
    packages: { [key: string]: Package };
    sections: Section[];
    analyzers: (("dwarf" | "disasm" | "symbol" | "pclntab")[]) | undefined;
}

export const parseResult = typia.json.createIsParse<Result>();
