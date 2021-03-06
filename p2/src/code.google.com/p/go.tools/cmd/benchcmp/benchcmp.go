// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Benchcmp is a utility for comparing benchmark runs.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
)

var (
	changedOnly = flag.Bool("changed", false, "show only benchmarks that have changed")
	magSort     = flag.Bool("mag", false, "sort benchmarks by magnitude of change")
)

const usageFooter = `
Each input file should be from:
	go test -test.run=NONE -test.bench=. > [old,new].txt

Benchcmp compares old and new for each benchmark.

If -test.benchmem=true is added to the "go test" command
benchcmp will also compare memory allocations.
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s old.txt new.txt\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, usageFooter)
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}

	before := parseFile(flag.Arg(0))
	after := parseFile(flag.Arg(1))

	cmps, warnings := Correlate(before, after)

	for _, warn := range warnings {
		fmt.Fprintln(os.Stderr, warn)
	}

	if len(cmps) == 0 {
		fatal("benchcmp: no repeated benchmarks")
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 5, ' ', 0)
	defer w.Flush()

	var header bool // Has the header has been displayed yet for a given block?

	if *magSort {
		sort.Sort(ByDeltaNsOp(cmps))
	} else {
		sort.Sort(ByParseOrder(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(NsOp) {
			continue
		}
		if delta := cmp.DeltaNsOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "benchmark\told ns/op\tnew ns/op\tdelta\t\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", cmp.Name(), formatNs(cmp.Before.NsOp), formatNs(cmp.After.NsOp), delta.Percent())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaMbS(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(MbS) {
			continue
		}
		if delta := cmp.DeltaMbS(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\told MB/s\tnew MB/s\tspeedup\t\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%.2f\t%.2f\t%s\t\n", cmp.Name(), cmp.Before.MbS, cmp.After.MbS, delta.Multiple())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaAllocsOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(AllocsOp) {
			continue
		}
		if delta := cmp.DeltaAllocsOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\told allocs\tnew allocs\tdelta\t\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\t\n", cmp.Name(), cmp.Before.AllocsOp, cmp.After.AllocsOp, delta.Percent())
		}
	}

	header = false
	if *magSort {
		sort.Sort(ByDeltaBOp(cmps))
	}
	for _, cmp := range cmps {
		if !cmp.Measured(BOp) {
			continue
		}
		if delta := cmp.DeltaBOp(); !*changedOnly || delta.Changed() {
			if !header {
				fmt.Fprintf(w, "\nbenchmark\told bytes\tnew bytes\tdelta\t\n")
				header = true
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\t\n", cmp.Name(), cmp.Before.BOp, cmp.After.BOp, cmp.DeltaBOp().Percent())
		}
	}
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func parseFile(path string) BenchSet {
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	bb, err := ParseBenchSet(f)
	if err != nil {
		fatal(err)
	}
	return bb
}

// formatNs formats ns measurements to expose a useful amount of
// precision. It mirrors the ns precision logic of testing.B.
func formatNs(ns float64) string {
	prec := 0
	switch {
	case ns < 10:
		prec = 2
	case ns < 100:
		prec = 1
	}
	return strconv.FormatFloat(ns, 'f', prec, 64)
}
