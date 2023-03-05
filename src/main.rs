mod cli;
mod utils;
mod elf;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}
