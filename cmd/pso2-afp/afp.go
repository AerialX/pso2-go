package main

import (
	"fmt"
	"os"
	"flag"
	"aaronlindsay.com/go/pkg/pso2/afp"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: afp [flags] archive.afp")
	flag.PrintDefaults()
	os.Exit(2)
}

func ragequit(apath string, err error) {
	if err != nil {
		if apath != "" {
			fmt.Fprintf(os.Stderr, "error with file `%s`\n", apath)
		}
		fmt.Fprintln(os.Stderr, err);
		os.Exit(1)
	}
}

func main() {
	var flagPrint bool
	var flagExtract string
	var flagWrite string

	flag.Usage = usage
	flag.BoolVar(&flagPrint, "p", false, "print details about the archive")
	flag.StringVar(&flagExtract, "x", "", "extract the archive to a folder")
	flag.StringVar(&flagWrite, "w", "", "write a repacked archive")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "no archive provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	apath := flag.Arg(0)
	fmt.Fprintf(os.Stderr, "Opening archive `%s`...\n", apath)
	f, err := os.OpenFile(apath, os.O_RDONLY, 0);
	ragequit(apath, err)

	a, err := afp.NewArchive(f)
	ragequit(apath, err)

	if flagPrint {
		fmt.Println(a)
	}

	if flagExtract != "" {
	}

	if flagWrite != "" {
		ofile, err := os.Create(flagWrite)
		ragequit(flagWrite, err)

		fmt.Fprintf(os.Stderr, "Writing to archive `%s`...\n", flagWrite)
		a.Write(ofile)
	}
}
