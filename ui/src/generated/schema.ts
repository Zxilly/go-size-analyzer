import * as __typia_transform__isTypeUint64 from "typia/lib/internal/_isTypeUint64.js";
import type {tags} from "typia";

export interface Section {
  name: string;
  size: number & tags.Type<"uint64">;
  file_size: number & tags.Type<"uint64">;
  known_size: number & tags.Type<"uint64">;
  offset: number & tags.Type<"uint64">;
  end: number & tags.Type<"uint64">;
  addr: number & tags.Type<"uint64">;
  addr_end: number & tags.Type<"uint64">;
  only_in_memory: boolean;
  debug: boolean;
}

export interface File {
  file_path: string;
  size: number & tags.Type<"uint64">;
  pcln_size: number & tags.Type<"uint64">;
}

export interface FileSymbol {
  name: string;
  addr: number & tags.Type<"uint64">;
  size: number & tags.Type<"uint64">;
  type: "unknown" | "text" | "data";
}

export interface Package {
  name: string;
  type: "main" | "std" | "vendor" | "generated" | "unknown" | "cgo";
  subPackages: {
    [key: string]: Package;
  };
  files: File[];
  symbols: FileSymbol[];
  size: number & tags.Type<"uint64">;
}

export interface Result {
  name: string;
  size: number & tags.Type<"uint64">;
  packages: {
    [key: string]: Package;
  };
  sections: Section[];
  analyzers: (("dwarf" | "disasm" | "symbol" | "pclntab")[]) | undefined;
}

export const parseResult = (() => {
  const _io0 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.size && __typia_transform__isTypeUint64._isTypeUint64(input.size)) && ("object" === typeof input.packages && null !== input.packages && false === Array.isArray(input.packages) && _io1(input.packages)) && (Array.isArray(input.sections) && input.sections.every((elem: any) => "object" === typeof elem && null !== elem && _io6(elem))) && (undefined === input.analyzers || Array.isArray(input.analyzers) && input.analyzers.every((elem: any) => "symbol" === elem || "dwarf" === elem || "disasm" === elem || "pclntab" === elem));
  const _io1 = (input: any): boolean => Object.keys(input).every((key: any) => {
    const value = input[key];
    if (undefined === value)
      return true;
    return "object" === typeof value && null !== value && _io2(value);
  });
  const _io2 = (input: any): boolean => "string" === typeof input.name && ("main" === input.type || "std" === input.type || "vendor" === input.type || "generated" === input.type || "unknown" === input.type || "cgo" === input.type) && ("object" === typeof input.subPackages && null !== input.subPackages && false === Array.isArray(input.subPackages) && _io3(input.subPackages)) && (Array.isArray(input.files) && input.files.every((elem: any) => "object" === typeof elem && null !== elem && _io4(elem))) && (Array.isArray(input.symbols) && input.symbols.every((elem: any) => "object" === typeof elem && null !== elem && _io5(elem))) && ("number" === typeof input.size && __typia_transform__isTypeUint64._isTypeUint64(input.size));
  const _io3 = (input: any): boolean => Object.keys(input).every((key: any) => {
    const value = input[key];
    if (undefined === value)
      return true;
    return "object" === typeof value && null !== value && _io2(value);
  });
  const _io4 = (input: any): boolean => "string" === typeof input.file_path && ("number" === typeof input.size && __typia_transform__isTypeUint64._isTypeUint64(input.size)) && ("number" === typeof input.pcln_size && __typia_transform__isTypeUint64._isTypeUint64(input.pcln_size));
  const _io5 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.addr && __typia_transform__isTypeUint64._isTypeUint64(input.addr)) && ("number" === typeof input.size && __typia_transform__isTypeUint64._isTypeUint64(input.size)) && ("unknown" === input.type || "text" === input.type || "data" === input.type);
  const _io6 = (input: any): boolean => "string" === typeof input.name && ("number" === typeof input.size && __typia_transform__isTypeUint64._isTypeUint64(input.size)) && ("number" === typeof input.file_size && __typia_transform__isTypeUint64._isTypeUint64(input.file_size)) && ("number" === typeof input.known_size && __typia_transform__isTypeUint64._isTypeUint64(input.known_size)) && ("number" === typeof input.offset && __typia_transform__isTypeUint64._isTypeUint64(input.offset)) && ("number" === typeof input.end && __typia_transform__isTypeUint64._isTypeUint64(input.end)) && ("number" === typeof input.addr && __typia_transform__isTypeUint64._isTypeUint64(input.addr)) && ("number" === typeof input.addr_end && __typia_transform__isTypeUint64._isTypeUint64(input.addr_end)) && "boolean" === typeof input.only_in_memory && "boolean" === typeof input.debug;
  const __is = (input: any): input is Result => "object" === typeof input && null !== input && _io0(input);
  return (input: string): import("typia").Primitive<Result> | null => {
    input = JSON.parse(input);
    return __is(input) ? input as any : null;
  };
})();
