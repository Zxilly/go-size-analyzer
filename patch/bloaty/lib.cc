#include <sstream>
#include "bloaty.h"
#include "bloaty.pb.h"

std::string run(bloaty::Options options) {
    bloaty::RollupOutput output;
    bloaty::MmapInputFileFactory mmap_factory;
    std::string error;
    if (!bloaty::BloatyMain(options, mmap_factory, &output, &error)) {
        if (!error.empty()) {
            fprintf(stderr, "bloaty: %s\n", error.c_str());
        }
    }
    // print to a string
    std::stringstream output_stream;

    bloaty::OutputOptions output_options;

    if (!options.dump_raw_map()) {
        output.Print(output_options, &output_stream);
    }

    return output_stream.str();
}