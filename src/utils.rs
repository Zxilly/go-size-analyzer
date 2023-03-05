use goblin::elf::Elf;
use goblin::Object;
use std::fs;
use std::path::Path;
use std::process::exit;

pub(crate) fn require_binary(binary: &str) -> Vec<u8> {
    let path = Path::new(binary);
    if !path.exists() {
        eprintln!("The binary {} does not exist", binary);
        exit(1);
    }

    if !path.is_file() {
        eprintln!("The binary {} is not a file", binary);
        exit(1);
    }

    let buffer = fs::read(path).unwrap();

    println!("binary size: {}", pretty_print_size(buffer.len() as u64));

    buffer
}

pub(crate) fn require_elf_64(buffer: &[u8]) -> Elf {
    let object = Object::parse(buffer).unwrap();
    match object {
        Object::Elf(elf) => {
            if elf.is_lib {
                eprintln!("The binary is a shared library");
                exit(1);
            }
            if !elf.is_64 {
                eprintln!("The binary is not a 64-bit binary");
                exit(1);
            }
            elf
        }
        _ => {
            eprintln!("The binary is not a ELF file");
            exit(1);
        }
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
