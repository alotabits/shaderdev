package gx

import (
	"fmt"
	"io/ioutil"
	"log"
	"unsafe"

	"github.com/go-gl/gl/all-core/gl"
)

var LogProc = gl.DebugProc(
	func(
		source uint32,
		gltype uint32,
		id uint32,
		severity uint32,
		length int32,
		message string,
		userParam unsafe.Pointer,
	) {
		log.Println(message)
	},
)

func ErrorStr(e uint32) string {
	switch e {
	case gl.NO_ERROR:
		return ""
	case gl.INVALID_ENUM:
		return "invalid enum"
	case gl.INVALID_VALUE:
		return "invalid value"
	case gl.INVALID_OPERATION:
		return "invalid operation"
	case gl.INVALID_FRAMEBUFFER_OPERATION:
		return "invalid framebuffer operation"
	case gl.OUT_OF_MEMORY:
		return "out of memory"
	default:
		return "unknown"
	}
}

func StageStr(stage uint32) string {
	switch stage {
	case gl.VERTEX_SHADER:
		return "vertex"
	case gl.GEOMETRY_SHADER:
		return "geometry"
	case gl.TESS_EVALUATION_SHADER:
		return "tesselation evaluation"
	case gl.TESS_CONTROL_SHADER:
		return "tesselation control"
	case gl.FRAGMENT_SHADER:
		return "fragment"
	default:
		return "unknown"
	}
}

func StageEnum(stage uint32) string {
	switch stage {
	case gl.VERTEX_SHADER:
		return "VERTEX"
	case gl.GEOMETRY_SHADER:
		return "GEOMETRY"
	case gl.TESS_EVALUATION_SHADER:
		return "TESS_EVALUATION"
	case gl.TESS_CONTROL_SHADER:
		return "TESS_CONTROL"
	case gl.FRAGMENT_SHADER:
		return "FRAGMENT"
	default:
		return "UNKNOWN"
	}
}

func LogError() {
	errStr := ErrorStr(gl.GetError())
	if errStr != "" {
		log.Println("GL error:", errStr)
	}
}

func IsValidAttribLoc(l uint32) bool {
	return (l & 0x80000000) == 0
}

func IsValidUniformLoc(l int32) bool {
	return l != -1
}

func IsValidUniformIdx(l uint32) bool {
	return l != gl.INVALID_INDEX
}

func CompileSource(sha uint32, src [][]byte) error {
	srcptr := make([]*byte, len(src))
	srclen := make([]int32, len(src))
	for i, s := range src {
		srcptr[i] = &s[0]
		srclen[i] = int32(len(s))
	}

	gl.ShaderSource(sha, int32(len(src)), &srcptr[0], &srclen[0])
	gl.CompileShader(sha)
	var status int32
	gl.GetShaderiv(sha, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var loglen int32
		gl.GetShaderiv(sha, gl.INFO_LOG_LENGTH, &loglen)
		buf := make([]byte, loglen)
		gl.GetShaderInfoLog(sha, loglen, nil, &buf[0])
		return fmt.Errorf("%s", buf)
	}

	return nil
}

func AppendFiles(sources [][]byte, files []string) ([][]byte, error) {
	for _, f := range files {
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			return sources, err
		}
		sources = append(sources, buf)
	}

	return sources, nil
}

func LinkProgram(prog uint32) error {
	gl.LinkProgram(prog)
	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var loglen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &loglen)
		buf := make([]byte, loglen)
		gl.GetProgramInfoLog(prog, loglen, nil, &buf[0])
		return fmt.Errorf("%s", buf)
	}

	return nil
}

func CreateTexture1D(internalformat int32, width int32, format uint32, xtype uint32, pixels unsafe.Pointer) uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_1D, tex)
	gl.TexImage1D(gl.TEXTURE_1D, 0, internalformat, width, 0, format, xtype, pixels)
	return tex
}

func CreateTexture2D(internalformat int32, width int32, height int32, format uint32, xtype uint32, pixels unsafe.Pointer) uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internalformat, width, height, 0, format, xtype, pixels)
	return tex
}

func ActiveTexture(unit uint32) {
	gl.ActiveTexture(gl.TEXTURE0 + unit)
}

func GenSampler() uint32 {
	var o uint32
	gl.GenSamplers(1, &o)
	return o
}

func GenBuffer() uint32 {
	var o uint32
	gl.GenBuffers(1, &o)
	return o
}

func GenVertexArray() uint32 {
	var o uint32
	gl.GenVertexArrays(1, &o)
	return o
}
