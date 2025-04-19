import { z } from "zod";

export const SectionSchema = z.object({
  name: z.string(),
  size: z.number(),
  file_size: z.number(),
  known_size: z.number(),
  offset: z.number(),
  end: z.number(),
  addr: z.number(),
  addr_end: z.number(),
  only_in_memory: z.boolean(),
  debug: z.boolean(),
});

export type Section = z.infer<typeof SectionSchema>;

export const FileSchema = z.object({
  file_path: z.string(),
  size: z.number(),
  pcln_size: z.number(),
});

export type File = z.infer<typeof FileSchema>;

export const FileSymbolSchema = z.object({
  name: z.string(),
  addr: z.number(),
  size: z.number(),
  type: z.union([z.literal("unknown"), z.literal("text"), z.literal("data")]),
});

export type FileSymbol = z.infer<typeof FileSymbolSchema>;

interface packageRefer {
  name: string;
  type: "main" | "std" | "vendor" | "generated" | "unknown" | "cgo";
  subPackages: Record<string, Package>;
  files: File[];
  symbols: FileSymbol[];
  size: number;
}

export const PackageSchema: z.ZodSchema<packageRefer> = z.lazy(() =>
  z.object({
    name: z.string(),
    type: z.union([
      z.literal("main"),
      z.literal("std"),
      z.literal("vendor"),
      z.literal("generated"),
      z.literal("unknown"),
      z.literal("cgo"),
    ]),
    subPackages: z.record(PackageSchema),
    files: z.array(FileSchema),
    symbols: z.array(FileSymbolSchema),
    size: z.number(),
  }),
);

export type Package = z.infer<typeof PackageSchema>;

export const ResultSchema = z.object({
  name: z.string(),
  size: z.number(),
  packages: z.record(PackageSchema),
  sections: z.array(SectionSchema),
  analyzers: z.union([
    z.array(
      z.union([
        z.literal("dwarf"),
        z.literal("disasm"),
        z.literal("symbol"),
        z.literal("pclntab"),
      ]),
    ),
    z.undefined(),
  ]),
});

export type Result = z.infer<typeof ResultSchema>;

export function parseResult(data: string): Result | null {
  const obj = JSON.parse(data);

  const ret = ResultSchema.safeParse(obj);
  if (ret.success) {
    return ret.data;
  }
  console.warn(ret.error);
  return null;
}
