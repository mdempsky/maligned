// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"log"
	"sort"
	"unsafe"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
)

var fset = token.NewFileSet()

func main() {
	flag.Parse()

	var conf loader.Config
	conf.Fset = fset
	for _, arg := range flag.Args() {
		conf.Import(arg)
	}
	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	var v visitor
	for _, pkg := range prog.InitialPackages() {
		v.pkg = pkg
		for _, file := range pkg.Files {
			ast.Walk(&v, file)
		}
	}
}

type visitor struct {
	pkg *loader.PackageInfo
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	if str, ok := node.(*ast.StructType); ok {
		malign(node.Pos(), v.pkg.Types[str].Type.(*types.Struct))
	}
	return v
}

func malign(pos token.Pos, str *types.Struct) {
	wordSize := int64(8)
	maxAlign := int64(8)
	switch build.Default.GOARCH {
	case "386", "arm":
		wordSize, maxAlign = 4, 4
	case "amd64p32":
		wordSize = 4
	}

	s := gcSizes{wordSize, maxAlign}
	sz, opt := s.Sizeof(str), optimalSize(str, &s)
	if sz != opt {
		fmt.Printf("%s: struct of size %d could be %d\n", fset.Position(pos), sz, opt)
	}
}

func optimalSize(str *types.Struct, sizes *gcSizes) int64 {
	nf := str.NumFields()
	fields := make([]*types.Var, nf)
	alignofs := make([]int64, nf)
	sizeofs := make([]int64, nf)
	for i := 0; i < nf; i++ {
		fields[i] = str.Field(i)
		ft := fields[i].Type()
		alignofs[i] = sizes.Alignof(ft)
		sizeofs[i] = sizes.Sizeof(ft)
	}
	sort.Sort(&byAlignAndSize{fields, alignofs, sizeofs})
	return sizes.Sizeof(types.NewStruct(fields, nil))
}

type byAlignAndSize struct {
	fields   []*types.Var
	alignofs []int64
	sizeofs  []int64
}

func (s *byAlignAndSize) Len() int { return len(s.fields) }
func (s *byAlignAndSize) Swap(i, j int) {
	s.fields[i], s.fields[j] = s.fields[j], s.fields[i]
	s.alignofs[i], s.alignofs[j] = s.alignofs[j], s.alignofs[i]
	s.sizeofs[i], s.sizeofs[j] = s.sizeofs[j], s.sizeofs[i]
}

func (s *byAlignAndSize) Less(i, j int) bool {
	// Place zero sized objects before non-zero sized objects.
	if s.sizeofs[i] == 0 && s.sizeofs[j] != 0 {
		return true
	}
	if s.sizeofs[j] == 0 && s.sizeofs[i] != 0 {
		return false
	}

	// Next, place more tightly aligned objects before less tightly aligned objects.
	if s.alignofs[i] != s.alignofs[j] {
		return s.alignofs[i] > s.alignofs[j]
	}

	// Lastly, order by size.
	if s.sizeofs[i] != s.sizeofs[j] {
		return s.sizeofs[i] > s.sizeofs[j]
	}

	return false
}

// Code below based on go/types.StdSizes.

type gcSizes struct {
	WordSize int64
	MaxAlign int64
}

func (s *gcSizes) Alignof(T types.Type) int64 {
	// NOTE: On amd64, complex64 is 8 byte aligned,
	// even though float32 is only 4 byte aligned.

	// For arrays and structs, alignment is defined in terms
	// of alignment of the elements and fields, respectively.
	switch t := T.Underlying().(type) {
	case *types.Array:
		// spec: "For a variable x of array type: unsafe.Alignof(x)
		// is the same as unsafe.Alignof(x[0]), but at least 1."
		return s.Alignof(t.Elem())
	case *types.Struct:
		// spec: "For a variable x of struct type: unsafe.Alignof(x)
		// is the largest of the values unsafe.Alignof(x.f) for each
		// field f of x, but at least 1."
		max := int64(1)
		for i, nf := 0, t.NumFields(); i < nf; i++ {
			if a := s.Alignof(t.Field(i).Type()); a > max {
				max = a
			}
		}
		return max
	}
	a := s.Sizeof(T) // may be 0
	// spec: "For a variable x of any type: unsafe.Alignof(x) is at least 1."
	if a < 1 {
		return 1
	}
	if a > s.MaxAlign {
		return s.MaxAlign
	}
	return a
}

var basicSizes = [...]uintptr{
	types.Bool:       unsafe.Sizeof(false),
	types.Int:        unsafe.Sizeof(int(0)),
	types.Int8:       unsafe.Sizeof(int8(0)),
	types.Int16:      unsafe.Sizeof(int16(0)),
	types.Int32:      unsafe.Sizeof(int32(0)),
	types.Int64:      unsafe.Sizeof(int64(0)),
	types.Uint:       unsafe.Sizeof(uint(0)),
	types.Uint8:      unsafe.Sizeof(uint8(0)),
	types.Uint16:     unsafe.Sizeof(uint16(0)),
	types.Uint32:     unsafe.Sizeof(uint32(0)),
	types.Uint64:     unsafe.Sizeof(uint64(0)),
	types.Uintptr:    unsafe.Sizeof(uintptr(0)),
	types.Float32:    unsafe.Sizeof(float32(0)),
	types.Float64:    unsafe.Sizeof(float64(0)),
	types.Complex64:  unsafe.Sizeof(complex64(0)),
	types.Complex128: unsafe.Sizeof(complex128(0)),
}

func (s *gcSizes) Sizeof(T types.Type) int64 {
	switch t := T.Underlying().(type) {
	case *types.Basic:
		k := t.Kind()
		if int(k) < len(basicSizes) {
			if s := basicSizes[k]; s > 0 {
				return int64(s)
			}
		}
		if k == types.String {
			return s.WordSize * 2
		}
	case *types.Array:
		n := t.Len()
		if n == 0 {
			return 0
		}
		a := s.Alignof(t.Elem())
		z := s.Sizeof(t.Elem())
		return align(z, a)*(n-1) + z
	case *types.Slice:
		return s.WordSize * 3
	case *types.Struct:
		nf := t.NumFields()
		if nf == 0 {
			return 0
		}

		var o int64
		max := int64(1)
		for i := 0; i < nf; i++ {
			ft := t.Field(i).Type()
			a, sz := s.Alignof(ft), s.Sizeof(ft)
			if a > max {
				max = a
			}
			if i == nf-1 && sz == 0 && o != 0 {
				sz = 1
			}
			o = align(o, a) + sz
		}
		return align(o, max)
	case *types.Interface:
		return s.WordSize * 2
	}
	return s.WordSize // catch-all
}

// align returns the smallest y >= x such that y % a == 0.
func align(x, a int64) int64 {
	y := x + a - 1
	return y - y%a
}
