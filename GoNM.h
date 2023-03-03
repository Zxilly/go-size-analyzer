#ifndef GO_SIZE_ANALYSIS_GONM_H
#define GO_SIZE_ANALYSIS_GONM_H

#include <string>

namespace gsv {

    class GoNM {
    private:
        static bool check_golang_toolchain();
    public:
        GoNM();
        void execute(std::string binary);
    };

} // gsv

#endif //GO_SIZE_ANALYSIS_GONM_H
