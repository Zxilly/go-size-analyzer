use clap::{Parser};
use crate::go;
use crate::go_symbol::GoSymbol;

/// Analysis golang compiled binary size
#[derive(Parser)]
pub(crate) struct Cli {
    /// The port to listen on
    #[arg(
    short,
    long,
    value_parser(clap::value_parser ! (u16).range(1..65535)),
    default_value = "8888")]
    pub(crate) port: u16,

    /// The binary to analysis
    #[arg(name = "BINARY", required = true)]
    pub(crate) binary: String,
}

impl Cli {
    pub(crate) fn new() -> Self {
        Cli::parse()
    }

    pub(crate) fn execute(&self) {
        let go_installed = go::check_golang_toolchain();
        if !go_installed {
            eprintln!("Could not find go in PATH, is go installed?")
        }

        let symbols = go::execute_go_tool_nm(self.binary.clone());
        if symbols.is_err() {
            eprintln!("{}", symbols.err().unwrap());
            return;
        }

        let symbols = symbols.unwrap();
        let packages = go::merge_symbols(symbols);
        // sort by size
        let mut packages = packages;
        packages.sort_by(|a, b| b.size.cmp(&a.size));
        for package in packages {
            // display size in MB or KB, now size is in bytes
            let size_str = if package.size > 1024 * 1024 {
                format!("{}M", package.size / 1024 / 1024)
            } else {
                format!("{}K", package.size / 1024)
            };

            println!("{}: {}", package.name, size_str);
        }
    }
}