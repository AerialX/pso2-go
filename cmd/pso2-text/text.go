package main

import (
	"fmt"
	"os"
	"flag"
	"aaronlindsay.com/go/pkg/pso2/text"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: text.go [flags] file.text")
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
	var flagWrite string

	flag.Usage = usage
	flag.BoolVar(&flagPrint, "p", false, "print details about the file")
	flag.StringVar(&flagWrite, "w", "", "write a repacked file")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "no filename provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	tpath := flag.Arg(0)
	fmt.Fprintf(os.Stderr, "Opening file `%s`...\n", tpath)
	f, err := os.OpenFile(tpath, os.O_RDONLY, 0);
	ragequit(tpath, err)

	t, err := text.NewTextFile(f)
	ragequit(tpath, err)

	if flagPrint {
		for i, entry := range t.Entries {
			fmt.Printf("%08x: %s\n", entry.Value, entry.Text)

			if entry.TextStatus == text.TextEntryString {
				t.Entries[i].Text = "LOLOLOL"
			}
		}

		for _, p := range t.Pairs {
			fmt.Printf("%s: %s\n", p.Identifier, p.String)
		}
	}

	if flagWrite != "" {
		ofile, err := os.Create(flagWrite)
		ragequit(flagWrite, err)

		fmt.Fprintf(os.Stderr, "Writing to `%s`...\n", flagWrite)
		t.Write(ofile)
	}
}
