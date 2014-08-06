package cmd

import (
	"io"
	"os"
	"path"
	"time"
	"sync"
	"bufio"
	"bytes"
	"errors"
	"runtime"
	"net/url"
	"os/exec"
	"io/ioutil"
	"crypto/md5"
	"encoding/json"
	"github.com/cheggaaa/pb"
	"aaronlindsay.com/go/pkg/pso2/download"
	"aaronlindsay.com/go/pkg/pso2/util"
)

const (
	PathPatchlist = "patchlist.txt"
	PathPatchlistOld = "patchlist-old.txt"
	PathPatchlistInstalled = "patchlist-installed.txt"
	PathVersion = "version.ver"
	PathVersionInstalled = "version-installed.ver"
	PathTranslateDll = "translate.dll"
	PathTranslationBin = "translation.bin"
	PathEnglishDb = "english.db"
	PathTranslationCfg = "translation.cfg"
	EnglishUpdateURL = "http://aaronlindsay.com/pso2/download.json"
)

func PathScratch(pso2path string) string {
	return path.Join(pso2path, "download")
}

func CommitInstalled(pso2path string, patchlist *download.PatchList) error {
	scratch := PathScratch(pso2path)

	pathPatchlistInstalled := path.Join(scratch, PathPatchlistInstalled)
	f, err := os.Create(pathPatchlistInstalled)
	if err != nil {
		return err
	}
	err = patchlist.Write(f)
	f.Close()
	if err != nil {
		return err
	}

	pathVersionInstalled := path.Join(scratch, PathVersionInstalled)
	err = util.CopyFile(path.Join(scratch, PathVersion), pathVersionInstalled)

	return err
}

func LoadVersion(r io.Reader) (string, error) {
	ver, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(ver), nil
}

func DownloadProductionVersion() (s string, err error) {
	resp, err := download.Request(download.ProductionVersion)
	if err != nil {
		return
	}

	s, err = LoadVersion(resp.Body)
	resp.Body.Close()
	return
}

func LoadVersionFile(filename string) (s string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}

	s, err = LoadVersion(f)
	f.Close()
	return
}

func LoadPatchlistFile(filename, urlStr string) (p *download.PatchList, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}

	p, err = download.ParseListCap(bufio.NewReader(f), urlStr, 20000)
	f.Close()
	return
}

func LoadPatchlist(pso2path string) (patchlist *download.PatchList, err error) {
	scratch := PathScratch(pso2path)

	patchlist, err = LoadPatchlistFile(path.Join(scratch, PathPatchlist), download.ProductionPatchlist)
	if err != nil {
		return
	}

	patchlistOld, err := LoadPatchlistFile(path.Join(scratch, PathPatchlistOld), download.ProductionPatchlistOld)
	if err != nil {
		return
	}

	patchlist = patchlist.MergeOld(patchlistOld)

	return
}

func DownloadPatchlist(pso2path, version string) (patchlist *download.PatchList, err error) {
	scratch := PathScratch(pso2path)

	pathPatchlist := path.Join(scratch, PathPatchlist)
	pathPatchlistOld := path.Join(scratch, PathPatchlistOld)
	pathVersion := path.Join(scratch, PathVersion)

	patchlist, err = download.DownloadList(download.ProductionPatchlist)
	if err != nil {
		return
	}

	patchlistOld, err := download.DownloadList(download.ProductionPatchlistOld)
	if err != nil {
		return
	}

	launcherlist, err := download.DownloadList(download.ProductionLauncherlist)
	if err != nil {
		return
	}

	patchlist.Append(launcherlist)

	f, err := os.Create(pathPatchlist)
	if err != nil {
		return
	}
	err = patchlist.Write(f)
	f.Close()
	if err != nil {
		return
	}

	f, err = os.Create(pathPatchlistOld)
	if err != nil {
		return
	}
	err = patchlistOld.Write(f)
	f.Close()
	if err != nil {
		return
	}

	patchlist = patchlist.MergeOld(patchlistOld)

	err = ioutil.WriteFile(pathVersion, []byte(version), 0666)

	return
}

func DownloadEnglishFiles(pso2path string) (translationChanged bool, err error) {
	resp, err := download.Request(EnglishUpdateURL)
	if err != nil {
		return
	}

	var data struct {
		ItemTimestamp, EnglishTimestamp, TranslationTimestamp int64
		ItemURL, EnglishURL, TranslationURL string
	}

	d := json.NewDecoder(resp.Body)
	err = d.Decode(&data)
	resp.Body.Close()

	if err != nil {
		return
	}

	scratch := PathScratch(pso2path)
	itemPath := path.Join(scratch, PathTranslateDll)
	translationPath := path.Join(scratch, PathTranslationBin)
	englishPath := path.Join(scratch, PathEnglishDb)
	rootUrl, _ := url.Parse(EnglishUpdateURL)

	updateEnglishFile := func(path string, timestamp int64, urlStr string) (bool, error) {
		url, err := url.Parse(urlStr)
		if err != nil {
			return false, err
		}

		st, err := os.Stat(path)

		if os.IsNotExist(err) || (err == nil && st.ModTime().Unix() < timestamp) {
			resp, err := download.Request(rootUrl.ResolveReference(url).String())
			if err != nil {
				return false, err
			}

			f, err := os.Create(path)
			if err != nil {
				return false, err
			}

			_, err = io.Copy(f, resp.Body)
			f.Close()
			resp.Body.Close()

			return true, err
		}

		return false, err
	}

	translationChanged, err = updateEnglishFile(englishPath, data.EnglishTimestamp, data.EnglishURL)
	if err != nil {
		return
	}

	_, err = updateEnglishFile(translationPath, data.TranslationTimestamp, data.TranslationURL)
	if err != nil {
		return
	}

	_, err = updateEnglishFile(itemPath, data.ItemTimestamp, data.ItemURL)

	return
}

func LoadTranslationConfig(pso2dir string) (t TranslationConfig, err error) {
	f, err := os.Open(path.Join(pso2dir, PathTranslationCfg))
	if err != nil {
		return
	}

	t, err = NewTranslationConfig(f)
	f.Close()

	return
}

func SaveTranslationConfig(pso2dir string, t TranslationConfig) (err error) {
	f, err := os.Create(path.Join(pso2dir, PathTranslationCfg))
	if err != nil {
		return
	}

	err = t.Write(f)
	f.Close()

	return
}

func CheckFiles(pso2path string, checkHash bool, patches *download.PatchList) (changes []*download.PatchEntry, err error) {
	pbar := pb.New(len(patches.Entries))
	pbar.SetRefreshRate(time.Second / 30)
	pbar.Start()

	for i := range patches.Entries {
		e := &patches.Entries[i]
		filepath := path.Join(pso2path, download.RemoveExtension(e.Path))

		st, err := os.Stat(filepath)

		if os.IsNotExist(err) {
			changes = append(changes, e)
		} else {
			if err != nil {
				return nil, err
			}

			if st.Size() != e.Size {
				changes = append(changes, e)
			} else if checkHash {
				f, err := os.Open(filepath)
				if err != nil {
					return nil, err
				}
				h := md5.New()
				_, err = io.Copy(h, f)
				f.Close()
				if err != nil {
					return nil, err
				}
				if bytes.Compare(h.Sum(nil), e.MD5[:]) != 0 {
					changes = append(changes, e)
				}
			}
		}

		pbar.Increment()
		runtime.Gosched()
	}

	pbar.Finish()

	return
}

func DownloadChanges(pso2path string, changes []*download.PatchEntry, parallel int) (errs []error) {
	if parallel <= 0 {
		parallel = 1
	}

	changesSize := int64(0)
	for _, e := range changes {
		changesSize += e.Size
	}

	pbar := pb.New64(changesSize)
	pbar.SetUnits(pb.U_BYTES)
	pbar.SetRefreshRate(time.Second / 30)
	pbar.ShowSpeed = true
	pbar.Start()

	queue := make(chan *download.PatchEntry)
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
			h := md5.New()

			for {
				e, ok := <-queue
				if !ok {
					break
				}

				filepath := path.Join(pso2path, download.RemoveExtension(e.Path))

				err := os.MkdirAll(path.Dir(filepath), 0777)
				if complain(err) {
					break
				}

				pathUrl, err := e.URL()
				if complain(err) {
					continue
				}

				resp, err := download.Request(pathUrl.String())
				if complain(err) {
					continue
				}

				if resp.StatusCode != 200 {
					complain(errors.New(pathUrl.String() + ": " + resp.Status))
					continue
				}

				if resp.ContentLength >= 0 && resp.ContentLength != e.Size {
					resp.Body.Close()
					complain(errors.New(e.Path + ": invalid file size"))
					continue
				}

				f, err := os.Create(filepath)
				if complain(err) {
					resp.Body.Close()
					continue
				}

				h.Reset()
				n, err := io.Copy(io.MultiWriter(f, h, pbar), resp.Body)

				resp.Body.Close()
				f.Close()

				if !complain(err) {
					if n != e.Size {
						complain(errors.New(pathUrl.String() + ": download finished prematurely"))
					} else if bytes.Compare(h.Sum(nil), e.MD5[:]) != 0 {
						complain(errors.New(pathUrl.String() + ": download hash mismatch"))
					}
				}
			}

			done <-true
		}()
	}

	for _, e := range changes {
		queue <-e
	}
	close(queue)

	for i := 0; i < parallel; i++ {
		<-done
	}

	pbar.Finish()

	return
}

func PruneFiles(pso2path string, patchlist *download.PatchList) (size int64, err error) {
	win32 := path.Join(pso2path, "data/win32")

	f, err := os.Open(win32)
	if err != nil {
		return
	}

	for err == nil {
		var files []os.FileInfo
		files, err = f.Readdir(0x80)
		for _, f := range files {
			if f.IsDir() {
				continue
			}

			e := patchlist.EntryMap[f.Name() + ".pat"]

			if e == nil {
				size += f.Size()
				win32 := path.Join(win32, f.Name())
				err = os.Remove(win32)
				if err != nil {
					break
				}
			}
		}
	}

	f.Close()

	if err == io.EOF {
		err = nil
	}

	return
}

func LaunchGame(pso2path string) (cmd *exec.Cmd, err error) {
	cmd = exec.Command("./pso2.exe", "+0x33aca2b9", "-pso2")
	cmd.Env = append(os.Environ(), "-pso2=+0x01e3f1e9")
	cmd.Dir = pso2path
	err = cmd.Start()

	return
}
