use std::path::Path;
use crate::utils::{check_file, pretty_print_size, require_binary};
use crate::{bloaty, go};
use clap::Parser;

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
        self.prepare();

        let (binary, go_packages) = self.prepare();

    }

    fn prepare(&self) -> (Box<&Path>,Vec<String>) {
        let binary = *require_binary(&self.binary);
        let buffer = std::fs::read(binary.clone()).unwrap();
        let file = object::File::parse(&*buffer).unwrap();
        check_file(&file);
        let go_packages = go::parse_go_packages(&file);

        return (Box::from(binary), go_packages);
    }
}
