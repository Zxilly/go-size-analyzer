pub(crate) fn require_linux() {
    if !cfg!(linux) {
        panic!("This program only supports Linux");
    }
}