package main

import (
	"fmt"
	"io"
	"os"
	"flag"
	"path"
	"time"
	"bufio"
	"bytes"
	"errors"
	"runtime"
	"io/ioutil"
	"crypto/md5"
	"github.com/cheggaaa/pb"
	"aaronlindsay.com/go/pkg/pso2/download"
	"aaronlindsay.com/go/pkg/pso2/util"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pso2-download [flags] pso2/root/path")
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

const (
	pathPatchlist = "patchlist.txt"
	pathPatchlistOld = "patchlist-old.txt"
	pathPatchlistInstalled = "patchlist-installed.txt"
	pathPatchlistOldInstalled = "patchlist-old-installed.txt"
	pathVersion = "version.ver"
	pathVersionInstalled = "version-installed.ver"
)

func success(scratch string) {
	pathPatchlistInstalled := path.Join(scratch, pathPatchlistInstalled)
	err := util.CopyFile(path.Join(scratch, pathPatchlist), pathPatchlistInstalled)
	ragequit(pathPatchlistInstalled, err)

	pathPatchlistOldInstalled := path.Join(scratch, pathPatchlistOldInstalled)
	err = util.CopyFile(path.Join(scratch, pathPatchlistOld), pathPatchlistOldInstalled)
	ragequit(pathPatchlistOldInstalled, err)

	pathVersionInstalled := path.Join(scratch, pathVersionInstalled)
	err = util.CopyFile(path.Join(scratch, pathVersion), pathVersionInstalled)
	ragequit(pathVersionInstalled, err)
}

func main() {
	var flagPrint, flagAll, flagCheck, flagHash, flagDownload, flagUpdate bool
	var flagParallel int

	flag.Usage = usage
	flag.BoolVar(&flagAll, "a", false, "ignore patchlist, check all files")
	flag.BoolVar(&flagCheck, "c", false, "check files")
	flag.BoolVar(&flagHash, "h", false, "check file hashes")
	flag.BoolVar(&flagDownload, "d", false, "download files")
	flag.IntVar(&flagParallel, "j", 3, "max parallel downloads")
	flag.BoolVar(&flagPrint, "p", false, "print verbose information")
	flag.BoolVar(&flagUpdate, "u", false, "update patchlist")
	flag.Parse()

	if flagHash {
		flagCheck = true
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "no pso2 path provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	var err error

	pso2path := flag.Arg(0)
	scratch := path.Join(pso2path, "download")

	err = os.Mkdir(scratch, 0777)
	if !os.IsExist(err) {
		ragequit(scratch, err)
	}

	var version, installedVersion, netVersion string

	fmt.Fprintln(os.Stderr, "Checking version...")
	resp, err := download.Request(download.ProductionVersion)
	if !complain(download.ProductionVersion, err) {
		ver, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if !complain(download.ProductionVersion, err) {
			netVersion = string(ver)
		}
	}

	pathVersion := path.Join(scratch, pathVersion)
	f, err := os.Open(pathVersion)
	if !complain(pathVersion, err) {
		ver, err := ioutil.ReadAll(f)
		f.Close()
		if !complain(pathVersion, err) {
			version = string(ver)
		}
	}

	pathVersionInstalled := path.Join(scratch, pathVersionInstalled)
	f, err = os.Open(pathVersionInstalled)
	if !complain(pathVersionInstalled, err) {
		ver, err := ioutil.ReadAll(f)
		f.Close()
		if !complain(pathVersionInstalled, err) {
			installedVersion = string(ver)
			fmt.Fprintln(os.Stderr, "Current version:", installedVersion)
		}
	}

	if netVersion != "" && version != netVersion {
		fmt.Fprintln(os.Stderr, "Update", netVersion, "found")
		flagUpdate = true
	}

	var patchlist, patchlistOld, installedPatchlist, installedPatchlistOld, patchdiff *download.PatchList

	fmt.Fprintln(os.Stderr, "Loading patchlist...")
	pathPatchlist := path.Join(scratch, pathPatchlist)
	f, err = os.Open(pathPatchlist)
	if !complain(pathPatchlist, err) {
		patchlist, err = download.ParseListCap(bufio.NewReader(f), download.ProductionPatchlist, 20000)
		f.Close()
		complain(pathPatchlist, err)
	}

	pathPatchlistOld := path.Join(scratch, pathPatchlistOld)
	f, err = os.Open(pathPatchlistOld)
	if !complain(pathPatchlist, err) {
		patchlistOld, err = download.ParseListCap(bufio.NewReader(f), download.ProductionPatchlistOld, 20000)
		f.Close()
		complain(pathPatchlistOld, err)
	}

	if flagAll {
		fmt.Fprintln(os.Stderr, "Ignoring patchlist, checking all files...")
	} else {
		pathPatchlistInstalled := path.Join(scratch, pathPatchlistInstalled)
		f, err = os.Open(pathPatchlistInstalled)
		if !complain(pathPatchlistInstalled, err) {
			installedPatchlist, err = download.ParseListCap(bufio.NewReader(f), download.ProductionPatchlist, 20000)
			f.Close()
			complain(pathPatchlistInstalled, err)
		}

		pathPatchlistOldInstalled := path.Join(scratch, pathPatchlistOldInstalled)
		f, err = os.Open(pathPatchlistOldInstalled)
		if !complain(pathPatchlistInstalled, err) {
			installedPatchlistOld, err = download.ParseListCap(bufio.NewReader(f), download.ProductionPatchlistOld, 20000)
			f.Close()
			complain(pathPatchlistOldInstalled, err)
		}
	}

	if flagUpdate {
		fmt.Fprintln(os.Stderr, "Downloading patchlist.txt...")
		patchlist, err = download.DownloadList(download.ProductionPatchlist)
		ragequit(download.ProductionPatchlist, err)

		patchlistOld, err = download.DownloadList(download.ProductionPatchlistOld)
		ragequit(download.ProductionPatchlistOld, err)

		fmt.Fprintln(os.Stderr, "Downloading launcherlist.txt...")
		launcherlist, err := download.DownloadList(download.ProductionLauncherlist)
		ragequit(download.ProductionLauncherlist, err)

		patchlist.Append(launcherlist)

		f, err := os.Create(pathPatchlist)
		ragequit(pathPatchlist, err)
		err = patchlist.Write(f)
		f.Close()
		ragequit(pathPatchlist, err)

		f, err = os.Create(pathPatchlistOld)
		ragequit(pathPatchlistOld, err)
		err = patchlistOld.Write(f)
		f.Close()
		ragequit(pathPatchlistOld, err)

		version = netVersion
		err = ioutil.WriteFile(pathVersion, []byte(version), 0666)
	}

	if patchlist == nil {
		ragequit(pathPatchlist, errors.New("no patchlist found - use the -u flag to download one"))
	}

	patchlist = patchlist.MergeOld(patchlistOld)
	installedPatchlist = installedPatchlist.MergeOld(installedPatchlistOld)

	patchdiff = patchlist.Diff(installedPatchlist)

	fmt.Fprintln(os.Stderr, len(patchdiff.Entries), "file(s) changed since last update")

	if flagPrint {
		for _, e := range patchdiff.Entries {
			fmt.Fprintf(os.Stderr, "\t%s (0x%08x): %x\n", download.RemoveExtension(e.Path), e.Size, e.MD5)
		}
	}

	var changes []*download.PatchEntry
	if flagCheck && len(patchdiff.Entries) > 0 {
		fmt.Fprintln(os.Stderr, "Checking files...")

		pbar := pb.New(len(patchdiff.Entries))
		pbar.SetRefreshRate(time.Second / 30)
		pbar.Start()

		for i := range patchdiff.Entries {
			e := &patchdiff.Entries[i]
			filepath := path.Join(pso2path, download.RemoveExtension(e.Path))

			st, err := os.Stat(filepath)

			if os.IsNotExist(err) {
				changes = append(changes, e)
			} else {
				ragequit(filepath, err)

				if st.Size() != e.Size {
					changes = append(changes, e)
				} else if flagHash {
					f, err := os.Open(filepath)
					ragequit(filepath, err)
					h := md5.New()
					_, err = io.Copy(h, f)
					f.Close()
					ragequit(filepath, err)
					if bytes.Compare(h.Sum(nil), e.MD5[:]) != 0 {
						changes = append(changes, e)
					}
				}
			}

			pbar.Increment()
			runtime.Gosched()
		}

		pbar.Finish()

		if len(changes) == 0 {
			success(scratch)
		}
	} else {
		for i := range patchdiff.Entries {
			changes = append(changes, &patchdiff.Entries[i])
		}
	}

	changesSize := int64(0)
	for _, e := range changes {
		changesSize += e.Size
	}

	fmt.Fprintf(os.Stderr, "%d file(s) (%0.2f MB) need updating\n", len(changes), float32(changesSize) / 1024 / 1024)

	if flagPrint {
		for _, e := range changes {
			fmt.Fprintf(os.Stderr, "\t%s (0x%08x): %x\n", download.RemoveExtension(e.Path), e.Size, e.MD5)
		}
	}

	if flagDownload && len(changes) > 0 {
		errorCount := 0

		pbar := pb.New64(changesSize)
		pbar.SetUnits(pb.U_BYTES)
		pbar.SetRefreshRate(time.Second / 30)
		pbar.ShowSpeed = true
		pbar.Start()

		queue := make(chan *download.PatchEntry, flagParallel)
		done := make(chan bool)

		go func() {
			complain := func(apath string, err error) bool {
				if complain(apath, err) {
					errorCount++
					return true
				}

				return false
			}

			h := md5.New()

			for {
				e, ok := <-queue
				if !ok {
					break
				}

				filepath := path.Join(pso2path, download.RemoveExtension(e.Path))

				err := os.MkdirAll(path.Dir(filepath), 0777)
				ragequit(path.Dir(filepath), err)

				pathUrl, err := e.URL()
				ragequit(e.Path, err)

				resp, err := download.Request(pathUrl.String())
				if complain(pathUrl.String(), err) {
					continue
				}

				if resp.StatusCode != 200 {
					complain(pathUrl.String(), errors.New(resp.Status))
					continue
				}

				if resp.ContentLength >= 0 && resp.ContentLength != e.Size {
					resp.Body.Close()
					complain(pathUrl.String(), errors.New("invalid file size"))
					continue
				}

				f, err := os.Create(filepath)
				if complain(filepath, err) {
					resp.Body.Close()
					continue
				}

				h.Reset()
				n, err := io.Copy(io.MultiWriter(f, h, pbar), resp.Body)

				resp.Body.Close()
				f.Close()

				if !complain(filepath, err) {
					if n != e.Size {
						complain(pathUrl.String(), errors.New("download finished prematurely"))
					} else if bytes.Compare(h.Sum(nil), e.MD5[:]) != 0 {
						complain(pathUrl.String(), errors.New("download hash mismatch"))
					}
				}
			}

			done <-true
		}()

		for _, e := range changes {
			queue <- e
		}
		close(queue)
		<-done

		pbar.Finish()

		if errorCount > 0 {
			fmt.Fprintln(os.Stderr, "Update unsuccessful, errors encountered")
		} else {
			fmt.Fprintln(os.Stderr, "Update complete!")
			success(scratch)
		}
	}
}
