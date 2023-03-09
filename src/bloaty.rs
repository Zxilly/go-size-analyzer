use crate::object::Targets;
use std::fs::{File, Permissions};
use std::io::Write;
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use tempfile::tempfile;
use regex::Regex;

// let output = std::process::Command::new(bloaty)
// .args(["-w", "-n", "0", "-d", "symbols", "--csv"])
// .arg(full_path)
// .output()
// .expect("failed to execute bloaty process");

// [section .rodata] to .rodata
fn parse_section_name(s: &str) -> &str {
    let section_re: Regex = Regex::new(r"\[section\s(\.\w+)]").unwrap();
    section_re.captures(s).unwrap().get(1).unwrap().as_str()
}