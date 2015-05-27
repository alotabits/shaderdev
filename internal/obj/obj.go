package obj

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type state func(s string) state

type elem int

const (
	comElem = elem(iota)
	posElem
	texElem
	norElem
	facElem
	errElem
)

func toElem(s string) elem {
	if strings.HasPrefix(s, "#") {
		return comElem
	}

	switch s {
	case "v":
		return posElem
	case "vt":
		return texElem
	case "vn":
		return norElem
	case "f":
		return facElem
	default:
		return errElem
	}
}

func toIndex(s string) (int, error) {
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	if val == 0 {
		return 0, fmt.Errorf("0 is not a valid index")
	}
	return int(val), nil
}

func adjustIndex(attIdx int, attLen int) (int, error) {
	res := attIdx
	if res < 0 {
		res += attLen
		if res < 0 {
			return 0, fmt.Errorf("relative index %v does not resolve to an attribute (i.e. too negative)", attIdx)
		}
	} else {
		res--
	}
	if res > attLen {
		return 0, fmt.Errorf("index %v does not resolve to an attribute (i.e. too large)")
	}

	return res, nil
}

type Obj struct {
	Pos  [][4]float32
	Tex  [][3]float32
	Nor  [][3]float32
	Face [][3][3]int
}

func (o *Obj) VertPos(face, vertex int) *[4]float32 {
	i := o.Face[face][vertex][0]
	return &o.Pos[i]
}

func (o *Obj) VertTex(face, vertex int) *[3]float32 {
	i := o.Face[face][vertex][1]
	return &o.Tex[i]
}

func (o *Obj) VertNor(face, vertex int) *[3]float32 {
	i := o.Face[face][vertex][2]
	return &o.Nor[i]
}

func parseFace(fields []string, _face *[][3]int) error {
	const (
		P = iota
		T
		N
	)

	face := *_face

	var err error
	var vertices [][]string
	for i, v := range fields {
		vertex := strings.Split(v, "/")
		if len(vertex) > 3 {
			return fmt.Errorf("vertex %v:%s: vertices cannot have more than three attributes", i, vertex)
		}
		vertices = append(vertices, vertex)
	}

	// The first vertex is a template for the following vertices
	numAtt := len(vertices[0])
	var skipTex bool
	if numAtt == 3 {
		skipTex = (len(vertices[0][T]) == 0)
	}

	for i, v := range vertices {
		var vertex [3]int
		if len(v) != numAtt {
			return fmt.Errorf("vertex %v:%s: all vertices must have the same number of attributes", i, vertices[i])
		}

		vertex[P], err = toIndex(v[P])
		if err != nil {
			return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
		}

		switch numAtt {
		case 2:
			vertex[T], err = toIndex(v[T])
			if err != nil {
				return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
			}
		case 3:
			if skipTex {
				if len(v[T]) != 0 {
					return fmt.Errorf("vertex %v:%s: all texture indices must be present or elided", i, vertices[i])
				}
				vertex[T] = 0
			} else {
				vertex[T], err = toIndex(v[T])
				if err != nil {
					return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
				}
			}

			vertex[N], err = toIndex(v[N])
			if err != nil {
				return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
			}
		}
	}

	*_face = face
	return nil
}

func Stream(r io.Reader) error {
	var emitPos func([4]float32)
	var emitNor func([3]float32)
	var emitTex func([3]float32)
	var emitFace func([][3]int)

	line := 0
	// reuse face between loops to reduce allocations
	var face [][3]int

	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line++
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			// rather than nest in len(fields) != 0
			continue
		}

		switch toElem(fields[0]) {
		case comElem:
			// nop
		case posElem:
			if len(fields) < 4 || len(fields) > 5 {
				return fmt.Errorf("%v: v requires 3 or 4 values", line)
			}

			var pos [4]float32
			// default w coordinate to 1, per spec
			pos[3] = 1
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return fmt.Errorf("%v: %v", line, err)
				}
				pos[i] = float32(f)
			}

			emitPos(pos)
		case texElem:
			if len(fields) < 3 || len(fields) > 4 {
				return fmt.Errorf("%v: vt requires 2 or 3 values", line)
			}

			var tex [3]float32
			// w coordinate defaults to 0, per spec
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return fmt.Errorf("%v: %v", line, err)
				}
				tex[i] = float32(f)
			}

			emitTex(tex)
		case norElem:
			if len(fields) != 4 {
				return fmt.Errorf("%v: vn requires 3 values", line)
			}

			var nor [3]float32
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return fmt.Errorf("%v: %v", line, err)
				}
				nor[i] = float32(f)
			}

			emitNor(nor)
		case facElem:
			if len(fields) != 4 {
				return fmt.Errorf("%v: f requires 3 vertices", line)
			}

			err := parseFace(fields[1:], &face)
			if err != nil {
				return fmt.Errorf("%v: %v", line, err)
			}

			emitFace(face)
		case errElem:
			fmt.Printf("%v: %s element not supported\n", line, fields[0])
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func Decode(r io.Reader) (*Obj, error) {
	const (
		P = iota
		T
		N
	)

	var o Obj

	addFace := func(fields []string) error {
		// addFace is always called with 3 fields
		if len(fields) != 3 {
			panic("addFace: number of fields != 3, have " + strconv.Itoa(len(fields)))
		}

		var err error
		var vertices [3][]string
		for i, v := range fields {
			vertices[i] = strings.Split(v, "/")
			if len(vertices[i]) > 3 {
				return fmt.Errorf("vertex %v:%s: vertices cannot have more than three attributes", i, vertices[i])
			}
		}

		// The first vertex is a template for the following vertices
		numAtt := len(vertices[0])
		var skipTex bool
		if numAtt == 3 {
			skipTex = (len(vertices[0][T]) == 0)
		}

		// Start a new face
		f := len(o.Face)
		o.Face = append(o.Face, [3][3]int{})

		for i, vert := range vertices {
			if len(vert) != numAtt {
				return fmt.Errorf("vertex %v:%s: all vertices must have the same number of attributes", i, vertices[i])
			}

			o.Face[f][i][P], err = toIndex(vert[P])
			if err != nil {
				return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
			}

			switch numAtt {
			case 2:
				o.Face[f][i][T], err = toIndex(vert[T])
				if err != nil {
					return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
				}
			case 3:
				if skipTex {
					if len(vert[T]) != 0 {
						return fmt.Errorf("vertex %v:%s: all texture indices must be present or elided", i, vertices[i])
					}
					o.Face[f][i][T] = 0
				} else {
					o.Face[f][i][T], err = toIndex(vert[T])
					if err != nil {
						return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
					}
				}

				o.Face[f][i][N], err = toIndex(vert[N])
				if err != nil {
					return fmt.Errorf("vertex %v:%s: %v", i, vertices[i], err)
				}
			}
		}

		for i := range o.Face[f] {
			o.Face[f][i][P], err = adjustIndex(o.Face[f][i][P], len(o.Pos))
			if err != nil {
				return fmt.Errorf("vertex %v:%s:v-index: %v", i, vertices[i], err)
			}
			o.Face[f][i][T], err = adjustIndex(o.Face[f][i][T], len(o.Tex))
			if err != nil {
				return fmt.Errorf("vertex %v:%s:vt-index: %v", i, vertices[i], err)
			}
			o.Face[f][i][N], err = adjustIndex(o.Face[f][i][N], len(o.Nor))
			if err != nil {
				return fmt.Errorf("vertex %v:%s:vn-index: %v", i, vertices[i], err)
			}
		}

		return nil
	}

	line := 0
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line++
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			// rather than nest in len(fields) != 0
			continue
		}

		switch toElem(fields[0]) {
		case comElem:
			// nop
		case posElem:
			if len(fields) < 4 || len(fields) > 5 {
				return nil, fmt.Errorf("%v: v requires 3 or 4 values", line)
			}

			p := len(o.Pos)
			o.Pos = append(o.Pos, [4]float32{})
			// default w coordinate to 1, per spec
			o.Pos[p][3] = 1
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return nil, fmt.Errorf("%v: %v", line, err)
				}
				o.Pos[p][i] = float32(f)
			}
		case texElem:
			if len(fields) < 3 || len(fields) > 4 {
				return nil, fmt.Errorf("%v: vt requires 2 or 3 values", line)
			}

			t := len(o.Tex)
			o.Tex = append(o.Tex, [3]float32{})
			// w coordinate defaults to 0, per spec
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return nil, fmt.Errorf("%v: %v", line, err)
				}
				o.Tex[t][i] = float32(f)
			}
		case norElem:
			if len(fields) != 4 {
				return nil, fmt.Errorf("%v: vn requires 3 values", line)
			}

			n := len(o.Nor)
			o.Nor = append(o.Nor, [3]float32{})
			for i, v := range fields[1:] {
				f, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return nil, fmt.Errorf("%v: %v", line, err)
				}
				o.Nor[n][i] = float32(f)
			}
		case facElem:
			if len(fields) != 4 {
				return nil, fmt.Errorf("%v: f requires 3 vertices", line)
			}

			err := addFace(fields[1:])
			if err != nil {
				return nil, fmt.Errorf("%v: %v", line, err)
			}
		case errElem:
			fmt.Printf("%v: %s element not supported\n", line, fields[0])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &o, nil
}
