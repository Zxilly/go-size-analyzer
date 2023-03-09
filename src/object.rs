use std::collections::HashMap;

#[derive(Debug,Copy, Clone)]
pub(crate) struct Targets {
    inner: HashMap<String, Target>,
}

impl Targets {
    pub(crate) fn new() -> Self {
        Targets {
            inner: HashMap::new(),
        }
    }

    pub(crate) fn add(&mut self, package: String, section_name: String, size: u64) {
        let entry = self
            .inner
            .entry(package.clone())
            .or_insert(Target::new(package));
        entry.add(section_name, size);
    }
}

#[derive(Debug,Copy, Clone)]
pub(crate) struct Target {
    package: String,
    size: TargetSize,
}

impl Target {
    fn new(package: String) -> Self {
        Target {
            package,
            size: TargetSize::new(),
        }
    }

    fn add(&mut self, section_name: String, size: u64) {
        self.size.add(section_name, size);
    }
}

#[derive(Debug,Copy, Clone)]
pub(crate) struct TargetSize {
    inner: HashMap<String, u64>,
}

impl TargetSize {
    pub(crate) fn new() -> Self {
        TargetSize {
            inner: HashMap::new(),
        }
    }

    pub(crate) fn add(&mut self, section_name: String, size: u64) {
        let entry = self.inner.entry(section_name).or_insert(0);
        *entry += size;
    }
}
