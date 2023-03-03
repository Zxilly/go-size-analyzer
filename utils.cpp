#include "utils.h"

namespace gsv {
    void fatal(const char *msg) {
        std::cerr << msg << std::endl;
        exit(1);
    }
}