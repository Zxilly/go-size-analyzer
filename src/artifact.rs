use std::collections::{BTreeMap, HashSet};
use std::fmt::{Display, Formatter};
use std::hash::Hash;
use clap::__macro_refs::once_cell::sync::Lazy;
use regex::Regex;
use crate::utils::pretty_print_size;

static SECTION_RE: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"\[section\s(.+)]").unwrap()
});

const SECTION_PREFIX: &str = "[section";
const GO_ITAB_PREFIX: &str = "go:itab";
const GO_TYPE_PREFIX: &str = "type:";
const TEMP_VAR_PREFIX: &str = "$";
const GO_PREFIX: &str = "go:";
const CPP_SEPARATOR: &str = "::";
const CGO_SYMBOL: &str = "_cgo_";

#[derive(Debug, Clone)]
pub(crate) struct Packages {
    inner: BTreeMap<String, Package>,

    go_packages: HashSet<String>,
}

impl Packages {
    pub(crate) fn new(go_packages: HashSet<String>) -> Self {
        Packages {
            inner: BTreeMap::new(), // symbol -> Artifact
            go_packages,
        }
    }

    pub(crate) fn add(&mut self, symbol: String, size: u64) {
        let (package, kind) = self.parse_symbol(symbol.clone());
        let entry = self
            .inner
            .entry(package.clone())
            .or_insert(Package::new(package, kind));
        entry.add(symbol, size);
    }


    fn parse_symbol(&self, symbol: String) -> (String, ArtifactType) {
        let package;
        let kind;

        if symbol.starts_with(SECTION_PREFIX) {
            package = Packages::strip_section_name(&symbol).to_string();
            kind = ArtifactType::Section;
        } else if let Some(go_package) = self.try_get_go_package(symbol.clone()) {
            package = go_package;
            kind = ArtifactType::Go;
        } else if symbol.starts_with(GO_ITAB_PREFIX) {
            package = "Go Interface".to_string();
            kind = ArtifactType::Go;
        } else if symbol.starts_with(GO_TYPE_PREFIX) {
            package = "Go Type".to_string();
            kind = ArtifactType::Go;
        } else if symbol.starts_with(TEMP_VAR_PREFIX) {
            package = "Temp Var".to_string();
            kind = ArtifactType::Unknown;
        } else if symbol.contains(CGO_SYMBOL) {
            package = "Cgo related".to_string();
            kind = ArtifactType::Go;
        } else if symbol.starts_with(GO_PREFIX) {
            package = "Go struct".to_string();
            kind = ArtifactType::Go;
        } else if symbol.contains(CPP_SEPARATOR) {
            package = symbol.split(CPP_SEPARATOR).next().unwrap().to_string();
            kind = ArtifactType::Cpp;
        } else {
            package = "C".to_string();
            kind = ArtifactType::C;
        }

        (package, kind)
    }

    // [section .rodata] to .rodata
    fn strip_section_name(s: &str) -> &str {
        SECTION_RE.captures(s).unwrap().get(1).unwrap().as_str()
    }

    fn try_get_go_package(&self, symbol: String) -> Option<String> {
        for package in self.go_packages.iter() {
            if symbol.starts_with(package.clone().as_str()) {
                return Some(package.clone());
            }
        }
        None
    }
}


impl Display for Packages {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let mut total = 0;
        let mut packages: Vec<(String, u64)> = self.inner.values().map(|v| (v.name.clone(), v.size)).collect();
        packages.sort_by(|a, b| b.1.cmp(&a.1));

        // name width
        let mut name_width = 0;
        for (package, _) in packages.iter() {
            if package.len() > name_width {
                name_width = package.len();
            }
        }
        name_width += 2;

        for (package, size) in packages {
            writeln!(f, "{:width$}: {}", package, pretty_print_size(size), width = name_width)?;
            total += size;
        }
        writeln!(f, "{:width$}: {}", "Total", pretty_print_size(total), width = name_width)
    }
}

#[derive(Debug, Clone, Eq, Hash, PartialEq)]
pub(crate) enum ArtifactType {
    Go,
    C,
    Cpp,
    Section,
    Unknown,
}

#[derive(Debug, Clone)]
pub(crate) struct Package {
    pub(crate) symbols: Vec<Symbol>,
    pub(crate) size: u64,
    pub(crate) kind: ArtifactType,
    pub(crate) name: String,
}

type Symbol = (String, u64); // symbol, size

impl Package {
    fn new(name: String, kind: ArtifactType) -> Self {
        Package {
            name,
            kind,
            symbols: Vec::new(),
            size: 0,
        }
    }

    fn add(&mut self, symbol: String, size: u64) {
        self.size += size;
        self.symbols.push((symbol, size));
    }

}
