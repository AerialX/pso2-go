package download

import (
	"io"
	"fmt"
	"errors"
	"strings"
	"net/url"
	"net/http"
)

const (
	ProductionRoot = "http://download.pso2.jp/patch_prod/"
	ProductionPatchlist = ProductionRoot + "patches/patchlist.txt"
	ProductionPatchlistOld = ProductionRoot + "patches_old/patchlist.txt"
	ProductionLauncherlist = ProductionRoot + "patches/launcherlist.txt"
	ProductionVersion = ProductionRoot + "patches/version.ver"
)

var httpClient http.Client

func Request(urlStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "AQUA_HTTP")

	resp, err := httpClient.Do(req)
	if err == nil && resp.StatusCode != 200 {
		return resp, errors.New(urlStr + " " + resp.Status)
	}

	return resp, err
}

func DownloadList(urlStr string) (p *PatchList, err error) {
	resp, err := Request(urlStr)

	if err != nil {
		return
	}

	capHint := 20
	if strings.Contains(urlStr, "patchlist") {
		capHint = 20000
	}

	p, err = ParseListCap(resp.Body, urlStr, capHint)

	resp.Body.Close()

	return
}

func ParseList(r io.Reader, urlStr string) (p *PatchList, err error) {
	return ParseListCap(r, urlStr, 20)
}

func ParseListCap(r io.Reader, urlStr string, capHint int) (p *PatchList, err error) {
	url, err := url.Parse(urlStr)
	if err != nil {
		return
	}
	p = &PatchList{URL: url, Entries: make([]PatchEntry, 0, capHint)}

	for err != io.EOF {
		var filename, hash string
		var filesize int64
		var n int
		n, err = fmt.Fscanln(r, &filename, &filesize, &hash)

		if err != nil || n != 3 {
			continue
		}

		var filehash []uint8
		n, err := fmt.Sscanf(hash, "%x", &filehash)
		if err != nil || n != 1 || len(filehash) != 0x10 {
			continue
		}

		e := PatchEntry{PatchList: p, Path: filename, Size: filesize}
		copy(e.MD5[:], filehash)

		p.Entries = append(p.Entries, e)
	}

	err = nil

	p.fillMap()

	return
}
