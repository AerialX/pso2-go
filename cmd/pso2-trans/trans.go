package main

import (
	"fmt"
	"os"
	"flag"
	"path"
	"aaronlindsay.com/go/pkg/pso2/ice"
	"aaronlindsay.com/go/pkg/pso2/text"
	"aaronlindsay.com/go/pkg/pso2/util"
	"aaronlindsay.com/go/pkg/pso2/trans"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: trans.go [flags] db.sqlite [...]")
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

func complain(apath string, err error) bool {
	if err != nil {
		if apath != "" {
			fmt.Fprintf(os.Stderr, "error with file `%s`\n", apath)
		}
		fmt.Fprintln(os.Stderr, err);
		return true
	}

	return false
}

func main() {
	var flagPrint bool
	var flagWrite string
	var flagImport int
	var flagTrans string

	flag.Usage = usage
	flag.BoolVar(&flagPrint, "p", false, "print details about the file")
	flag.IntVar(&flagImport, "i", 0, "import files with the specified version")
	flag.StringVar(&flagTrans, "t", "eng", "translation name")
	flag.StringVar(&flagWrite, "w", "", "write a repacked file")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "no database provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	dbpath := flag.Arg(0)
	fmt.Fprintf(os.Stderr, "Opening database `%s`...\n", dbpath)
	db, err := trans.NewDatabase(dbpath)
	ragequit(dbpath, err)

	if flagImport != 0 {
		for i := 1; i < flag.NArg(); i++ {
			name := flag.Arg(i)
			aname, err := trans.ArchiveNameFromString(path.Base(name))
			if complain(name, err) {
				continue
			}

			fmt.Fprintf(os.Stderr, "Opening archive `%s`...\n", name)
			af, err := os.OpenFile(name, os.O_RDONLY, 0);
			if complain(name, err) {
				continue
			}

			archive, err := ice.NewArchive(util.BufReader(af))
			if complain(name, err) {
				continue
			}

			var a *trans.Archive

			for i := 0; i < archive.GroupCount(); i++ {
				group := archive.Group(i)

				for _, file := range group.Files {
					if file.Type == "text" {
						fmt.Fprintf(os.Stderr, "Importing file `%s`...\n", file.Name)

						t, err := text.NewTextFile(file.Data)
						if complain(file.Name, err) {
							continue
						}

						if a == nil {
							a, err = db.QueryArchive(aname)
							if complain(name, err) {
								continue
							}

							if a == nil {
								a, err = db.InsertArchive(aname)
								if complain(name, err) {
									continue
								}
							}
						}

						f, err := db.QueryFile(a, file.Name)
						if complain(file.Name, err) {
							continue
						}

						if f == nil {
							f, err = db.InsertFile(a, file.Name)
							if complain(file.Name, err) {
								continue
							}
						}

						collisions := make(map[string]int)

						db.Begin()
						for _, p := range t.Pairs {
							collision := collisions[p.Identifier]
							collisions[p.Identifier] = collision + 1

							s, err := db.QueryString(f, collision, p.Identifier)
							if complain(f.Name + ": " + p.Identifier, err) {
								db.End()
								continue
							}

							if s != nil {
								if s.Value != p.String {
									_, err := db.UpdateString(s, flagImport, p.String)
									if complain(f.Name + ": " + p.Identifier, err) {
										db.End()
										continue
									}
								}
							} else {
								_, err := db.InsertString(f, flagImport, collision, p.Identifier, p.String)
								if complain(f.Name + ": " + p.Identifier, err) {
									db.End()
									continue
								}
							}
						}
						db.End()
					}
				}
			}
		}
	}

	db.Close()

	/*if flagPrint {
		for i, entry := range t.Entries {
			fmt.Printf("%08x: %s\n", entry.Value, entry.Text)

			if entry.TextStatus == text.TextEntryString {
				t.Entries[i].Text = "LOLOLOL"
			}
		}
	}

	if flagWrite != "" {
		ofile, err := os.Create(flagWrite)
		ragequit(flagWrite, err)

		fmt.Fprintf(os.Stderr, "Writing to `%s`...\n", flagWrite)
		t.Write(ofile)
	}*/
}
