package main

import (
	"fmt"
	"os"
	"flag"
	"path"
	"strings"
	"runtime"
	"io/ioutil"
	"aaronlindsay.com/go/pkg/pso2/trans"
	transcmd "aaronlindsay.com/go/pkg/pso2/trans/cmd"
	"aaronlindsay.com/go/pkg/pso2/download"
	"aaronlindsay.com/go/pkg/pso2/download/cmd"
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
		fmt.Fprintln(os.Stderr, err)
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
	var flagPrint, flagAll, flagCheck, flagHash, flagDownload, flagUpdate, flagGarbage, flagLaunch, flagItemTranslation, flagBackup bool
	var flagTranslate, flagPublicKey, flagDumpPublicKey string
	var flagParallel int

	flag.Usage = usage
	flag.BoolVar(&flagAll, "a", false, "ignore patchlist, check all files")
	flag.BoolVar(&flagCheck, "c", false, "check files")
	flag.BoolVar(&flagHash, "h", false, "check file hashes")
	flag.BoolVar(&flagDownload, "d", false, "download files")
	flag.IntVar(&flagParallel, "p", 3, "max parallel downloads")
	flag.BoolVar(&flagPrint, "v", false, "print verbose information")
	flag.BoolVar(&flagUpdate, "u", false, "refresh patchlist")
	flag.BoolVar(&flagGarbage, "g", false, "clean up old/unused files")
	flag.BoolVar(&flagBackup, "b", false, "back up any modified files")
	flag.BoolVar(&flagItemTranslation, "i", false, "enable the item translation")
	flag.BoolVar(&flagLaunch, "l", false, "launch the game")
	flag.StringVar(&flagTranslate, "t", "", "use the translation with the specified comma-separated names (example: eng,story-eng)")
	flag.StringVar(&flagPublicKey, "pubkey", "", "inject a public key (path relative to pso2_bin)")
	flag.StringVar(&flagDumpPublicKey, "dumppubkey", "", "dump the PSO2 public key (path relative to pso2_bin)")
	flag.Parse()

	maxprocs := runtime.GOMAXPROCS(0)
	if maxprocs < 0x10 {
		runtime.GOMAXPROCS(0x10)
	}

	if flagHash {
		flagCheck = true
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "no pso2 path provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	pso2path := flag.Arg(0)

	scratch := cmd.PathScratch(pso2path)
	err := os.Mkdir(scratch, 0777)
	if !os.IsExist(err) {
		ragequit(scratch, err)
	}

	flagTranslations := strings.Split(flagTranslate, ",")
	if flagTranslate == "" {
		flagTranslations = nil
	}

	fmt.Fprintln(os.Stderr, "Checking for updates...")
	netVersion, err := cmd.DownloadProductionVersion()
	complain(download.ProductionVersion, err)

	version, _ := cmd.LoadVersionFile(path.Join(scratch, cmd.PathVersion))

	needsTranslation, err := cmd.DownloadEnglishFiles(pso2path)
	complain("", err)

	installedVersion, _ := cmd.LoadVersionFile(path.Join(scratch, cmd.PathVersionInstalled))
	if installedVersion != "" {
		fmt.Fprintln(os.Stderr, "Current version:", installedVersion)
	}

	if netVersion != "" && version != netVersion {
		fmt.Fprintln(os.Stderr, "Update", netVersion, "found")
		flagUpdate = true
	}

	fmt.Fprintln(os.Stderr, "Loading patchlist...")
	patchlist, err := cmd.LoadPatchlistFile(path.Join(scratch, cmd.PathPatchlist))

	var installedPatchlist *download.PatchList
	if !flagAll {
		installedPatchlist, _ = cmd.LoadPatchlistFile(path.Join(scratch, cmd.PathPatchlistInstalled))
		needsTranslation = true
	} else {
		fmt.Fprintln(os.Stderr, "Ignoring patchlist, checking all files...")
	}

	if flagUpdate || patchlist == nil {
		fmt.Fprintln(os.Stderr, "Downloading patchlist.txt...")
		patchlist, err = cmd.DownloadPatchlist(pso2path, netVersion)
		ragequit(download.ProductionPatchlist, err)
	}

	patchdiff := patchlist.Diff(installedPatchlist)

	if installedPatchlist == nil {
		flagCheck = true
	}

	fmt.Fprintln(os.Stderr, len(patchdiff.Entries), "file(s) changed since last update")

	if flagPrint {
		for _, e := range patchdiff.Entries {
			fmt.Fprintf(os.Stderr, "\t%s (0x%08x): %x\n", download.RemoveExtension(e.Path), e.Size, e.MD5)
		}
	}

	var changes []*download.PatchEntry
	if flagCheck && len(patchdiff.Entries) > 0 {
		fmt.Fprintln(os.Stderr, "Checking files...")
		changes, err = cmd.CheckFiles(pso2path, flagHash, patchdiff)
		ragequit("", err)

		if len(changes) == 0 {
			cmd.CommitInstalled(pso2path)
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
		errors := cmd.DownloadChanges(pso2path, changes, flagParallel)

		if len(errors) > 0 {
			for err := range errors {
				fmt.Fprintln(os.Stderr, err)
			}
			fmt.Fprintln(os.Stderr, "Update unsuccessful, errors encountered")
		} else {
			fmt.Fprintln(os.Stderr, "Update complete!")
			cmd.CommitInstalled(pso2path)
			needsTranslation = true
		}
	}

	if flagGarbage {
		fmt.Fprintln(os.Stderr, "Deleting old, unused files...")

		garbageSize, err := cmd.PruneFiles(pso2path, patchlist)
		complain("", err)

		fmt.Fprintf(os.Stderr, "Done! Saved %0.2f MB of space.\n", float32(garbageSize) / 1024 / 1024)
	}

	if needsTranslation && len(flagTranslations) > 0 {
		fmt.Fprintln(os.Stderr, "Applying english patches...")
		db, err := trans.NewDatabase(path.Join(scratch, cmd.PathEnglishDb))
		backupPath := ""
		if flagBackup {
			backupPath = path.Join(scratch, "backup")
			err = os.MkdirAll(backupPath, 0777)
			if complain(backupPath, err) {
				backupPath = ""
			}
		}
		if !complain("", err) {
			var errs []error
			for _, translation := range flagTranslations {
				errs = append(errs, transcmd.PatchFiles(db, path.Join(pso2path, "data/win32"), translation, backupPath, pso2path, runtime.NumCPU() + 1)...)
			}

			for _, err := range errs {
				complain("", err)
			}
		}
		db.Close()
	}

	if flagLaunch {
		config, _ := cmd.LoadTranslationConfig(pso2path)
		configChanged := false

		if config == nil {
			config = make(cmd.TranslationConfig)
		}

		setConfig := func(key, value string) {
			if config[key] != value {
				config[key] = value
				configChanged = true
			}
		}

		if flagItemTranslation {
			setConfig(cmd.ConfigTranslationPath, path.Join("download", cmd.PathTranslationBin))
		} else {
			setConfig(cmd.ConfigTranslationPath, "")
		}

		if flagDumpPublicKey != "" {
			setConfig(cmd.ConfigPublicKeyPath, flagDumpPublicKey)
			setConfig(cmd.ConfigPublicKeyDump, "1")
		} else {
			setConfig(cmd.ConfigPublicKeyPath, flagPublicKey)
			setConfig(cmd.ConfigPublicKeyDump, "")
		}

		if configChanged {
			cmd.SaveTranslationConfig(pso2path, config)
		}

		fmt.Fprintln(os.Stderr, "Launching PSO2...")

		command, err := cmd.LaunchGame(pso2path)
		ragequit("", err)

		loadDll := flagItemTranslation || flagPublicKey != "" || flagDumpPublicKey != ""
		ddraw := path.Join(pso2path, "ddraw.dll")

		if loadDll {
			err = ioutil.WriteFile(ddraw, cmd.DdrawDll[:], 0777)
			complain(ddraw, err)
		}

		err = command.Wait()

		if loadDll {
			err = os.Remove(ddraw)
			complain(ddraw, err)
		}

		ragequit("", err)
	}
}
