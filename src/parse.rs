use crate::object::Targets;
use object::{File, Object, ObjectSection, ObjectSymbol, SectionIndex};

pub(crate) fn parse_symbols(targets: &mut Targets, obj: &File, go_packages: &Vec<String>) {
    let symbols = obj.symbols();


    for symbol in symbols {
        if symbol.section_index().is_none() || symbol.section_index().unwrap() == SectionIndex(0) {
            continue;
        }

        let section_index = symbol.section_index().unwrap();
        let raw_name = symbol.name().unwrap();


        // for gc
        let raw_name = raw_name.replace("%2e", ".");
        let name = raw_name.as_str();

        let size = symbol.size();
        if size == 0 {
            continue;
        }

        let section = obj.section_by_index(section_index).unwrap();
        let section_name = section.name().unwrap();

        if section_name == ".bss" || section_name == ".noptrbss" {
            continue;
        }

        let mut namespace = "";

        if name.starts_with("_Z") {
            let demanged = cpp_demangle::Symbol::new(raw_name);
            if demanged.is_err() {
                continue;
            } else {
                namespace = "C++";
            }
        } else {
            for package in go_packages {
                if name.starts_with(package) {
                    if package.starts_with("runtime") {
                        namespace = "Go runtime"
                    } else {
                        namespace = package;
                    }
                    break;
                }
            }

            if namespace == "" {
                if name.starts_with("go:itab") {
                    namespace = "Go itab"
                } else if name.starts_with("type:") {
                    namespace = "Go type"
                } else if name.starts_with("$") && section_name == ".rodata" {
                    namespace = "Go tmp var"
                } else if name.contains("_cgo") {
                    namespace = "Go cgo"
                } else if name.starts_with("go:") {
                    namespace = "Go runtime"
                } else if name.contains("/") {
                    // maybe golang structure TODO: dfs dwarf to check all go package
                    if name.contains(".") {
                        let mut parts = name.split(".");
                        let package = parts.next().unwrap();
                        namespace = package;
                    } else {
                        namespace = name;
                    }
                } else {
                    namespace = "C";
                }
            }
        }

        targets.add(namespace.to_string(), section_name.to_string(), size);
    }
}