package trans

import (
	"fmt"
	"errors"
)

type ArchiveName [0x10]uint8

type Archive struct {
	id int64
	DB *Database
	Name ArchiveName
}

type File struct {
	id int64
	archiveid int64
	DB *Database
	Name string
}

type String struct {
	id int64
	fileid int64
	DB *Database
	Version int
	Collision int
	Identifier, Value string
}

type Translation struct {
	id int64
	DB *Database
	Name string
}

type TranslationString struct {
	stringid int64
	translationid int64
	DB *Database
	Translation string
}

func (a *ArchiveName) String() string {
	return fmt.Sprintf("%x", a[:])
}

func ArchiveNameFromString(value string) (*ArchiveName, error) {
	var name []uint8
	n, err := fmt.Sscanf(value, "%x", &name)
	if err != nil {
		return nil, err
	} else if n != 1 {
		return nil, errors.New("invalid archive name format")
	}

	var aname ArchiveName
	if len(name) != len(aname) {
		return nil, errors.New("invalid archive name length")
	}

	copy(aname[:], name)
	return &aname, nil
}
