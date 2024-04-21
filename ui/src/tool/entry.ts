import {File, isFile, isPackage, isResult, isSection, Package, Result, Section} from "../generated/schema.ts";
import {id} from "./id.ts";
import {formatBytes} from "./utils.ts";
import {max} from "d3-array";

type Candidate = Section | File | Package | Result;

interface Disasm {
    name: string;
    size: number;
}

type EntryType = "section" | "file" | "package" | "result" | "disasm" | "unknown";

export class Entry {
    private readonly type: EntryType;
    private readonly data: Candidate | Disasm;
    private readonly size: number;
    private readonly name: string;
    private readonly children: Entry[] = [];
    private readonly uid = id();

    constructor(data: Candidate)
    constructor(name: string, size: number, type: EntryType)
    constructor(data_or_name: Candidate | string, size?: number, type?: EntryType) {
        if (typeof data_or_name === "string") {
            this.type = type!;
            this.data = {name: data_or_name, size: size!};
            this.size = size!;
            this.name = data_or_name;
            this.children = [];
            return
        }

        this.type = Entry.checkType(data_or_name);
        this.data = data_or_name;
        this.size = Entry.loadSize(data_or_name);
        this.name = Entry.candidateName(data_or_name, this.type);
        this.children = Entry.childrenFromData(data_or_name, this.type);
    }

    static childrenFromData(data: Candidate, type: EntryType): Entry[] {
        switch (type) {
            case "section":
            case "file":
                return []; // no children for section or file
            case "package":
                return childrenFromPackage(data as Package);
            case "result":
                return childrenFromResult(data as Result);
            default:
                throw new Error("Unknown candidate type");
        }
    }

    static candidateName(candidate: Candidate, type: EntryType): string {
        switch (type) {
            case "section":
            case "result":
            case "package":
                return (<Section | Result | Package>candidate).name;

            case "file":
                return (<File>candidate).file_path.split("/").pop()!;

            default:
                throw new Error("Unknown candidate type");
        }

    }

    static loadSize(data: Candidate): number {
        switch (true) {
            case isSection(data):
                return data.size - data.known_size;
            default:
                return data.size;
        }
    }

    static checkType(candidate: Candidate): EntryType {
        switch (true) {
            case isSection(candidate):
                return "section";
            case isFile(candidate):
                return "file";
            case isPackage(candidate):
                return "package";
            case isResult(candidate):
                return "result";
            default:
                throw new Error("Unknown candidate type");
        }
    }

    public toString(): string {
        const align = new aligner();

        function assertTyp<T extends Candidate | Disasm>(_c: Candidate | Disasm): asserts _c is T {
        }

        switch (this.type) {
            case "section":
                assertTyp<Section>(this.data);
                align.add("Section:", this.name);
                align.add("Size:", formatBytes(this.size));
                align.add("Known size:", formatBytes(this.data.known_size));
                align.add("Unknown size:", formatBytes(this.data.size - this.data.known_size));
                align.add("Offset:", `0x${this.data.offset.toString(16)} - 0x${this.data.end.toString(16)}`);
                align.add("Address:", `0x${this.data.addr.toString(16)} - 0x${this.data.addr_end.toString(16)}`);
                align.add("Memory:", this.data.only_in_memory.toString());
                return align.toString();

            case "file":
                assertTyp<File>(this.data);
                align.add("File:", this.data.file_path);
                align.add("Path:", this.data.file_path);
                align.add("Size:", formatBytes(this.data.size));
                return align.toString();

            case "package":
                assertTyp<Package>(this.data);
                align.add("Package:", this.data.name);
                align.add("Type:", this.data.type);
                align.add("Size:", formatBytes(this.data.size));
                return align.toString();

            case "result":
                assertTyp<Result>(this.data);
                align.add("Result:", this.data.name);
                align.add("Size:", formatBytes(this.data.size));
                return align.toString();

            case "disasm": {
                align.add("Disasm:", this.name);
                align.add("Size:", formatBytes(this.size));
                let ret = align.toString();
                ret += "\n" +
                    "This size was not accurate." +
                    "The real size determined by disassembling can be larger.";
                return ret;
            }

            case "unknown": {
                align.add("Size:", formatBytes(this.size));
                let ret = align.toString();
                ret += "\n\n" +
                    "The unknown part in the binary.\n" +
                    "Can be ELF Header, Program Header, align offset...\n" +
                    "We just don't know.";
                return ret;
            }

            default:
                throw new Error("Unknown candidate type");
        }
    }

    public getSize(): number {
        return this.size;
    }

    public getName(): string {
        return this.name;
    }

    public getChildren(): Entry[] {
        return this.children;
    }

    public getData(): Candidate | Disasm {
        return this.data;
    }

    public getID(): string {
        return this.uid.toString(16);
    }
}

function childrenFromPackage(pkg: Package): Entry[] {
    const children: Entry[] = [];
    for (const file of pkg.files) {
        children.push(new Entry(file));
    }
    for (const subPackage of Object.values(pkg.subPackages)) {
        children.push(new Entry(subPackage));
    }

    const leftSize = pkg.size - children.reduce((acc, child) => acc + child.getSize(), 0);
    if (leftSize > 0) {
        const name = `${pkg.name} Disasm`
        children.push(new Entry(name, leftSize, "disasm"));
    }

    return children;
}

function childrenFromResult(result: Result): Entry[] {
    const children: Entry[] = [];
    for (const section of result.sections) {
        children.push(new Entry(section));
    }
    for (const pkg of Object.values(result.packages)) {
        children.push(new Entry(pkg));
    }
    const leftSize = result.size - children.reduce((acc, child) => acc + child.getSize(), 0);
    if (leftSize > 0) {
        const name = `Unknown`
        children.push(new Entry(name, leftSize, "unknown"));
    }

    return children;
}

class aligner {
    private pre: string[] = [];
    private post: string[] = [];

    public add(pre: string, post: string): void {
        this.pre.push(pre);
        this.post.push(post);
    }

    public toString(): string {
        // determine the maximum length of the pre strings
        const maxPreLength = max(this.pre, (d) => d.length) ?? 0;
        let ret = "";
        for (let i = 0; i < this.pre.length; i++) {
            ret += this.pre[i].padEnd(maxPreLength + 1) + this.post[i] + "\n";
        }
        ret = ret.trimEnd();
        return ret;
    }
}
