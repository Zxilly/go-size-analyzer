import {
    File,
    isFile,
    isPackage,
    isResult,
    isSection,
    isSymbol,
    Package,
    Result,
    Section,
    Symbol as FileSymbol
} from "../generated/schema.ts";
import {id} from "./id.ts";
import {formatBytes, title} from "./utils.ts";
import {max} from "d3-array";

type Candidate = Section | File | Package | Result | FileSymbol;

type EntryType = "section" | "file" | "package" | "result" | "symbol" | "disasm" | "unknown" | "container";

export class Entry {
    private readonly type: EntryType;
    private readonly data?: Candidate;
    private readonly size: number;
    private readonly name: string;
    private readonly children: Entry[] = [];
    private readonly uid = id();
    explain: string = ""; // should only be used by the container type

    constructor(data: Candidate)
    constructor(name: string, size: number, type: EntryType, children?: Entry[])
    constructor(data_or_name: Candidate | string, size?: number, type?: EntryType, children: Entry[] = []) {
        if (typeof data_or_name === "string") {
            this.type = type!;
            this.size = size!;
            this.name = data_or_name;
            this.children = children;
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
            case "symbol":
                return []; // no children for section or file
            case "package":
                return childrenFromPackage(data as Package);
            case "result":
                return childrenFromResult(data as Result);
            default:
                throw new Error(`Unknown type: ${type} in childrenFromData`);
        }
    }

    static candidateName(candidate: Candidate, type: EntryType): string {
        switch (type) {
            case "section":
            case "result":
            case "package":
            case "symbol":
                return (<Section | Result | Package | FileSymbol>candidate).name;

            case "file":
                return (<File>candidate).file_path.split("/").pop()!;

            default:
                throw new Error(`Unknown type: ${type} in candidateName`);
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
            case isSymbol(candidate):
                return "symbol";
            default:
                throw new Error(`Unknown type in checkType`);
        }
    }

    public toString(): string {
        const align = new aligner();

        function assertTyp<T extends Candidate>(_c?: Candidate): asserts _c is T {
        }

        switch (this.type) {
            case "section":
                assertTyp<Section>(this.data);
                align.add("Section:", this.name);
                align.add("Size:", formatBytes(this.size));
                align.add("File Size:", formatBytes(this.data.file_size));
                align.add("Known size:", formatBytes(this.data.known_size));
                align.add("Unknown size:", formatBytes(this.data.size - this.data.known_size));
                align.add("Offset:", `0x${this.data.offset.toString(16)} - 0x${this.data.end.toString(16)}`);
                align.add("Address:", `0x${this.data.addr.toString(16)} - 0x${this.data.addr_end.toString(16)}`);
                align.add("Memory:", this.data.only_in_memory.toString());
                align.add("Debug:", this.data.debug.toString());
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
                ret += "\n\n" +
                    "This size was not accurate." +
                    "The real size determined by disassembling can be larger.";
                return ret;
            }

            case "symbol": {
                assertTyp<FileSymbol>(this.data);
                align.add("Symbol:", this.data.name);
                align.add("Size:", formatBytes(this.size));
                align.add("Address:", `0x${this.data.addr.toString(16)}`);
                align.add("Type:", this.data.type);
                return align.toString();
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

            case "container": {
                let ret = this.explain + "\n"
                align.add("Size:", formatBytes(this.size));
                ret += "\n" + align.toString();
                return ret;
            }
        }
    }

    public getSize(): number {
        return this.size;
    }

    public getType(): EntryType {
        return this.type;
    }

    public getName(): string {
        return this.name;
    }

    public getChildren(): Entry[] {
        return this.children;
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

    for (const s of pkg.symbols) {
        children.push(new Entry(s));
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

    const sectionContainerChildren: Entry[] = []
    for (const section of result.sections) {
        sectionContainerChildren.push(new Entry(section));
    }
    const sectionContainerSize = sectionContainerChildren.reduce((acc, child) => acc + child.getSize(), 0);
    const sectionContainer = new Entry("Unknown Sections Size", sectionContainerSize, "container", sectionContainerChildren);
    sectionContainer.explain = "The unknown size of the sections in the binary."
    children.push(sectionContainer);

    const typedPackages: Record<string, Package[]> = {};
    for (const pkg of Object.values(result.packages)) {
        if (typedPackages[pkg.type] == null) {
            typedPackages[pkg.type] = [];
        }
        typedPackages[pkg.type].push(pkg);
    }
    const typedPackagesChildren: Entry[] = []
    for (const [type, packages] of Object.entries(typedPackages)) {
        const packageContainerChildren: Entry[] = [];
        for (const pkg of packages) {
            packageContainerChildren.push(new Entry(pkg));
        }
        const packageContainerSize = packageContainerChildren.reduce((acc, child) => acc + child.getSize(), 0);
        const packageContainer = new Entry(`${title(type)} Packages Size`, packageContainerSize, "container", packageContainerChildren);
        packageContainer.explain = `The size of the ${type} packages in the binary.`
        typedPackagesChildren.push(packageContainer);
    }
    const packageContainerSize = typedPackagesChildren.reduce((acc, child) => acc + child.getSize(), 0);
    const packageContainer = new Entry("Packages Size", packageContainerSize, "container", typedPackagesChildren);
    packageContainer.explain = "The size of the packages in the binary."
    children.push(packageContainer);

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
        // determine the maximum length of the pre-strings
        const maxPreLength = max(this.pre, (d) => d.length) ?? 0;
        let ret = "";
        for (let i = 0; i < this.pre.length; i++) {
            ret += this.pre[i].padEnd(maxPreLength + 1) + this.post[i] + "\n";
        }
        ret = ret.trimEnd();
        return ret;
    }
}
