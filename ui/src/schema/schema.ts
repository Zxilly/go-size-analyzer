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

export const isSection = typia.createIs<Section>();

export interface File {
    file_path: string;
    size: number & tags.Type<'uint64'>;
    pcln_size: number & tags.Type<'uint64'>;
}

export const isFile = typia.createIs<File>();

export interface Symbol {
    name: string
    addr: number & tags.Type<'uint64'>;
    size: number & tags.Type<'uint64'>;
    type: "unknown" | "text" | "data"
}

export const isSymbol = typia.createIs<Symbol>();

export interface Package {
    name: string;
    type: 'main' | 'std' | 'vendor' | 'generated' | 'unknown';
    subPackages: { [key: string]: Package };
    files: File[];
    symbols: Symbol[];
    size: number & tags.Type<'uint64'>;
}

export const isPackage = typia.createIs<Package>();

export interface Result {
    name: string;
    size: number & tags.Type<'uint64'>;
    packages: { [key: string]: Package };
    sections: Section[];
}

export const isResult = typia.createIs<Result>();

export const parseResult = typia.json.createIsParse<Result>();
