use std::collections::HashMap;
use std::fmt::{Display, Formatter};
use crate::utils::pretty_print_size;

#[derive(Debug, Clone)]
pub(crate) struct Artifacts {
    inner: HashMap<String, Artifact>,
}

impl Artifacts {
    pub(crate) fn new() -> Self {
        Artifacts {
            inner: HashMap::new(),
        }
    }

    pub(crate) fn add(&mut self, package: String, size: u64) {
        let entry = self
            .inner
            .entry(package.clone())
            .or_insert(Artifact::new());
        entry.add(size);
    }
}

impl Display for Artifacts {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let mut total = 0;
        let mut packages: Vec<(String, u64)> = self.inner.iter().map(|(k, v)| (k.clone(), v.size)).collect();
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

#[derive(Debug, Clone)]
pub(crate) struct Artifact {
    size: u64,
}

impl Artifact {
    fn new() -> Self {
        Artifact {
            size: 0
        }
    }

    fn add(&mut self, size: u64) {
        self.size += size
    }
}
