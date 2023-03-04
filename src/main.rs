mod go;
mod cli;
mod go_symbol;

fn main() {
    let cli = cli::Cli::new();
    cli.execute();
}
