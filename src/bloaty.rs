use std::ffi::CStr;
use std::path::Path;
use crate::artifact::Packages;

extern "C" {
    pub fn runBloaty(filename: *const ::std::os::raw::c_char) -> *const ::std::os::raw::c_char;
}

pub(crate) fn execute(path: &Path, artifacts: &mut Packages) {
    let raw = execute_bloaty(path);
    parse_bloaty_result(raw, artifacts);
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

fn parse_bloaty_result(output: String, packages: &mut Packages) {

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

        packages.add(name, size);
    }
}



