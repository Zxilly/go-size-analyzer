use std::collections::HashMap;
use std::process::Command;
use crate::go_symbol::GoSymbol;

pub(crate) fn check_golang_toolchain() -> bool {
    let output = Command::new("go")
        .arg("version")
        .output()
        .expect("failed to execute \"go version\", is go installed?");

    let version = String::from_utf8_lossy(&output.stdout);
    version.contains("go version")
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
        let symbol = GoSymbol::parse(line);
        if symbol.is_none() {
            continue;
        }
        symbols.push(symbol.unwrap());
    }

    Ok(symbols)
}

pub(crate) struct Package {
    pub(crate) name: String,
    pub(crate) size: i64,
}

pub(crate) fn merge_symbols(symbols: Vec<GoSymbol>) -> Vec<Package> {
    let mut symbols_mp = HashMap::new();

    for symbol in symbols {
        let key = symbol.package.clone();
        let size = symbol.size;
        let value = symbols_mp.entry(key).or_insert(0);
        *value += size;
    }

    let mut ret = Vec::new();
    for (key, value) in symbols_mp {
        ret.push(Package {
            name: key,
            size: value,
        });
    }
    ret
}