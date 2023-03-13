use std::collections::HashSet;
use std::path::Path;
use crate::utils::{check_file, require_file};
use crate::{bloaty, go, web};
use clap::Parser;
use crate::artifact::Packages;

/// Analysis golang compiled binary size
#[derive(Parser)]
pub(crate) struct Cli {
    /// The port to listen on for web mode
    #[arg(
    short,
    long,
    value_parser = clap::value_parser ! (u16).range(1..65535),
    default_value = "8888")]
    pub(crate) port: u16,


    /// View the result in the browser
    #[arg(short, long)]
    pub(crate) web: bool,

    /// The binary to analysis
    #[arg(name = "BINARY", required = true)]
    pub(crate) binary: String,
}

impl Cli {
    pub(crate) fn new() -> Self {
        Cli::parse()
    }

    pub(crate) fn execute(&self) {
        println!("Analyzing binary: {}", self.binary);

        let (binary, go_packages) = self.pre_check();
        let mut packages = Packages::new(go_packages);
        bloaty::execute(binary, &mut packages);

        if !self.web {
            println!("{}", packages);
        } else {
            web::start(self.port, packages);
        }
    }

    fn pre_check(&self) -> (&Path, HashSet<String>) {
        let path = Path::new(&self.binary);

        require_file(path);

        let buffer = std::fs::read(path).unwrap();
        let file = object::File::parse(&*buffer).unwrap();
        check_file(&file);
        let go_packages = go::parse_go_packages(&file);

        (path, go_packages)
    }
}
