package main

import (
	"fmt"
	"os"
	"flag"
	"path"
	"bufio"
	"errors"
	"io/ioutil"
	"aaronlindsay.com/go/pkg/pso2/ice"
	"aaronlindsay.com/go/pkg/pso2/text"
	"aaronlindsay.com/go/pkg/pso2/util"
	"aaronlindsay.com/go/pkg/pso2/trans"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pso2-trans [flags] db.sqlite [...]")
	flag.PrintDefaults()
	os.Exit(2)
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

func ragequit(apath string, err error) {
	if complain(apath, err) {
		os.Exit(1)
	}
}

func main() {
	var flagImport int
	var flagTrans string
	var flagBackup string
	var flagStrip string

	flag.Usage = usage
	flag.IntVar(&flagImport, "i", 0, "import files with the specified version")
	flag.StringVar(&flagTrans, "t", "", "translation name")
	flag.StringVar(&flagBackup, "b", "", "backup files to this path before modifying them")
	flag.StringVar(&flagStrip, "s", "", "write out a stripped database")
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
				af.Close()
				continue
			}

			var a *trans.Archive
			var translation *trans.Translation

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
								break
							}

							if s != nil {
								if s.Value != p.String {
									if flagTrans != "" {
										if translation == nil {
											translation, err = db.QueryTranslation(flagTrans)
											if translation == nil {
												translation, err = db.InsertTranslation(flagTrans)
												if complain(flagTrans, err) {
													break
												}
											}
										}

										ts, _ := db.QueryTranslationString(translation, s)
										if ts != nil {
											_, err = db.UpdateTranslationString(ts, p.String)
										} else {
											_, err = db.InsertTranslationString(translation, s, p.String)
										}
									} else {
										_, err = db.UpdateString(s, flagImport, p.String)
									}

									if complain(f.Name + ": " + p.Identifier, err) {
										break
									}
								}
							} else {
								if flagTrans != "" {
									complain(f.Name + ": " + p.Identifier + ": " + p.String, errors.New("translated identifier does not exist"))
								} else {
									_, err := db.InsertString(f, flagImport, collision, p.Identifier, p.String)
									if complain(f.Name + ": " + p.Identifier, err) {
										break
									}
								}
							}
						}
						db.End()
					}
				}
			}

			af.Close()
		}
	} else if flagTrans != "" {
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "no pso2 dir provided")
			return
		}

		if flagBackup != "" {
			err := os.MkdirAll(flagBackup, 0777)
			ragequit(flagBackup, err)
		}

		pso2dir := flag.Arg(1)

		translation, err := db.QueryTranslation(flagTrans)
		ragequit(flagTrans, err)

		archives, err := db.QueryArchivesTranslation(translation)
		ragequit(dbpath, err)

		for _, a := range archives {
			aname := path.Join(pso2dir, a.Name.String())
			fmt.Fprintf(os.Stderr, "Opening archive `%s`...\n", aname)
			af, err := os.OpenFile(aname, os.O_RDONLY, 0);
			if complain(aname, err) {
				continue
			}

			archive, err := ice.NewArchive(util.BufReader(af))
			if complain(aname, err) {
				continue
			}

			files, err := db.QueryFiles(&a)
			if complain(aname, err) {
				continue
			}

			fmt.Fprintf(os.Stderr, "\tParsing text files...\n")
			var textfiles []*os.File
			for _, f := range files {
				tstrings, err := db.QueryTranslationStringsFile(translation, &f)
				if complain(f.Name, err) || len(tstrings) == 0 {
					continue
				}

				fmt.Fprintf(os.Stderr, "\t\t%s\n", f.Name)

				strings := make([]*trans.String, len(tstrings))
				for i, ts := range tstrings {
					strings[i], err = db.QueryStringTranslation(&ts)
				}

				file := archive.FindFile(-1, f.Name)
				if file == nil {
					complain(f.Name, errors.New("file not found"))
				}
				textfile, err := text.NewTextFile(file.Data)

				collisions := make(map[string]int)

				for _, p := range textfile.Pairs {
					collision := collisions[p.Identifier]
					collisions[p.Identifier] = collision + 1

					var ts *trans.TranslationString
					for i, s := range strings {
						if s.Identifier == p.Identifier && s.Collision == collision {
							ts = &tstrings[i]
							break
						}
					}
					if ts == nil {
						continue
					}

					entry := textfile.PairString(&p)
					entry.Text = ts.Translation
				}

				tf, err := ioutil.TempFile("", "")
				if complain(f.Name, err) {
					continue
				}

				fmt.Fprintf(os.Stderr, "\t\t\tRewriting...\n")

				writer := bufio.NewWriter(tf)
				err = textfile.Write(writer)
				writer.Flush()
				if complain(tf.Name(), err) {
					tf.Close()
					os.Remove(tf.Name())
					continue
				}
				pos, _ := tf.Seek(0, 1)
				tf.Seek(0, 0)

				archive.ReplaceFile(file, tf, uint32(pos))
				textfiles = append(textfiles, tf)
			}

			fmt.Fprintf(os.Stderr, "\tWriting modified archive...\n")
			ofile, err := ioutil.TempFile("", "")

			if !complain(aname, err) {
				writer := bufio.NewWriter(ofile)
				err = archive.Write(writer)
				writer.Flush()
				ofile.Close()

				if flagBackup != "" {
					opath := path.Join(flagBackup, path.Base(aname))
					err = os.Rename(aname, opath)
					if err != nil {
						err = util.CopyFile(aname, opath)
					}
				}

				if complain(aname, err) {
					os.Remove(ofile.Name())
				} else {
					err = os.Rename(ofile.Name(), aname)
					if err != nil {
						err = util.CopyFile(ofile.Name(), aname)
					}
					complain(aname, err)
				}
			}


			for _, tf := range textfiles {
				tf.Close()
				os.Remove(tf.Name())
			}

			af.Close()
		}
	}

	db.Close()

	if flagStrip != "" {
		err = util.CopyFile(dbpath, flagStrip)
		ragequit(flagStrip, err)

		db, err := trans.NewDatabase(flagStrip)
		ragequit(flagStrip, err)

		err = db.Strip()
		ragequit(flagStrip, err)

		db.Close()
	}
}
