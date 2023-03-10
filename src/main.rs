mod cli;
mod utils;
mod go;
mod artifact;
mod bloaty;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}

