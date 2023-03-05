use crate::utils::pretty_print_size;
use goblin::elf::Elf;

pub(crate) fn parse_elf(elf: Elf) {
    let mut total_size: u64 = 0;

    // for ph in elf.program_headers {
    //     let name = elf.strtab.get_at(ph.p_offset as usize).unwrap_or("");
    //     println!("{} size: {}", name, pretty_print_size(ph.p_filesz as usize));
    //     total_size += ph.p_filesz as usize;
    // }

    // println!("total size: {}", pretty_print_size(total_size));
    let shdr_strtab = elf.shdr_strtab.to_vec().unwrap().len();
    println!("shdr_strtab size: {}", pretty_print_size(shdr_strtab as u64));

    elf.header.e_phentsize

    let strtab_size = elf.strtab.to_vec().unwrap().len();
    println!("strtab size: {}", pretty_print_size(strtab_size as u64));
    let symtab_size = elf.syms.len() * 24;
    println!("symtab size: {}", pretty_print_size(symtab_size as u64));

    for symbol in elf.syms.iter() {
        let name = elf.strtab.get_at(symbol.st_name).unwrap_or("");
        // println!("{} size: {}", name, pretty_print_size(symbol.st_size));
        total_size += symbol.st_size;
    }
    println!("total size: {}", pretty_print_size(total_size));
}
