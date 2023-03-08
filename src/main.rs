mod cli;
mod utils;
mod go;
mod parse;
mod object;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}

