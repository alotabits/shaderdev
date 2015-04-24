package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"bitbucket.org/alotabits/gx"
	"github.com/go-gl/gl/all-core/gl"
)

type shader struct {
	id     uint32
	paths  []string
	update bool
}

type program struct {
	id            uint32
	shaderByStage map[uint32]*shader
	shadersByPath map[string][]*shader
	update        bool

	viewportLoc   int32
	projectionLoc int32
	viewLoc       int32
	modelLoc      int32
	cursorLoc     int32
	timeLoc       int32

	positionLoc uint32
	colorLoc    uint32
}

func newProgram() *program {
	var p program
	p.id = gl.CreateProgram()
	p.shaderByStage = make(map[uint32]*shader)
	p.shadersByPath = make(map[string][]*shader)
	p.update = true
	return &p
}

func updateShader(s *shader) error {
	if !s.update {
		return nil
	}

	s.update = false

	files := make([]io.Reader, len(s.paths))
	for i, p := range s.paths {
		file, err := os.Open(p)
		if err != nil {
			return err
		}
		files[i] = io.Reader(file)
	}

	b, err := ioutil.ReadAll(io.MultiReader(files...))
	if err != nil {
		return err
	}

	err = gx.CompileSource(s.id, [][]byte{b})
	if err != nil {
		return err
	}

	return nil
}

func getUniformLocation(program uint32, name string) int32 {
	loc := gl.GetUniformLocation(program, gl.Str(name))
	if !gx.IsValidUniformLoc(loc) {
		log.Println("missing uniform", name)
	}
	return loc
}

func getAttribLocation(program uint32, name string) uint32 {
	loc := uint32(gl.GetAttribLocation(program, gl.Str(name)))
	if !gx.IsValidAttribLoc(loc) {
		log.Println("missing attribute", name)
	}
	return loc
}

func updateProgram(p *program) error {
	if !p.update {
		return nil
	}

	p.update = false

	for _, s := range p.shaderByStage {
		err := updateShader(s)
		if err != nil {
			return err
		}
	}

	err := gx.LinkProgram(p.id)
	if err != nil {
		return err
	}

	p.viewportLoc = getUniformLocation(p.id, "viewport\x00")
	p.cursorLoc = getUniformLocation(p.id, "cursor\x00")
	p.timeLoc = getUniformLocation(p.id, "time\x00")
	p.projectionLoc = getUniformLocation(p.id, "projection\x00")
	p.viewLoc = getUniformLocation(p.id, "view\x00")
	p.modelLoc = getUniformLocation(p.id, "model\x00")
	p.positionLoc = getAttribLocation(p.id, "position\x00")
	p.colorLoc = getAttribLocation(p.id, "color\x00")

	return nil
}

func addPath(p *program, stage uint32, path string) {
	p.update = true
	s := p.shaderByStage[stage]
	if s == nil {
		s = &shader{}
		s.id = gl.CreateShader(stage)
		gl.AttachShader(p.id, s.id)
		p.shaderByStage[stage] = s
	}
	s.paths = append(s.paths, path)
	s.update = true

	p.shadersByPath[path] = append(p.shadersByPath[path], s)
}

func pathChanged(p *program, path string) error {
	var ss []*shader
	var ok bool
	if ss, ok = p.shadersByPath[path]; !ok {
		return fmt.Errorf("no shader associated with path %v", path)
	}

	p.update = true
	for _, s := range ss {
		s.update = true
	}

	return nil
}
