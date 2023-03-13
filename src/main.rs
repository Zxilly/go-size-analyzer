mod cli;
mod utils;
mod go;
mod artifact;
mod bloaty;
mod web;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}

