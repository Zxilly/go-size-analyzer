cmake_minimum_required(VERSION 3.0)
project(go-size-analysis)

set(CMAKE_CXX_STANDARD 20)

include_directories(${PROJECT_SOURCE_DIR})
add_executable(go-size-analysis main.cpp GoNM.cpp utils.cpp utils.h)
