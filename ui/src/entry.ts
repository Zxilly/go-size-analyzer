import {File, isFile, isPackage, isResult, isSection, Package, Result, Section} from "./generated/schema.ts";
import {id} from "./id.ts";

type Candidate = Section | File | Package | Result;

type EntryType = "section" | "file" | "package" | "result";


export class Entry {
    private readonly type: EntryType;
    private readonly data: Candidate;
    private readonly size: number;
    private readonly name: string;
    private readonly children: Entry[] = [];
    private readonly uid = id();

    constructor(data: Candidate) {
        this.type = Entry.checkType(data);
        this.data = data;
        this.size = data.size;
        this.name = candidateName(data, this.type);
        this.children = childrenFromData(data, this.type);
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
        return candidateStringify(this.data);
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

    public getData(): Candidate {
        return this.data;
    }

    public getID(): string {
        return this.uid.toString(16);
    }
}

function childrenFromData(data: Candidate, type: EntryType): Entry[] {
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

function childrenFromPackage(pkg: Package): Entry[] {
    const children: Entry[] = [];
    for (const file of pkg.files) {
        children.push(new Entry(file));
    }
    for (const subPackage of Object.values(pkg.subPackages)) {
        children.push(new Entry(subPackage));
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
    return children;
}

function formatBytes(bytes: number) {
    if (bytes == 0) return '0 B';
    const k = 1024,
        dm = 2,
        sizes = ['B', 'KB', 'MB', 'GB'],
        i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

function candidateName(candidate: Candidate, type: EntryType): string {
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

function candidateStringify(candidate: Candidate): string {
    switch (true) {
        case isSection(candidate):
            return `Section: ${candidate.name}\n` +
                `Size: ${formatBytes(candidate.size)}\n` +
                `Known size: ${formatBytes(candidate.known_size)}\n` +
                `Offset: ${candidate.offset.toString(16)} - ${candidate.end.toString(16)}\n` +
                `Address: ${candidate.addr.toString(16)} - ${candidate.addr_end.toString(16)}\n` +
                `Only in memory: ${candidate.only_in_memory}\n`;

        case isFile(candidate):
            return `File: ${candidate.file_path}\n` +
                `Path: ${candidate.file_path}\n` +
                `Size: ${formatBytes(candidate.size)}\n`;

        case isPackage(candidate):
            return `Package: ${candidate.name}\n` +
                `Type: ${candidate.type}\n` +
                `Size: ${formatBytes(candidate.size)}\n`;

        case isResult(candidate):
            return `Result: ${candidate.name}\n` +
                `Size: ${formatBytes(candidate.size)}\n`;
        default:
            throw new Error("Unknown candidate type");
    }
}
