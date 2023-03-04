use std::io::stderr;
use std::process::Command;

pub(crate) fn check_golang_toolchain() -> bool {
    let output = Command::new("go")
        .arg("version")
        .output()
        .expect("failed to execute \"go version\", is go installed?");

    let version = String::from_utf8_lossy(&output.stdout);
    version.contains("go version")
}

pub(crate) struct GoSymbol {
    // in hexadecimal
    address: String,
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
    symbolType: String,
    // name of symbol
    name: String,
    // size of symbol
    size: i64,
}

pub(crate) fn execute_go_tool_nm(name: String) -> Result<Vec<GoSymbol>, String> {
    let output = Command::new("go")
        .arg("tool")
        .arg("nm")
        .arg("-size")
        .arg(name)
        .output()
        .expect("failed to execute \"go tool nm\"");

    if !output.status.success() {
        let error = String::from_utf8_lossy(&output.stderr);
        return Err(error.to_string());
    }

    let output_str = String::from_utf8_lossy(&output.stdout);
    if output_str.is_empty() {
        return Err("go tool nm output is empty".to_string());
    }
    if output_str.contains("no symbols") {
        return Err("no symbols found, don't build binary with ldflags \"-s -w\"".to_string());
    }

    let mut symbols = Vec::new();

    let lines = output_str.lines();
    for line in lines {
        let mut parts = line.split_whitespace();

        let address = parts.next().unwrap();
        let size = parts.next().unwrap();
        let symbol_type = parts.next().unwrap();


        let name = parts.next().unwrap();

        let symbol = GoSymbol {
            address: address.to_string(),
            symbolType: symbol_type.to_string(),
            name: name.to_string(),
            size: size.parse().unwrap(),
        };
        symbols.push(symbol);
    }

    return Ok(symbols);
}