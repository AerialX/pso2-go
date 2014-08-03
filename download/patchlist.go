package download

import (
	"io"
	"fmt"
	"path"
	"bytes"
	"strings"
	"net/url"
)

type PatchList struct {
	URL *url.URL

	Entries []PatchEntry

	EntryMap map[string]*PatchEntry
}

type PatchEntry struct {
	PatchList *PatchList
	Path string
	Size int64
	MD5 [0x10]uint8
}

func (p *PatchList) fillMap() {
	p.EntryMap = make(map[string]*PatchEntry)

	for i, _ := range p.Entries {
		e := &p.Entries[i]

		p.EntryMap[e.BaseName()] = e
	}
}

func (p *PatchEntry) URL() (ourl *url.URL, err error) {
	url, err := url.Parse(p.Path)
	if err != nil {
		return
	}

	return p.PatchList.URL.ResolveReference(url), nil
}

func (p *PatchEntry) BaseName() string {
	return path.Base(p.Path)
}

func RemoveExtension(s string) string {
	return strings.TrimSuffix(s, ".pat")
}

func (p *PatchList) Write(w io.Writer) error {
	for _, e := range p.Entries {
		_, err := fmt.Fprintf(w, "%s\t%d\t%X\n", e.Path, e.Size, e.MD5)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PatchList) Diff(po *PatchList) (pn *PatchList) {
	if po == nil {
		return p
	}

	pn = &PatchList{EntryMap: make(map[string]*PatchEntry)}

	for _, e := range p.Entries {
		eo := po.EntryMap[e.BaseName()]

		if eo == nil || e.Size != eo.Size || bytes.Compare(e.MD5[:], eo.MD5[:]) != 0 {
			pn.Entries = append(pn.Entries, e)
		}
	}

	pn.fillMap()

	return
}

func (p *PatchList) MergeOld(po *PatchList) (pn *PatchList) {
	if po == nil {
		return p
	}

	pn = &PatchList{Entries: p.Entries, EntryMap: make(map[string]*PatchEntry)}

	for _, e := range po.Entries {
		eo := p.EntryMap[e.BaseName()]

		if eo == nil {
			pn.Entries = append(pn.Entries, e)
		}
	}

	pn.fillMap()

	return
}

func (p *PatchList) Append(po *PatchList) {
	p.Entries = append(p.Entries, po.Entries...)
	p.fillMap()
}
