package main

import (
	"fmt"
	"os"
	"io"
	"flag"
	"bufio"
	"strings"
	"errors"
	"path"
	"aaronlindsay.com/go/pkg/pso2/ice"
	"aaronlindsay.com/go/pkg/pso2/util"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pso2-ice [flags] archive.ice")
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

type flagReplaceType map[string]string

func (f flagReplaceType) String() (value string) {
	value = `"`
	first := true
	for i, s := range f {
		if !first {
			value += ","
		}
		first = false
		value += i + ":" + s
	}
	value += `"`
	return
}

func (f *flagReplaceType) Set(value string) error {
	*f = make(map[string]string)

	values := strings.Split(value, ",")

	for _, v := range values {
		value := strings.Split(v, ":")

		if len(value) != 2 {
			return errors.New("invalid replacement format")
		}

		(*f)[value[0]] = value[1]
	}

	return nil
}

func main() {
	var flagPrint bool
	var flagExtract string
	var flagWrite string
	var flagReplace flagReplaceType

	flag.Usage = usage
	flag.BoolVar(&flagPrint, "p", false, "print details about the archive")
	flag.StringVar(&flagExtract, "x", "", "extract the archive to a folder")
	flag.StringVar(&flagWrite, "w", "", "write a repacked archive")
	flag.Var(&flagReplace, "r", `replace a file while repacking, use with -w (comma-separated, entry format is "filename:path". an empty path deletes the file from the archive)`)
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

	a, err := ice.NewArchive(util.BufReader(f))
	ragequit(apath, err)

	if flagPrint {
		for i := 0; i < a.GroupCount(); i++ {
			group := a.Group(i)

			fmt.Printf("Archive Group %d (0x%04x files)\n", i, len(group.Files))
			for _, file := range group.Files {
				fmt.Printf("\t%s (%s):\t0x%08x\n", file.Name, file.Type, file.Size);
			}
		}
	}

	if flagExtract != "" {
		for i := 0; i < a.GroupCount(); i++ {
			extPath := path.Join(flagExtract, fmt.Sprintf("%d", i))
			os.MkdirAll(extPath, 0777);

			group := a.Group(i)

			for _, file := range group.Files {
				fmt.Println("Extracting", file.Name, "...")

				f, err := os.Create(path.Join(extPath, file.Name));
				ragequit(file.Name, err)

				io.Copy(f, file.Data)
				f.Close()
			}
		}
	}

	if flagWrite != "" {
		ofile, err := os.Create(flagWrite)
		ragequit(flagWrite, err)

		for i := 0; i < a.GroupCount(); i++ {
			group := a.Group(i)

			for _, file := range group.Files {
				if newpath, ok := flagReplace[file.Name]; ok {
					if newpath == "" {
						a.ReplaceFile(file, nil, 0)
					} else {
						newfile, err := os.Open(newpath)
						ragequit(newpath, err)

						st, err := newfile.Stat()
						ragequit(newpath, err)

						if st.Size() > int64(^uint32(0)) {
							ragequit(newpath, errors.New("file too large"))
						}

						a.ReplaceFile(file, newfile, uint32(st.Size()))
					}
				}
			}
		}

		fmt.Fprintf(os.Stderr, "Writing to archive `%s`...\n", flagWrite)
		writer := bufio.NewWriter(ofile)
		a.Write(writer)
		writer.Flush()
		ofile.Close()
	}
}
