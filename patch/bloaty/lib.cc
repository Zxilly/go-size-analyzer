#include <sstream>
#include "bloaty.h"
#include "bloaty.pb.h"

using namespace bloaty;

extern "C" const char* runBloaty(const char* filename) {
    RollupOutput output;
    Options options;
    OutputOptions output_options;

    output_options.output_format = OutputFormat::kCSV;
    options.add_data_source("symbols");
    options.set_demangle(Options::DEMANGLE_SHORT);
    options.set_max_rows_per_level(INT64_MAX);
    options.add_filename(filename);

    bloaty::MmapInputFileFactory mmap_factory;
    std::string error;
    if (!bloaty::BloatyMain(options, mmap_factory, &output, &error)) {
        if (!error.empty()) {
            fprintf(stderr, "bloaty: %s\n", error.c_str());
        }
    }
    // print to a string
    std::stringstream output_stream;

    if (!options.dump_raw_map()) {
        output.Print(output_options, &output_stream);
    }

    char* ret = new char[output_stream.str().length() + 1];
    strcpy(ret, output_stream.str().c_str());

    return ret;
}