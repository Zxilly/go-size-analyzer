import type {
  GenericSchema,
  InferInput,
} from "valibot";
import {
  array,
  boolean,
  lazy,
  literal,
  number,
  object,
  optional,
  record,
  safeParse,
  string,
  union,
} from "valibot";

export const SectionSchema = object({
  name: string(),
  size: number(),
  file_size: number(),
  known_size: number(),
  offset: number(),
  end: number(),
  addr: number(),
  addr_end: number(),
  only_in_memory: boolean(),
  debug: boolean(),
});

export type Section = InferInput<typeof SectionSchema>;

export const FileSchema = object({
  file_path: string(),
  size: number(),
  pcln_size: number(),
});

export type File = InferInput<typeof FileSchema>;

export const FileSymbolSchema = object({
  name: string(),
  addr: number(),
  size: number(),
  type: union([literal("unknown"), literal("text"), literal("data")]),
});

export type FileSymbol = InferInput<typeof FileSymbolSchema>;

interface PackageRef {
  name: string;
  type: "main" | "std" | "vendor" | "generated" | "unknown" | "cgo";
  subPackages: Record<string, PackageRef>;
  files: File[];
  symbols: FileSymbol[];
  size: number;
}

export const PackageSchema: GenericSchema<PackageRef> = object({
  name: string(),
  type: union([
    literal("main"),
    literal("std"),
    literal("vendor"),
    literal("generated"),
    literal("unknown"),
    literal("cgo"),
  ]),
  subPackages: record(string(), lazy(() => PackageSchema)),
  files: array(FileSchema),
  symbols: array(FileSymbolSchema),
  size: number(),
});

export const ResultSchema = object({
  name: string(),
  size: number(),
  packages: record(string(), PackageSchema),
  sections: array(SectionSchema),
  analyzers: optional(array(union([literal("dwarf"), literal("disasm"), literal("symbol"), literal("pclntab")]))),
});

export type Result = InferInput<typeof ResultSchema>;

export function parseResult(data: string): Result | null {
  try {
    const obj = JSON.parse(data);
    const result = safeParse(ResultSchema, obj);

    if (result.success) {
      return result.output;
    }

    console.warn(result.issues);
    return null;
  }
  catch (error) {
    console.error("Failed to parse JSON:", error);
    return null;
  }
}
