use std::path::Path;
use crate::utils::{check_file, require_binary};
use crate::{bloaty, go};
use clap::Parser;
use crate::artifact::Artifacts;

/// Analysis golang compiled binary size
#[derive(Parser)]
pub(crate) struct Cli {
    /// The port to listen on
    #[arg(
    short,
    long,
    value_parser = clap::value_parser ! (u16).range(1..65535),
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
        let mut artifacts = Artifacts::new();
        let (binary, go_packages) = self.prepare();
        bloaty::execute(binary, go_packages, &mut artifacts);
        println!("{}", artifacts);
    }

    fn prepare(&self) -> (&Path, Vec<String>) {
        let path = Path::new(&self.binary);

        require_binary(path);

        let buffer = std::fs::read(path).unwrap();
        let file = object::File::parse(&*buffer).unwrap();
        check_file(&file);
        let go_packages = go::parse_go_packages(&file);

        (path, go_packages)
    }
}
