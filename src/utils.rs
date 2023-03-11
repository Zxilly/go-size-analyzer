use object::{Architecture, File, Object};
use std::path::Path;
use std::process::exit;

pub(crate) fn require_file(path: &Path) {
    if !path.exists() {
        eprintln!("The binary {} does not exist", path.to_str().unwrap());
        exit(1);
    }

    if !path.is_file() {
        eprintln!("The binary {} is not a file", path.to_str().unwrap());
        exit(1);
    }
}

pub(crate) fn check_file(file: &File) {
    if !file.is_little_endian() {
        eprintln!("The binary is not little endian");
        exit(1);
    }

    if file.architecture() != Architecture::X86_64 {
        eprintln!("The binary is not x86_64");
        exit(1);
    }

    if !file.is_64() {
        eprintln!("The binary is not 64-bit");
        exit(1);
    }

    if file.format() != object::BinaryFormat::Elf {
        eprintln!("The binary is not ELF");
        exit(1);
    }

    if !file.has_debug_symbols() {
        eprintln!("The binary does not have debug symbols");
        exit(1);
    }
}

pub(crate) fn pretty_print_size(size: u64) -> String {
    let mut size = size as f64;
    let mut unit = "B";
    if size > 1024.0 {
        size /= 1024.0;
        unit = "KB";
    }
    if size > 1024.0 {
        size /= 1024.0;
        unit = "MB";
    }
    if size > 1024.0 {
        size /= 1024.0;
        unit = "GB";
    }
    format!("{:.2}{}", size, unit)
}
