package main

import (
	"os"
	"fmt"
	"flag"
	"runtime"
	"aaronlindsay.com/go/pkg/pso2/trans"
	"aaronlindsay.com/go/pkg/pso2/trans/cmd"
)

func usage() {
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, r)
		}
	}()

	var flagTrans, flagBackup, flagOutput string
	var flagParallel int

	flag.Usage = usage
	flag.IntVar(&flagParallel, "p", runtime.NumCPU() + 1, "max parallel tasks")
	flag.StringVar(&flagTrans, "t", "", "translation name")
	flag.StringVar(&flagBackup, "b", "", "backup files to this path before modifying them")
	flag.StringVar(&flagOutput, "o", "", "alternate output directory")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "no database provided")
		flag.Usage()
		flag.PrintDefaults()
	}

	maxprocs := runtime.GOMAXPROCS(0)
	if maxprocs < flagParallel {
		runtime.GOMAXPROCS(flagParallel)
	}

	dbpath := flag.Arg(0)
	db, err := trans.NewDatabase(dbpath)
	ragequit(dbpath, err)

	if flagTrans != "" {
		if flag.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "no pso2 dir provided")
			return
		}

		if flagBackup != "" {
			err := os.MkdirAll(flagBackup, 0777)
			ragequit(flagBackup, err)
		}

		pso2dir := flag.Arg(1)
		if flagOutput == "" {
			flagOutput = pso2dir
		} else {
			err := os.MkdirAll(flagOutput, 0777)
			ragequit(flagOutput, err)
		}

		cmd.PatchFiles(db, dbpath, pso2dir, flagTrans, flagBackup, flagOutput, flagParallel)
	} else {
		fmt.Fprintln(os.Stderr, "no translation name provided")
	}

	db.Close()
}
