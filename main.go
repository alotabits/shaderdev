package main

import (
	"flag"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/alotabits/shaderdev/internal/gx"
	"github.com/alotabits/shaderdev/internal/obj"
	"github.com/go-gl/gl/all-core/gl"
	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"gopkg.in/fsnotify.v1"
)

type model struct {
	pos [][4]float32
	nor [][3]float32
	tex [][3]float32
	idx []uint32

	vao    uint32
	posBuf uint32
	idxBuf uint32
}

var cubeVertices = []float32{
	0, 0, 1,
	0, 0, 0,
	1, 0, 1,
	1, 0, 0,
	1, 1, 1,
	1, 1, 0,
	0, 1, 1,
	0, 1, 0,
}

var cubeIndices = []uint32{
	0, 1,
	2, 3,
	4, 5,
	6, 7,
	0, 1,
	6, 0, 4, 2,
	5, 3, 7, 1,
}

func loadModel(file string) (*model, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	o, err := obj.Decode(f)
	if err != nil {
		return nil, err
	}

	var m model

	/*
		opengl requires all vertex attributes to have the same number of elements,
		so here we remember index triplets we've seen before and reuse those indices,
		otherwise we record a new index and add the indexed obj values to the attribute arrays
	*/
	knownVerts := make(map[[3]int]uint32)
	for iface := range o.Face {
		for ivert := range o.Face[iface] {
			overt := o.Face[iface][ivert]
			kv, ok := knownVerts[overt]
			if ok {
				m.idx = append(m.idx, kv)
			} else {
				i := uint32(len(m.pos))
				m.idx = append(m.idx, i)
				knownVerts[overt] = i

				ip := overt[0]
				m.pos = append(m.pos, o.Pos[ip])

				if len(o.Tex) > 0 {
					it := overt[1]
					m.tex = append(m.tex, o.Tex[it])
				}

				if len(o.Nor) > 0 {
					in := overt[2]
					m.nor = append(m.nor, o.Nor[in])
				}
			}
		}
	}

	return &m, nil
}

func initModel(m *model, positionLoc, colorLoc uint32) {
	vao := gx.GenVertexArray()
	gl.BindVertexArray(vao)
	defer gl.BindVertexArray(0)

	var posBuf uint32
	gl.GenBuffers(1, &posBuf)
	gl.BindBuffer(gl.ARRAY_BUFFER, posBuf)
	defer gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	posLen := len(m.pos) * int(unsafe.Sizeof([4]float32{}))
	gl.BufferData(gl.ARRAY_BUFFER, posLen, gl.Ptr(m.pos), gl.STATIC_DRAW)
	if gx.IsValidAttribLoc(positionLoc) {
		gl.EnableVertexAttribArray(positionLoc)
		gl.VertexAttribPointer(positionLoc, 4, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}

	if gx.IsValidAttribLoc(colorLoc) {
		gl.EnableVertexAttribArray(colorLoc)
		gl.VertexAttribPointer(colorLoc, 4, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}

	var idxBuf uint32
	gl.GenBuffers(1, &idxBuf)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, idxBuf)
	idxLen := len(m.idx) * int(unsafe.Sizeof(uint32(0)))
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, idxLen, gl.Ptr(m.idx), gl.STATIC_DRAW)

	m.vao = vao
	m.posBuf = posBuf
	m.idxBuf = idxBuf
}

func updateModel(m *model, positionLoc, colorLoc uint32) {
	gl.BindVertexArray(m.vao)
	defer gl.BindVertexArray(0)

	gl.BindBuffer(gl.ARRAY_BUFFER, m.posBuf)
	defer gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	if gx.IsValidAttribLoc(positionLoc) {
		gl.EnableVertexAttribArray(positionLoc)
		gl.VertexAttribPointer(positionLoc, 4, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}

	if gx.IsValidAttribLoc(colorLoc) {
		gl.EnableVertexAttribArray(colorLoc)
		gl.VertexAttribPointer(colorLoc, 4, gl.FLOAT, false, 0, gl.PtrOffset(0))
	}
}

func drawModel(m *model) {
	gl.Enable(gl.DEPTH_TEST)
	defer gl.Disable(gl.DEPTH_TEST)
	gl.BindVertexArray(m.vao)
	defer gl.BindVertexArray(0)
	gl.DrawElements(gl.TRIANGLES, int32(len(m.idx)), gl.UNSIGNED_INT, gl.PtrOffset(0))
}

func init() {
	runtime.LockOSThread()
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	flag.Parse()

	err := glfw.Init()
	if err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, gl.TRUE)
	window, err := glfw.CreateWindow(400, 400, "Shaderdev", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer window.Destroy()
	window.MakeContextCurrent()

	log.Print("context: ", window.GetAttrib(glfw.ContextVersionMajor), ".", window.GetAttrib(glfw.ContextVersionMinor))

	err = gl.Init()
	if err != nil {
		log.Fatal(err)
	}

	gl.Enable(gl.DEBUG_OUTPUT)
	gl.DebugMessageCallback(gx.LogProc, unsafe.Pointer(nil))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	shaPrefixToStage := map[string]uint32{
		"vs":  gl.VERTEX_SHADER,
		"gs":  gl.GEOMETRY_SHADER,
		"tes": gl.TESS_EVALUATION_SHADER,
		"tcs": gl.TESS_CONTROL_SHADER,
		"fs":  gl.FRAGMENT_SHADER,
	}

	prog := newProgram()
	for _, arg := range flag.Args() {
		s := strings.SplitN(arg, ":", 2)
		if len(s) < 2 {
			log.Fatalln(arg, "is not a valid shader specification")
		}
		prefix, path := s[0], s[1]
		path = filepath.Clean(path)

		var ok bool
		var stage uint32
		if stage, ok = shaPrefixToStage[prefix]; !ok {
			log.Fatalln("unknown shader type for", arg)
		}

		dir, _ := filepath.Split(path)
		err = watcher.Add(dir)
		if err != nil {
			log.Fatalln(err)
		}

		addPath(prog, stage, path)
	}

	err = updateProgram(prog)
	if err != nil {
		log.Fatal(err)
	}

	modelObj, err := loadModel("monkey.obj")
	if err != nil {
		log.Fatal(err)
	}

	initModel(modelObj, prog.positionLoc, prog.colorLoc)

	ticker := time.NewTicker(1000 / 60 * time.Millisecond)
	start := time.Now()
	angle := float32(0)

	go func() {
		for err := range watcher.Errors {
			log.Println("watcher error:", err)
		}
	}()

	for !window.ShouldClose() {
		select {
		case evt := <-watcher.Events:
			if evt.Op&fsnotify.Write > 0 {
				log.Println(evt)
				err := pathChanged(prog, filepath.Clean(evt.Name))
				if err != nil {
					log.Println(err)
				}
			}
		case <-ticker.C:
			err := updateProgram(prog)
			if err != nil {
				log.Println(err)
				gl.UseProgram(0)
				gl.ClearColor(1, 0, 1, 1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				window.SwapBuffers()
				glfw.PollEvents()
				continue
			}

			updateModel(modelObj, prog.positionLoc, prog.colorLoc)

			windowWidth, windowHeight := window.GetSize()
			wdivh := float32(windowWidth) / float32(windowHeight)
			hdivw := float32(windowHeight) / float32(windowWidth)

			gl.UseProgram(prog.id)
			gl.ClearColor(0, 0, 0, 0)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			gl.Viewport(0, 0, int32(windowWidth), int32(windowHeight))

			if prog.viewportLoc >= 0 {
				gl.Uniform4f(prog.viewportLoc, 0, 0, float32(windowWidth), float32(windowHeight))
			}

			if prog.cursorLoc >= 0 {
				x, y := window.GetCursorPos()
				gl.Uniform4f(prog.cursorLoc, float32(x), float32(float64(windowHeight)-y), 0, 0)
			}

			if prog.timeLoc >= 0 {
				t := time.Now()
				d := t.Sub(start)
				gl.Uniform4f(prog.timeLoc, float32(t.Year()), float32(t.Month()), float32(t.Day()), float32(d.Seconds()))
			}

			if prog.projectionLoc >= 0 {
				var projectionMat mgl32.Mat4
				if wdivh > hdivw {
					projectionMat = mgl32.Frustum(wdivh*-0.75, wdivh*0.75, -0.75, 0.75, 20, 24)
				} else {
					projectionMat = mgl32.Frustum(-0.75, 0.75, hdivw*-0.75, hdivw*0.75, 20, 24)
				}
				gl.UniformMatrix4fv(prog.projectionLoc, 1, false, &projectionMat[0])
			}

			if prog.viewLoc >= 0 {
				viewMat := mgl32.Translate3D(0, 0, -22).Mul4(mgl32.HomogRotate3DX(math.Pi / 8))
				gl.UniformMatrix4fv(prog.viewLoc, 1, false, &viewMat[0])
			}

			var modelMat mgl32.Mat4

			if prog.modelLoc >= 0 {
				modelMat = mgl32.HomogRotate3DY(-angle).Mul4(mgl32.Translate3D(-0.5, -0.5, -0.5))
				gl.UniformMatrix4fv(prog.modelLoc, 1, false, &modelMat[0])
			}

			// Draw things that pivot only around Y-axis here

			/*
				if prog.modelLoc >= 0 {
					modelMat = modelMat.Mul4(
						mgl32.Translate3D(0.5, 0.5, 0.5),
					).Mul4(
						mgl32.HomogRotate3DX(angle),
					).Mul4(
						mgl32.Translate3D(-0.5, -0.5, -0.5),
					)
					gl.UniformMatrix4fv(prog.modelLoc, 1, false, &modelMat[0])
				}
			*/

			gl.Enable(gl.CULL_FACE)
			drawModel(modelObj)
			gl.Disable(gl.CULL_FACE)
			window.SwapBuffers()

			glfw.PollEvents()
			angle += 0.01
		}
	}
}
