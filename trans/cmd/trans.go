package cmd

import (
	"os"
	"fmt"
	"path"
	"time"
	"bufio"
	"errors"
	"io/ioutil"
	"github.com/cheggaaa/pb"
	"aaronlindsay.com/go/pkg/pso2/ice"
	"aaronlindsay.com/go/pkg/pso2/text"
	"aaronlindsay.com/go/pkg/pso2/util"
	"aaronlindsay.com/go/pkg/pso2/trans"
)

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

func StripDatabase(dbpath, flagStrip string) {
	err := util.CopyFile(dbpath, flagStrip)
	ragequit(flagStrip, err)

	db, err := trans.NewDatabase(flagStrip)
	ragequit(flagStrip, err)

	err = db.Strip()
	ragequit(flagStrip, err)

	db.Close()
}

func PatchFiles(db *trans.Database, dbpath, pso2dir, flagTrans, flagBackup, flagOutput string, flagParallel int) {
	translation, err := db.QueryTranslation(flagTrans)
	ragequit(flagTrans, err)

	archives, err := db.QueryArchivesTranslation(translation)
	ragequit(dbpath, err)

	pbar := pb.New(len(archives))
	pbar.SetRefreshRate(time.Second / 10)
	pbar.Start()

	queue := make(chan *trans.Archive)
	done := make(chan bool)

	for i := 0; i < flagParallel; i++ {
		go func() {
			for {
				a, ok := <-queue
				if !ok {
					break
				}

				aname := path.Join(pso2dir, a.Name.String())
				af, err := os.OpenFile(aname, os.O_RDONLY, 0);
				if complain(aname, err) {
					continue
				}

				archive, err := ice.NewArchive(util.BufReader(af))
				if complain(aname, err) {
					continue
				}

				files, err := db.QueryFiles(a)
				if complain(aname, err) {
					continue
				}

				fileDirty := false

				var textfiles []*os.File
				for _, f := range files {
					tstrings, err := db.QueryTranslationStringsFile(translation, &f)
					if complain(f.Name, err) || len(tstrings) == 0 {
						continue
					}

					strings := make([]*trans.String, len(tstrings))
					for i, ts := range tstrings {
						strings[i], err = db.QueryStringTranslation(&ts)
					}

					file := archive.FindFile(-1, f.Name)
					if file == nil {
						if complain(f.Name, errors.New("file not found")) {
							continue
						}
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

						if p.String != ts.Translation {
							entry := textfile.PairString(&p)
							entry.Text = ts.Translation
							fileDirty = true
						}
					}

					tf, err := ioutil.TempFile("", "")
					if complain(f.Name, err) {
						continue
					}

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

				if fileDirty {
					ofile, err := ioutil.TempFile("", "")

					aname := path.Join(flagOutput, a.Name.String())

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
								os.Remove(ofile.Name())
							}
							complain(aname, err)
						}
					}
				}

				for _, tf := range textfiles {
					tf.Close()
					os.Remove(tf.Name())
				}

				af.Close()

				pbar.Increment()
			}

			done <-true
		}()
	}

	for i := range archives {
		queue <-&archives[i]
	}
	close(queue)

	for i := 0; i < flagParallel; i++ {
		<-done
	}

	pbar.Finish()
}
