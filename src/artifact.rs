use std::collections::{BTreeMap, HashSet};
use std::fmt::{Display, Formatter};
use std::hash::Hash;
use once_cell::sync::Lazy;
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
            if package.starts_with("runtime") || package.starts_with("vendor") {
                kind = ArtifactType::GoRuntime;
            } else {
                kind = ArtifactType::Go;
            }
        } else if symbol.starts_with(GO_ITAB_PREFIX) {
            package = "Go Interface".to_string();
            kind = ArtifactType::GoRuntime;
        } else if symbol.starts_with(GO_TYPE_PREFIX) {
            package = "Go Type".to_string();
            kind = ArtifactType::GoRuntime;
        } else if symbol.starts_with(TEMP_VAR_PREFIX) {
            package = "Temp Var".to_string();
            kind = ArtifactType::Unknown;
        } else if symbol.contains(CGO_SYMBOL) {
            package = "Cgo related".to_string();
            kind = ArtifactType::GoRuntime;
        } else if symbol.starts_with(GO_PREFIX) {
            package = "Go struct".to_string();
            kind = ArtifactType::GoRuntime;
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
        let unescaped_symbol = symbol.replace("%2e", ".");

        for package in self.go_packages.iter() {
            if unescaped_symbol.starts_with(package.clone().as_str()) {
                return Some(package.clone());
            }
        }
        None
    }
}

impl Packages {
    /// Convert to csv format
    /// display_name, relation, kind, size
    pub(crate) fn into_csv(self) -> String {
        let mut small_packages: BTreeMap<String, PackageCsv> = BTreeMap::new();

        for (_, package) in self.inner {
            for symbol in package.symbols {
                let pcsv = symbol.clone().into_package();
                let flattened = pcsv.flatten();

                for i in &flattened {
                    let key = i.id.clone();

                    small_packages.entry(key).and_modify(|e| {
                        e.size += i.size;
                    }).or_insert(i.clone());
                }
            }
        }

        let mut csv = csv::WriterBuilder::new()
            .has_headers(true)
            .from_writer(vec![]);
        csv.write_record(["display_name", "id", "parent_id", "kind", "size"]).unwrap();
        for (_, v) in small_packages {
            csv.write_record(&[
                v.display_name,
                v.id,
                v.parent_id,
                v.kind.to_string(),
                v.size.to_string(),
            ]).unwrap();
        }
        let bytes = csv.into_inner().unwrap();
        String::from_utf8(bytes).unwrap()
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

#[derive(Debug, Clone, Eq, Hash, PartialEq, Ord, PartialOrd)]
pub(crate) enum ArtifactType {
    Go,
    GoRuntime,
    C,
    Cpp,
    Section,
    Unknown,
}

impl Display for ArtifactType {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            ArtifactType::Go => write!(f, "Go"),
            ArtifactType::GoRuntime => write!(f, "Go Runtime"),
            ArtifactType::C => write!(f, "C"),
            ArtifactType::Cpp => write!(f, "C++"),
            ArtifactType::Section => write!(f, "Section"),
            ArtifactType::Unknown => write!(f, "Unknown"),
        }
    }
}

impl ArtifactType {
    fn separator(&self) -> &str {
        match self {
            ArtifactType::Go => "/",
            ArtifactType::Cpp => "::",
            _ => "",
        }
    }
}

const RELATION_SEPARATOR: &str = "->";
const TREE_ROOT: &str = "ROOT";

impl ArtifactType {
    /// Parse symbol to package and relation, not strip subpackage
    pub(crate) fn parse_symbol(&self, symbol: String) -> (String, String, String) {
        match self {
            ArtifactType::Go => ArtifactType::parse_go_symbol(symbol),
            ArtifactType::Cpp => ArtifactType::parse_cpp_symbol(symbol),
            _ => {
                let package = self.to_string();
                let id = vec![TREE_ROOT.to_string(), package.clone()].join(RELATION_SEPARATOR);
                let parent = TREE_ROOT.to_string();
                (package, id, parent)
            }
        }
    }

    fn parse_go_symbol(symbol: String) -> (String, String, String) {
        let clean = |s: &str| s.replace("%2e", ".");

        let single_package_name = !symbol.contains('/');

        let mut parts = symbol.split('/');
        let mut package_parts = Vec::new();

        if !single_package_name {
            package_parts.push(clean(parts.next().unwrap()))
        }

        for part in parts {
            if !part.contains('.') {
                package_parts.push(clean(part));
                continue;
            }
            let sub = part.split('.').next().unwrap();
            package_parts.push(clean(sub));
            break;
        };

        let package = package_parts.join("/");

        package_parts.insert(0, TREE_ROOT.to_string());
        let id = package_parts.join(RELATION_SEPARATOR);
        package_parts.pop();
        let parent = package_parts.join(RELATION_SEPARATOR);

        (package, id, parent)
    }

    fn parse_cpp_symbol(symbol: String) -> (String, String, String) {
        let mut all_namespace = symbol.split("::").collect::<Vec<&str>>();
        all_namespace.pop();
        let mut parts: Vec<String> = all_namespace.iter().map(|s| s.to_string()).collect();
        let namespace = parts.join(RELATION_SEPARATOR);

        parts.insert(0, TREE_ROOT.to_string());
        let id = parts.join(RELATION_SEPARATOR);
        parts.pop();
        let parent = parts.join(RELATION_SEPARATOR);

        (namespace, id, parent)
    }
}

#[derive(Debug, Clone)]
pub(crate) struct Package {
    pub(crate) symbols: Vec<Symbol>,
    pub(crate) size: u64,
    pub(crate) kind: ArtifactType,
    pub(crate) name: String,
}


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
        self.symbols.push(
            Symbol {
                value: symbol,
                size,
                kind: self.kind.clone(),
            });
    }
}

#[derive(Debug, Clone)]
pub(crate) struct PackageCsv {
    display_name: String,
    id: String,
    parent_id: String,
    kind: ArtifactType,
    size: u64,
}

impl PackageCsv {
    fn new(display_name: String, id: String, parent_id: String, kind: ArtifactType, size: u64) -> Self {
        PackageCsv {
            display_name,
            id,
            parent_id,
            kind,
            size,
        }
    }

    /// provide all parent node
    fn flatten(&self) -> Vec<PackageCsv> {
        let separator = self.kind.separator();

        let mut flattened = Vec::new();

        let mut parts = self.id.split(RELATION_SEPARATOR).collect::<Vec<&str>>();
        while parts.len() >= 2 {
            let display_name = &parts[1..].join(separator);
            let id = parts.join(RELATION_SEPARATOR);

            parts.pop();
            let parent_id = parts.join(RELATION_SEPARATOR);
            flattened.push(PackageCsv::new(
                display_name.to_string(),
                id,
                parent_id,
                self.kind.clone(),
                self.size,
            ))
        }
        flattened.push(PackageCsv::new(
            "".to_string(),
            TREE_ROOT.to_string(),
            "".to_string(),
            ArtifactType::Unknown,
            self.size,
        ));

        flattened
    }
}

#[derive(Debug, Clone)]
pub(crate) struct Symbol {
    pub(crate) value: String,
    pub(crate) size: u64,
    pub(crate) kind: ArtifactType,
}

impl Symbol {
    /// display_name, id, parent_id, kind, size
    pub(crate) fn into_package(self) -> PackageCsv {
        let (display, id, parent_id) = self.kind.parse_symbol(self.value.clone());
        PackageCsv::new(display, id, parent_id, self.kind, self.size)
    }
}