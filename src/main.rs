mod go;
mod cli;
mod utils;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}
