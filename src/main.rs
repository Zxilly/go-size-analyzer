mod cli;
mod utils;
mod go;
mod object;
mod bloaty;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}

