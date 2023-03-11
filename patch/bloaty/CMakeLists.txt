cmake_minimum_required(VERSION 3.5)
cmake_policy(SET CMP0048 NEW)
if (POLICY CMP0091)
    cmake_policy(SET CMP0091 NEW)
endif ()
project(Bloaty VERSION 1.1)

set(CMAKE_CXX_STANDARD 17)
set_property(GLOBAL PROPERTY USE_FOLDERS ON)

find_package(PkgConfig)

add_definitions(-D_LIBCXXABI_FUNC_VIS=)

set(THREADS_PREFER_PTHREAD_FLAG TRUE)
find_package(Threads REQUIRED)


add_subdirectory(third_party/capstone)
include_directories(third_party/capstone/include)

set(protobuf_BUILD_TESTS OFF CACHE BOOL "Build tests" FORCE)
set(protobuf_BUILD_EXAMPLES OFF CACHE BOOL "Build examples" FORCE)
set(protobuf_WITH_ZLIB OFF CACHE BOOL "Build with zlib" FORCE)
set(protobuf_BUILD_PROTOC_BINARIES ON CACHE BOOL "Build protoc" FORCE)
set(protobuf_BUILD_SHARED_LIBS OFF CACHE BOOL "Build shared libraries" FORCE)

add_subdirectory(third_party/protobuf/cmake)
# include_directories(third_party/protobuf/src)

add_subdirectory(third_party/zlib)
# include_directories(third_party/zlib)

include_directories(.)
include_directories(src)
include_directories(third_party/abseil-cpp)
include_directories("${CMAKE_CURRENT_BINARY_DIR}/src")

set(CMAKE_CXX_FLAGS "-W -Wall -Wno-sign-compare -fPIE")
set(CMAKE_CXX_FLAGS_DEBUG "-g1")
set(CMAKE_CXX_FLAGS_RELEASE "-O2")
set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "-O2 -g1")
set_source_files_properties(third_party/demumble/third_party/libcxxabi/cxa_demangle.cpp PROPERTIES COMPILE_FLAGS -Wno-implicit-fallthrough)

include(CheckCXXCompilerFlag)
CHECK_CXX_COMPILER_FLAG(-fdiagnostics-color=always SUPPORTS_COLOR_ALWAYS)
if (SUPPORTS_COLOR_ALWAYS)
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fdiagnostics-color=always")
endif ()

file(MAKE_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/src)
add_custom_command(
        OUTPUT ${CMAKE_CURRENT_BINARY_DIR}/src/bloaty.pb.cc
        COMMAND env protoc ${CMAKE_CURRENT_SOURCE_DIR}/src/bloaty.proto
        --cpp_out=${CMAKE_CURRENT_BINARY_DIR}/src
        -I${CMAKE_CURRENT_SOURCE_DIR}/src
)

file(COPY ${CMAKE_CURRENT_SOURCE_DIR}/src/bloaty_package.bloaty
        DESTINATION ${CMAKE_CURRENT_BINARY_DIR})

add_library(bloaty STATIC
        src/lib.cc
        src/bloaty.cc
        src/bloaty.h
        src/disassemble.cc
        ${CMAKE_CURRENT_BINARY_DIR}/src/bloaty.pb.cc
        src/dwarf/attr.h
        src/dwarf/attr.cc
        src/dwarf/dwarf_util.cc
        src/dwarf/debug_info.cc
        src/dwarf/line_info.cc
        src/dwarf.cc
        src/dwarf_constants.h
        src/eh_frame.cc
        src/elf.cc
        src/macho.cc
        src/pe.cc
        third_party/lief_pe/pe_structures.h
        src/range_map.cc
        src/range_map.h
        src/re.h
        src/util.cc
        src/util.h
        src/webassembly.cc
        # Until Abseil has a proper CMake build system
        third_party/abseil-cpp/absl/base/internal/raw_logging.cc # Grrrr...
        third_party/abseil-cpp/absl/base/internal/throw_delegate.cc
        third_party/abseil-cpp/absl/debugging/internal/demangle.cc
        third_party/abseil-cpp/absl/numeric/int128.cc
        third_party/abseil-cpp/absl/strings/ascii.cc
        third_party/abseil-cpp/absl/strings/charconv.cc
        third_party/abseil-cpp/absl/strings/escaping.cc
        third_party/abseil-cpp/absl/strings/internal/charconv_bigint.cc
        third_party/abseil-cpp/absl/strings/internal/charconv_parse.cc
        third_party/abseil-cpp/absl/strings/internal/escaping.cc
        third_party/abseil-cpp/absl/strings/internal/memutil.cc
        third_party/abseil-cpp/absl/strings/internal/utf8.cc
        third_party/abseil-cpp/absl/strings/match.cc
        third_party/abseil-cpp/absl/strings/numbers.cc
        third_party/abseil-cpp/absl/strings/str_cat.cc
        third_party/abseil-cpp/absl/strings/string_view.cc
        third_party/abseil-cpp/absl/strings/str_split.cc
        third_party/abseil-cpp/absl/strings/substitute.cc
        third_party/abseil-cpp/absl/types/bad_optional_access.cc
        # One source file, no special build system needed.
        third_party/demumble/third_party/libcxxabi/cxa_demangle.cpp
        )
set_property(TARGET bloaty PROPERTY FOLDER "bloaty")

set(LIBBLOATY_LIBS)
list(APPEND LIBBLOATY_LIBS libprotobuf)
list(APPEND LIBBLOATY_LIBS libprotoc)
list(APPEND LIBBLOATY_LIBS capstone-static)
list(APPEND LIBBLOATY_LIBS zlibstatic)
list(APPEND LIBBLOATY_LIBS Threads::Threads)

#add_executable(bloaty src/main.cc)
target_link_libraries(bloaty ${LIBBLOATY_LIBS})
set_property(TARGET bloaty PROPERTY FOLDER "bloaty")

add_custom_target(build-static DEPENDS bloaty
        COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:bloaty> ${CMAKE_CURRENT_BINARY_DIR}/$<TARGET_FILE_NAME:bloaty>
        COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:libprotobuf> ${CMAKE_CURRENT_BINARY_DIR}/$<TARGET_FILE_NAME:libprotobuf>
        COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:libprotoc> ${CMAKE_CURRENT_BINARY_DIR}/$<TARGET_FILE_NAME:libprotoc>
        COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:capstone-static> ${CMAKE_CURRENT_BINARY_DIR}/$<TARGET_FILE_NAME:capstone-static>
        COMMAND ${CMAKE_COMMAND} -E copy $<TARGET_FILE:zlibstatic> ${CMAKE_CURRENT_BINARY_DIR}/$<TARGET_FILE_NAME:zlibstatic>
        )