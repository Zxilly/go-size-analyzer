use cpp_demangle::Symbol;
use std::string::ToString;

const CGO_OR_RUNTIME: &str = "cgo or runtime";
const RAW_SECTION: &str = "raw section";

#[derive(Debug, Clone)]
pub(crate) struct GoSymbol {
    // in hexadecimal
    pub(crate) address: String,
    // T	text (code) segment symbol
    // t	static text segment symbol
    // R	read-only data segment symbol
    // r	static read-only data segment symbol
    // D	data segment symbol
    // d	static data segment symbol
    // B	bss segment symbol
    // b	static bss segment symbol
    // C	constant address
    // U	referenced but undefined symbol
    pub(crate) symbol_type: String,
    // name of symbol
    pub(crate) symbol_name: String,
    // The package symbol belongs to
    pub(crate) package: String,
    // size of symbol
    pub(crate) size: i64,
}

impl GoSymbol {
    pub(crate) fn parse(s: &str) -> Option<Self> {
        let mut parts = s.split_whitespace();

        if parts.clone().count() < 4 {
            let size = parts.next().unwrap();
            let symbol_type = parts.next().unwrap();
            let name = parts.next().unwrap();
            if symbol_type != "U" {
                eprintln!("Not U but have no address, {}", s);
                return None;
            }

            return Some(GoSymbol {
                address: "".to_string(),
                symbol_type: symbol_type.to_string(),
                symbol_name: name.to_string(),
                package: CGO_OR_RUNTIME.to_string(),
                size: size.parse().unwrap(),
            });
        }

        let address = parts.next().unwrap();
        let size = parts.next().unwrap();
        let symbol_type = parts.next().unwrap();
        let name = parts.next().unwrap();

        if name.starts_with('.') {
            // Looks like a section
            return Some(
                GoSymbol {
                    address: address.to_string(),
                    symbol_type: symbol_type.to_string(),
                    symbol_name: name.to_string(),
                    package: RAW_SECTION.to_string(),
                    size: size.parse().unwrap(),
                }
            )
        }

        let parsed_name = parse_symbol(name);
        Some(
            GoSymbol {
                address: address.to_string(),
                symbol_type: symbol_type.to_string(),
                symbol_name: parsed_name.0,
                package: parsed_name.1,
                size: size.parse().unwrap(),
            }
        )
    }
}

fn parse_cpp_symbol(s: &str) -> Result<String, String> {
    let sym = Symbol::new(s);

    if sym.is_err() {
        return Err("Could not parse cpp symbol".to_string());
    }
    let sym = sym.unwrap();

    Ok(sym.to_string())
}

fn parse_go_package(s: &str) -> Result<String, String> {
    let mut parts = s.split('/');

    if parts.clone().count() < 4 {
        return Err("Could not parse go package".to_string());
    }

    let domain = parts.next().unwrap();
    let owner = parts.next().unwrap();
    let repo = parts.next().unwrap();

    let pkg = format!("{}/{}/{}", domain, owner, repo);

    Ok(pkg)
}

fn parse_symbol(s: &str) -> (String, String) {
    let cpp_symbol = parse_cpp_symbol(s);
    if cpp_symbol.is_ok() {
        return (s.to_string(), CGO_OR_RUNTIME.to_string());
    }
    let go_package = parse_go_package(s);
    if let Ok(..) = go_package {
        return (s.to_string(), go_package.unwrap());
    }
    (s.to_string(), CGO_OR_RUNTIME.to_string())
}