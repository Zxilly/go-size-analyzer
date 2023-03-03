#include "GoNM.h"
#include "utils.h"
#include <iostream>

namespace gsv {
    GoNM::GoNM() {
        if (!check_golang_toolchain()) {
            fatal("golang toolchain not found");
        }
    }

    bool GoNM::check_golang_toolchain() {
        std::string cmd = "go version";
        std::string output;

#ifdef _WIN32
        cmd = "cmd /c " + cmd;
#else
        cmd = "/bin/sh -c " + cmd;
#endif

        auto fp = popen(cmd.c_str(), "r");
        if (fp == nullptr) {
            return false;
        }

        char buf[1024];
        while (fgets(buf, sizeof(buf), fp)) {
            output += buf;
        }
        pclose(fp);

        if (output.find("go version") != std::string::npos) {
            return true;
        }
        return false;
    }

    void GoNM::execute(const std::string& binary) {

    }


} // gsv