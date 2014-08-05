package cmd

import (
	"os"
	"path"
	"time"
	"sync"
	"bufio"
	"errors"
	"io/ioutil"
	"github.com/cheggaaa/pb"
	"aaronlindsay.com/go/pkg/pso2/ice"
	"aaronlindsay.com/go/pkg/pso2/text"
	"aaronlindsay.com/go/pkg/pso2/util"
	"aaronlindsay.com/go/pkg/pso2/trans"
)

func StripDatabase(dbpath, flagStrip string) (err error) {
	err = util.CopyFile(dbpath, flagStrip)
	if err != nil {
		return
	}

	db, err := trans.NewDatabase(flagStrip)
	if err != nil {
		return
	}

	err = db.Strip()

	db.Close()

	return
}

func PatchFiles(db *trans.Database, pso2dir, translationName, backupPath, outputPath string, parallel int) (errs []error) {
	translation, err := db.QueryTranslation(translationName)
	if err == nil && translation == nil {
		err = errors.New("translation not found")
	}
	if err != nil {
		return []error{err}
	}

	archives, err := db.QueryArchivesTranslation(translation)
	if err != nil {
		return []error{err}
	}

	pbar := pb.New(len(archives))
	pbar.SetRefreshRate(time.Second / 10)
	pbar.Start()

	queue := make(chan *trans.Archive)
	done := make(chan bool)

	errlock := sync.Mutex{}

	complain := func(err error) bool {
		if err != nil {
			errlock.Lock()
			errs = append(errs, err)
			errlock.Unlock()
			return true
		}

		return false
	}

	for i := 0; i < parallel; i++ {
		go func() {
			for {
				a, ok := <-queue
				if !ok {
					break
				}

				aname := path.Join(pso2dir, a.Name.String())
				af, err := os.OpenFile(aname, os.O_RDONLY, 0);
				if complain(err) {
					continue
				}

				archive, err := ice.NewArchive(util.BufReader(af))
				if complain(err) {
					continue
				}

				files, err := db.QueryFiles(a)
				if complain(err) {
					continue
				}

				fileDirty := false

				var textfiles []*os.File
				for _, f := range files {
					tstrings, err := db.QueryTranslationStringsFile(translation, &f)
					if complain(err) || len(tstrings) == 0 {
						continue
					}

					strings := make([]*trans.String, len(tstrings))
					for i, ts := range tstrings {
						strings[i], err = db.QueryStringTranslation(&ts)
						complain(err)
					}

					file := archive.FindFile(-1, f.Name)
					if file == nil {
						if complain(errors.New(f.Name + ": file not found")) {
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
					if complain(err) {
						continue
					}

					writer := bufio.NewWriter(tf)
					err = textfile.Write(writer)
					writer.Flush()
					if complain(err) {
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

					aname := path.Join(outputPath, a.Name.String())

					if !complain(err) {
						backupPath := backupPath
						if archive.IsModified() {
							backupPath = ""
						}

						writer := bufio.NewWriter(ofile)
						err = archive.Write(writer)
						writer.Flush()
						ofile.Close()

						if backupPath != "" {
							opath := path.Join(backupPath, path.Base(aname))
							err = os.Rename(aname, opath)
							if err != nil {
								err = util.CopyFile(aname, opath)
							}
						}

						if complain(err) {
							os.Remove(ofile.Name())
						} else {
							err = os.Rename(ofile.Name(), aname)
							if err != nil {
								err = util.CopyFile(ofile.Name(), aname)
								os.Remove(ofile.Name())
							}
							complain(err)
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

	for i := 0; i < parallel; i++ {
		<-done
	}

	pbar.Finish()

	return
}
