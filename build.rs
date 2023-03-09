use std::fs;
use std::process::Command;

fn main() {
    check_tools();
    git_checkout();
    bloaty_patch();
    cmake_build();
}

fn check_tools() {
    // check if git in path
    ["git", "ninja"]
        .iter()
        .for_each(|tool| check_tool(tool));
}

fn git_checkout() {
    Command::new("git")
        .args(["submodule", "update", "--init", "--recursive"])
        .output()
        .expect("failed to update git submodules");

    // git submodule foreach --recursive git reset --hard HEAD
    Command::new("git")
        .args([
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

fn bloaty_patch() {
    // copy file in patch/bloaty to third_party/bloaty

    fs::copy(
        "patch/bloaty/CMakeLists.txt",
        "third_party/bloaty/CMakeLists.txt",
    ).expect("failed to copy bloaty CMakeLists.txt");
    fs::copy(
        "patch/bloaty/lib.cc",
        "third_party/bloaty/src/lib.cc",
    ).expect("failed to copy bloaty lib.cc");
}

// find if a tool is in path
fn check_tool(name: &str) {
    Command::new(name)
        .arg("--version")
        .output()
        .unwrap_or_else(|_| panic!("{} not found in path", name));
}

fn cmake_build() {}