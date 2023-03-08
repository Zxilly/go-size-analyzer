use std::fs::{File, Permissions};
use std::io::Write;
use std::os::unix::fs::PermissionsExt;
use tempfile::tempfile;

const BLOATY_BYTES: &[u8] = include_bytes!("bloaty");

fn create_bloaty_file() -> File {
    let mut tmp_bloaty = tempfile()?;
    tmp_bloaty.set_permissions(Permissions::from_mode(0o755))?;
    tmp_bloaty.write_all(BLOATY_BYTES)?;
    return tmp_bloaty;
}

fn execute(path: &str) {
    let bloaty = create_bloaty_file();
}