export interface Section {
    name: string;
    size: number;
    known_size: number;
    offset: number;
    end: number;
    addr: number;
    addr_end: number;
    only_in_memory: boolean;
}

export interface File {
    file_path: string;
    size: number;
}

export interface Package {
    name: string;
    type: string;
    subPackages: {[key: string]: Package};
    files: File[];
    size: number;
}
export interface Result {
    name: string;
    size: number;
    packages: {[key: string]: Package};
    sections: Section[];
}
