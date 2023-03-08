use crate::go::parse_go_packages;
use crate::object::Targets;
use crate::parse::parse_symbols;
use crate::utils::{check_file, pretty_print_size, require_binary};
use clap::Parser;
use object::{Object, ObjectSection, Section};
use std::collections::HashMap;

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
        let binary = require_binary(&self.binary);

        let buffer = std::fs::read(binary).unwrap();
        let file = object::File::parse(&*buffer).unwrap();

        check_file(&file);

        let mut targets = Targets::new();

        let packages = parse_go_packages(&file);

        parse_symbols(&mut targets, &file, &packages);
    }
}
