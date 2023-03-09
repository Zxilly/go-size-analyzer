use crate::object::Targets;
use std::fs::{File, Permissions};
use std::io::Write;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use tempfile::tempfile;
use regex::Regex;

const BLOATY_BYTES: &[u8] = include_bytes!("../bloaty");

fn create_bloaty_file() -> File {
    let mut tmp_bloaty = tempfile()?;
    tmp_bloaty.set_permissions(Permissions::from_mode(0o755))?;
    tmp_bloaty.write_all(BLOATY_BYTES)?;
    return tmp_bloaty;
}

pub(crate) fn scan(path: &Path, go_packages: Vec<String>) {
    let bloaty = create_bloaty_file();
    let full_path = path.canonicalize().unwrap().to_str().unwrap();

    let output = std::process::Command::new(bloaty)
        .args(["-w", "-n", "0", "-d", "symbols", "--csv"])
        .arg(full_path)
        .output()
        .expect("failed to execute bloaty process");

    let output = String::from_utf8(output.stdout).unwrap();

    let mut lines = output.lines();
    let header = lines.next().unwrap();

    let mut targets = Targets::new();
    for line in lines {
        let mut coloumns = line.split(',');
        let symbol = coloumns.next().unwrap();
        let filesize = coloumns.nth(1).unwrap().parse::<u64>().unwrap();

        let mut resolved = false;
        if symbol.starts_with("[section"){
            let section_name = parse_section_name(symbol);
            targets.add(section_name.to_string(),section_name.to_string(),filesize);
            resolved = true;
        }

        if !resolved {
            for gp in go_packages{
                if symbol.starts_with(gp) {

                }
            }
        }
    }
}
const SECTION_RE: Regex = Regex::new(r"\[section\s(\.\w+)]").unwrap();

// [section .rodata] to .rodata
fn parse_section_name(s : &str) -> &str {
    SECTION_RE.captures(s).unwrap().get(1).unwrap().as_str()
}