package main

import (
	"fmt"
	"os"
	"io"
	"path"
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
		for i := 0; i < a.EntryCount(); i++ {
			file := a.Entry(i)

			fmt.Printf("\t%s (%s):\t0x%08x\n", file.Name, file.Type, file.Size);

			if file.Type == "aqo" {
				m, err := afp.NewModel(file.Data)
				ragequit(file.Name, err)

				for _, entry := range m.Entries {
					fmt.Printf("\t\t%s (%s):\t0x%08x\n", entry.Type, entry.SubType, entry.Size);
				}
			}
		}
	}

	if flagExtract != "" {
		os.MkdirAll(flagExtract, 0777);

		for i := 0; i < a.EntryCount(); i++ {
			file := a.Entry(i)
			fmt.Println("Extracting", file.Name, "...")

			f, err := os.Create(path.Join(flagExtract, file.Name));
			ragequit(file.Name, err)

			io.Copy(f, file.Data)
			f.Close()
		}
	}

	if flagWrite != "" {
		ofile, err := os.Create(flagWrite)
		ragequit(flagWrite, err)

		fmt.Fprintf(os.Stderr, "Writing to archive `%s`...\n", flagWrite)
		a.Write(ofile)
	}
}
