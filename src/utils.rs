use cpp_demangle::Symbol;

pub(crate) fn symbol_name_parse(s:String) -> String {
    let sym = Symbol::new(&s)
        .expect("failed to parse symbol name");
    sym.to_string()
}