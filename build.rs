fn main() {
    // target os
    let target_os = std::env::var("CARGO_CFG_TARGET_OS").unwrap();
    let target_arch = std::env::var("CARGO_CFG_TARGET_ARCH").unwrap();

    if target_os != "linux" || target_arch != "x86_64" {
        panic!("This program only supports Linux x86_64");
    }
}