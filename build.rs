use std::process::Command;

fn main() {
    // check if git in path
    ["git", "cmake", "ninja"]
        .iter()
        .for_each(|tool| check_tool(tool));

    // ensure all git submodules are checked
    Command::new("git")
        .args(&["submodule", "update", "--init", "--recursive"])
        .output()
        .expect("failed to update git submodules");

    // git submodule foreach --recursive git reset --hard HEAD
    Command::new("git")
        .args(&[
            "submodule",
            "foreach",
            "--recursive",
            "git",
            "reset",
            "--hard",
            "HEAD",
        ])
        .output()
        .expect("failed to reset git submodules");
}

// find if a tool is in path
fn check_tool(name: &str) {
    Command::new(name)
        .arg("--version")
        .output()
        .expect("failed to check tool " + name);
}
