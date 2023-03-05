use crate::utils::{require_binary, require_elf_64};
use clap::Parser;
use crate::elf::parse_elf;

/// Analysis golang compiled binary size
#[derive(Parser)]
pub(crate) struct Cli {
    /// The port to listen on
    #[arg(
    short,
    long,
    value_parser = clap::value_parser!(u16).range(1..65535),
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
        let buffer = require_binary(&self.binary);
        let elf = require_elf_64(&buffer);
        parse_elf(elf);
    }
}
