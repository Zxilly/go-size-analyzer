package dwarf

import "strconv"

type Language = int64

const (
	DwLangC89          Language = 0x0001
	DwLangC            Language = 0x0002
	DwLangAda83        Language = 0x0003
	DwLangCPP          Language = 0x0004
	DwLangCobol74      Language = 0x0005
	DwLangCobol85      Language = 0x0006
	DwLangFortran77    Language = 0x0007
	DwLangFortran90    Language = 0x0008
	DwLangPascal83     Language = 0x0009
	DwLangModula2      Language = 0x000a
	DwLangJava         Language = 0x000b
	DwLangC99          Language = 0x000c
	DwLangAda95        Language = 0x000d
	DwLangFortran95    Language = 0x000e
	DwLangPLI          Language = 0x000f
	DwLangObjC         Language = 0x0010
	DwLangObjCPP       Language = 0x0011
	DwLangUPC          Language = 0x0012
	DwLangD            Language = 0x0013
	DwLangPython       Language = 0x0014
	DwLangOpenCL       Language = 0x0015
	DwLangGo           Language = 0x0016
	DwLangModula3      Language = 0x0017
	DwLangHaskell      Language = 0x0018
	DwLangCPP03        Language = 0x0019
	DwLangCPP11        Language = 0x001a
	DwLangOCaml        Language = 0x001b
	DwLangRust         Language = 0x001c
	DwLangC11          Language = 0x001d
	DwLangSwift        Language = 0x001e
	DwLangJulia        Language = 0x001f
	DwLangDylan        Language = 0x0020
	DwLangCPP14        Language = 0x0021
	DwLangFortran03    Language = 0x0022
	DwLangFortran08    Language = 0x0023
	DwLangRenderScript Language = 0x0024
	DwLangBLISS        Language = 0x0025
	DwLangKotlin       Language = 0x0026
	DwLangZig          Language = 0x0027
	DwLangCrystal      Language = 0x0028
	DwLangCPP17        Language = 0x002a
	DwLangCPP20        Language = 0x002b
	DwLangC17          Language = 0x002c
	DwLangFortran18    Language = 0x002d
	DwLangAda2005      Language = 0x002e
	DwLangAda2012      Language = 0x002f
	DwLangHIP          Language = 0x0030
	DwLangAssembly     Language = 0x0031
	DwLangCSharp       Language = 0x0032
	DwLangMojo         Language = 0x0033
	DwLangGLSL         Language = 0x0034
	DwLangGLSLES       Language = 0x0035
	DwLangHLSL         Language = 0x0036
	DwLangOpenCLCPP    Language = 0x0037
	DwLangCPPForOpenCL Language = 0x0038
	DwLangSYCL         Language = 0x0039
	DwLangRuby         Language = 0x0040
	DwLangMove         Language = 0x0041
	DwLangHylo         Language = 0x0042
)

// LanguageString returns the string representation of the Language constant.
func LanguageString(l Language) string {
	switch l {
	case DwLangC89:
		return "C89"
	case DwLangC:
		return "C"
	case DwLangAda83:
		return "Ada83"
	case DwLangCPP:
		return "C++"
	case DwLangCobol74:
		return "Cobol74"
	case DwLangCobol85:
		return "Cobol85"
	case DwLangFortran77:
		return "Fortran77"
	case DwLangFortran90:
		return "Fortran90"
	case DwLangPascal83:
		return "Pascal83"
	case DwLangModula2:
		return "Modula2"
	case DwLangJava:
		return "Java"
	case DwLangC99:
		return "C99"
	case DwLangAda95:
		return "Ada95"
	case DwLangFortran95:
		return "Fortran95"
	case DwLangPLI:
		return "PLI"
	case DwLangObjC:
		return "ObjC"
	case DwLangObjCPP:
		return "ObjC++"
	case DwLangUPC:
		return "UPC"
	case DwLangD:
		return "D"
	case DwLangPython:
		return "Python"
	case DwLangOpenCL:
		return "OpenCL"
	case DwLangGo:
		return "Go"
	case DwLangModula3:
		return "Modula3"
	case DwLangHaskell:
		return "Haskell"
	case DwLangCPP03:
		return "C++03"
	case DwLangCPP11:
		return "C++11"
	case DwLangOCaml:
		return "OCaml"
	case DwLangRust:
		return "Rust"
	case DwLangC11:
		return "C11"
	case DwLangSwift:
		return "Swift"
	case DwLangJulia:
		return "Julia"
	case DwLangDylan:
		return "Dylan"
	case DwLangCPP14:
		return "C++14"
	case DwLangFortran03:
		return "Fortran03"
	case DwLangFortran08:
		return "Fortran08"
	case DwLangRenderScript:
		return "RenderScript"
	case DwLangBLISS:
		return "BLISS"
	case DwLangKotlin:
		return "Kotlin"
	case DwLangZig:
		return "Zig"
	case DwLangCrystal:
		return "Crystal"
	case DwLangCPP17:
		return "C++17"
	case DwLangCPP20:
		return "C++20"
	case DwLangC17:
		return "C17"
	case DwLangFortran18:
		return "Fortran18"
	case DwLangAda2005:
		return "Ada2005"
	case DwLangAda2012:
		return "Ada2012"
	case DwLangHIP:
		return "HIP"
	case DwLangAssembly:
		return "Assembly"
	case DwLangCSharp:
		return "C#"
	case DwLangMojo:
		return "Mojo"
	case DwLangGLSL:
		return "GLSL"
	case DwLangGLSLES:
		return "GLSLES"
	case DwLangHLSL:
		return "HLSL"
	case DwLangOpenCLCPP:
		return "OpenCL++"
	case DwLangCPPForOpenCL:
		return "C++ForOpenCL"
	case DwLangSYCL:
		return "SYCL"
	case DwLangRuby:
		return "Ruby"
	case DwLangMove:
		return "Move"
	case DwLangHylo:
		return "Hylo"
	default:
		return "Language(" + strconv.Itoa(int(l)) + ")"
	}
}
