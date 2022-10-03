// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"

	"github.com/vyeve/go-struct-optimize/fieldalignment"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	flag.Parse()
	singlechecker.Main(fieldalignment.Analyzer)
}
