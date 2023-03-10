use std::ffi::CStr;
use std::path::Path;
use regex::Regex;
use crate::artifact::Artifacts;

extern "C" {
    pub fn runBloaty(filename: *const ::std::os::raw::c_char) -> *const ::std::os::raw::c_char;
}

pub(crate) fn execute(path: &Path, go_packages: Vec<String>, artifacts: &mut Artifacts) {
    let raw = execute_bloaty(path);
    parse_bloaty_result(raw, go_packages, artifacts);
}

fn execute_bloaty(path: &Path) -> String {
    let absolute_path = path.canonicalize().unwrap();
    let path = absolute_path.to_str().unwrap();
    let path_cstr = std::ffi::CString::new(path).unwrap();
    let r: String;

    unsafe {
        let ret = runBloaty(path_cstr.as_ptr());
        let output = CStr::from_ptr(ret).to_str().unwrap();
        r = output.to_string();
    }
    r
}

type CSVItem = (String, u64, u64);

fn parse_bloaty_result(output: String, go_packages: Vec<String>, artifacts: &mut Artifacts) {
    let go_packages_ref = &go_packages;

    let records = csv::ReaderBuilder::new()
        .has_headers(true)
        .from_reader(output.as_bytes())
        .into_deserialize();

    for record in records {
        let item: CSVItem = record.unwrap();
        let (name, _, size) = item;
        if size == 0 {
            continue;
        }

        if name.starts_with("[section") {
            let mut section_name = strip_section_name(name.as_str());
            if section_name.starts_with(".debug") {
                section_name = "Debug Section"
            }
            artifacts.add(section_name.to_string(), size);
        } else {
            let name = name.replace("%2e", "."); // for gc rename

            let mut found = false;
            for package in go_packages_ref.iter() {
                let package = package.to_string();
                if name.starts_with(package.clone().as_str()) {
                    artifacts.add(package, size);
                    found = true;
                    break;
                }
            }
            if !found {
                artifacts.add(get_group_name(name.as_str()).to_string(), size);
            }
        }
    }
}

// [section .rodata] to .rodata
fn strip_section_name(s: &str) -> &str {
    let section_re: Regex = Regex::new(r"\[section\s(.+)]").unwrap();
    section_re.captures(s).unwrap().get(1).unwrap().as_str()
}

fn get_group_name(s: &str) -> &str {
    if s.starts_with("go:itab") {
        return "Go Interface Table";
    }
    if s.starts_with("type:") {
        return "Go Type";
    }
    if s.starts_with("$") {
        return "Temporary variable";
    }
    if s.contains("_cgo_") {
        return "Cgo related";
    }
    if s.starts_with("go:") {
        return "Go struct";
    }
    if s.starts_with("_Z") {
        return "C++";
    }
    "C"
}