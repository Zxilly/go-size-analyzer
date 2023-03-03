#include <iostream>
#include "GoNM.h"

int main(int argc, char **argv) {
    if (argc < 2) {
        std::cerr << "Usage: " << argv[0] << " <binary>" << std::endl;
        return 1;
    }
    auto binary = std::string(argv[1]);

    gsv::GoNM go;

    go.execute(binary);
    return 0;
}
