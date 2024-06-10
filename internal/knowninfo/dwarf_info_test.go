package knowninfo

import (
	"strconv"
	"testing"
)

func TestLanguageString(t *testing.T) {
	tests := []struct {
		input    Language
		expected string
	}{
		{DwLangC89, "C89"},
		{DwLangC, "C"},
		{DwLangAda83, "Ada83"},
		{DwLangCPP, "C++"},
		{DwLangCobol74, "Cobol74"},
		{DwLangCobol85, "Cobol85"},
		{DwLangFortran77, "Fortran77"},
		{DwLangFortran90, "Fortran90"},
		{DwLangPascal83, "Pascal83"},
		{DwLangModula2, "Modula2"},
		{DwLangJava, "Java"},
		{DwLangC99, "C99"},
		{DwLangAda95, "Ada95"},
		{DwLangFortran95, "Fortran95"},
		{DwLangPLI, "PLI"},
		{DwLangObjC, "ObjC"},
		{DwLangObjCPP, "ObjC++"},
		{DwLangUPC, "UPC"},
		{DwLangD, "D"},
		{DwLangPython, "Python"},
		{DwLangOpenCL, "OpenCL"},
		{DwLangGo, "Go"},
		{DwLangModula3, "Modula3"},
		{DwLangHaskell, "Haskell"},
		{DwLangCPP03, "C++03"},
		{DwLangCPP11, "C++11"},
		{DwLangOCaml, "OCaml"},
		{DwLangRust, "Rust"},
		{DwLangC11, "C11"},
		{DwLangSwift, "Swift"},
		{DwLangJulia, "Julia"},
		{DwLangDylan, "Dylan"},
		{DwLangCPP14, "C++14"},
		{DwLangFortran03, "Fortran03"},
		{DwLangFortran08, "Fortran08"},
		{DwLangRenderScript, "RenderScript"},
		{DwLangBLISS, "BLISS"},
		{DwLangKotlin, "Kotlin"},
		{DwLangZig, "Zig"},
		{DwLangCrystal, "Crystal"},
		{DwLangCPP17, "C++17"},
		{DwLangCPP20, "C++20"},
		{DwLangC17, "C17"},
		{DwLangFortran18, "Fortran18"},
		{DwLangAda2005, "Ada2005"},
		{DwLangAda2012, "Ada2012"},
		{DwLangHIP, "HIP"},
		{DwLangAssembly, "Assembly"},
		{DwLangCSharp, "C#"},
		{DwLangMojo, "Mojo"},
		{DwLangGLSL, "GLSL"},
		{DwLangGLSLES, "GLSLES"},
		{DwLangHLSL, "HLSL"},
		{DwLangOpenCLCPP, "OpenCL++"},
		{DwLangCPPForOpenCL, "C++ForOpenCL"},
		{DwLangSYCL, "SYCL"},
		{DwLangRuby, "Ruby"},
		{DwLangMove, "Move"},
		{DwLangHylo, "Hylo"},
		{Language(0x9999), "Language(39321)"}, // Test case for an unknown language constant
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(int(tt.input)), func(t *testing.T) {
			result := LanguageString(tt.input)
			if result != tt.expected {
				t.Errorf("LanguageString(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
