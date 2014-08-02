package trans

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

const sqlCreateTables = `
	CREATE TABLE IF NOT EXISTS archives (
		archiveid INTEGER PRIMARY KEY NOT NULL,
		name BLOB UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS files (
		fileid INTEGER PRIMARY KEY NOT NULL,
		archiveid INTEGER REFERENCES archives(archiveid) NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS strings (
		stringid INTEGER PRIMARY KEY NOT NULL,
		fileid INTEGER REFERENCES files(fileid) NOT NULL,
		version INTEGER NOT NULL,
		collision INTEGER NOT NULL,
		identifier TEXT NOT NULL,
		value TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS strings_index ON strings(fileid, identifier, collision);

	CREATE TABLE IF NOT EXISTS translations (
		translationid INTEGER PRIMARY KEY NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS translationstrings (
		translationid INTEGER REFERENCES translations(translationid) NOT NULL,
		stringid INTEGER REFERENCES strings(stringid) NOT NULL,
		translation TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS translationstrings_index ON translationstrings(translationid, stringid);
`


func NewDatabase(path string) (*Database, error){
	db, err := sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(sqlCreateTables)

	return &Database{db}, err
}

func (d *Database) Close() {
	d.db.Close()
}

func (d *Database) Begin() {
	d.db.Exec("BEGIN")
}

func (d *Database) End() {
	d.db.Exec("END")
}

func (d *Database) query(query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return
	}

	rows, err = stmt.Query(args...)

	stmt.Close()

	return
}

func (d *Database) exec(query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return
	}

	result, err = stmt.Exec(args...)

	stmt.Close()

	return
}

func (d *Database) InsertArchive(name *ArchiveName) (a *Archive, err error) {
	result, err := d.exec("INSERT INTO archives (name) VALUES (?)", name[:])
	if err != nil {
		return
	}

	id, err := result.LastInsertId()
	a = &Archive{id, d, *name}

	return
}

const sqlQueryArchive = "SELECT archiveid, name FROM archives "
func (d *Database) queryArchive(rows *sql.Rows) (a *Archive, err error) {
	a = &Archive{}
	a.DB = d
	var name []uint8
	err = rows.Scan(&a.id, &name)
	if err != nil {
		a = nil
	} else {
		copy(a.Name[:], name)
	}

	return
}

func (d *Database) queryArchives(query string, args ...interface{}) (archives []Archive, err error) {
	rows, err := d.query(query, args...)
	if err != nil {
		return
	}

	for rows.Next() {
		var a *Archive
		a, err = d.queryArchive(rows)
		if err != nil {
			break
		}

		archives = append(archives, *a)
	}

	rows.Close()
	return
}

func (d *Database) QueryArchive(name *ArchiveName) (a *Archive, err error) {
	rows, err := d.query(sqlQueryArchive + "WHERE name = ?", name[:])
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		a, err = d.queryArchive(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryArchives() (archives []Archive, err error) {
	return d.queryArchives(sqlQueryArchive)
}

func (d *Database) QueryArchivesTranslation(t *Translation) (archives []Archive, err error) {
	return d.queryArchives(`
		SELECT a.archiveid, a.name
		FROM translationstrings AS ts
			JOIN strings AS s ON s.stringid = ts.stringid
			JOIN files AS f ON f.fileid = s.fileid
			JOIN archives AS a ON a.archiveid = f.archiveid
		WHERE ts.translationid = ?
		GROUP BY a.archiveid`, t.id)
}

func (d *Database) InsertFile(a *Archive, name string) (f *File, err error) {
	result, err := d.exec("INSERT INTO files (archiveid, name) VALUES (?, ?)", a.id, name)
	if err != nil {
		return
	}

	id, err := result.LastInsertId()
	f = &File{id, a.id, d, name}

	return
}

const sqlQueryFile = "SELECT fileid, archiveid, name FROM files "
func (d *Database) queryFile(rows *sql.Rows) (f *File, err error) {
	f = &File{}
	f.DB = d
	err = rows.Scan(&f.id, &f.archiveid, &f.Name)
	if err != nil {
		f = nil
	}

	return
}

func (d *Database) QueryFile(a *Archive, name string) (f *File, err error) {
	rows, err := d.query(sqlQueryFile + "WHERE archiveid = ? AND name = ?", a.id, name)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		f, err = d.queryFile(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryFiles(a *Archive) (files []File, err error) {
	rows, err := d.query(sqlQueryFile + "WHERE archiveid = ?", a.id)
	if err != nil {
		return
	}

	for rows.Next() {
		var f *File
		f, err = d.queryFile(rows)
		if err != nil {
			break
		}

		files = append(files, *f)
	}

	rows.Close()
	return
}

func (d *Database) InsertString(f *File, version, collision int, identifier, value string) (s *String, err error) {
	result, err := d.exec("INSERT INTO strings (fileid, version, collision, identifier, value) VALUES (?, ?, ?, ?, ?)", f.id, version, collision, identifier, value)
	if err != nil {
		return
	}

	id, err := result.LastInsertId()
	s = &String{id, f.id, d, version, collision, identifier, value}

	return
}

func (d *Database) UpdateString(f *String, version int, value string) (s *String, err error) {
	_, err = d.exec("UPDATE strings SET version = ?, value = ? WHERE stringid = ?", version, value, f.id)
	if err != nil {
		return
	}

	s = &String{f.id, f.fileid, d, version, f.Collision, f.Identifier, value}

	return
}

const sqlQueryString = "SELECT stringid, fileid, version, collision, identifier, value FROM strings "
func (d *Database) queryString(rows *sql.Rows) (s *String, err error) {
	s = &String{}
	s.DB = d
	err = rows.Scan(&s.id, &s.fileid, &s.Version, &s.Collision, &s.Identifier, &s.Value)
	if err != nil {
		s = nil
	}

	return
}

func (d *Database) QueryString(f *File, collision int, id string) (s *String, err error) {
	rows, err := d.query(sqlQueryString + "WHERE fileid = ? AND collision = ? AND identifier = ?", f.id, collision, id)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		s, err = d.queryString(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryStringTranslation(t *TranslationString) (s *String, err error) {
	rows, err := d.query(sqlQueryString + "WHERE stringid = ?", t.stringid)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		s, err = d.queryString(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryStrings() (strings []String, err error) {
	rows, err := d.query(sqlQueryString)
	if err != nil {
		return
	}

	for rows.Next() {
		var s *String
		s, err = d.queryString(rows)
		if err != nil {
			break
		}

		strings = append(strings, *s)
	}

	rows.Close()
	return
}

func (d *Database) InsertTranslation(name string) (t *Translation, err error) {
	result, err := d.exec("INSERT INTO translations (name) VALUES (?)", name)
	if err != nil {
		return
	}

	id, err := result.LastInsertId()
	t = &Translation{id, d, name}

	return
}

const sqlQueryTranslation = "SELECT translationid, name FROM translations "
func (d *Database) queryTranslation(rows *sql.Rows) (t *Translation, err error) {
	t = &Translation{}
	t.DB = d
	err = rows.Scan(&t.id, &t.Name)
	if err != nil {
		t = nil
	}

	return
}

func (d *Database) QueryTranslation(name string) (t *Translation, err error) {
	rows, err := d.query(sqlQueryTranslation + "WHERE name = ?", name)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		t, err = d.queryTranslation(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryTranslations() (translations []Translation, err error) {
	rows, err := d.query(sqlQueryTranslation)
	if err != nil {
		return
	}

	for rows.Next() {
		var t *Translation
		t, err = d.queryTranslation(rows)
		if err != nil {
			break
		}

		translations = append(translations, *t)
	}

	rows.Close()
	return
}

func (d *Database) InsertTranslationString(t *Translation, f *String, translation string) (s *TranslationString, err error) {
	_, err = d.exec("INSERT INTO translationstrings (translationid, stringid, translation) VALUES (?, ?, ?)", t.id, f.id, translation)
	if err != nil {
		return
	}

	s = &TranslationString{f.id, t.id, d, translation}

	return
}

func (d *Database) UpdateTranslationString(f *TranslationString, value string) (s *TranslationString, err error) {
	_, err = d.exec("UPDATE translationstrings SET translation = ? WHERE translationid = ? AND stringid = ?", value, f.translationid, f.stringid)
	if err != nil {
		return
	}

	s = &TranslationString{f.translationid, f.stringid, d, value}

	return
}

const sqlQueryTranslationString = "SELECT translationid, stringid, translation FROM translationstrings "
func (d *Database) queryTranslationString(rows *sql.Rows) (s *TranslationString, err error) {
	s = &TranslationString{}
	s.DB = d
	err = rows.Scan(&s.translationid, &s.stringid, &s.Translation)
	if err != nil {
		s = nil
	}

	return
}

func (d *Database) queryTranslationStrings(query string, args ...interface{}) (strings []TranslationString, err error) {
	rows, err := d.query(query, args...)
	if err != nil {
		return
	}

	for rows.Next() {
		var s *TranslationString
		s, err = d.queryTranslationString(rows)
		if err != nil {
			break
		}

		strings = append(strings, *s)
	}

	rows.Close()
	return
}

func (d *Database) QueryTranslationString(t *Translation, f *String) (s *TranslationString, err error) {
	rows, err := d.query(sqlQueryTranslationString + "WHERE translationid = ? AND stringid = ?", t.id, f.id)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		s, err = d.queryTranslationString(rows)
	}

	rows.Close()
	return
}

func (d *Database) QueryTranslationStrings(t *Translation) (strings []TranslationString, err error) {
	return d.queryTranslationStrings(sqlQueryTranslationString + "WHERE translationid = ?", t)
}

func (d *Database) QueryTranslationStringsFile(t *Translation, f *File) (strings []TranslationString, err error) {
	return d.queryTranslationStrings(`
		SELECT ts.translationid, ts.stringid, ts.translation
		FROM translationstrings AS ts
			JOIN strings AS s ON s.stringid = ts.stringid
		WHERE ts.translationid = ? AND s.fileid = ?`, t.id, f.id)
}

func (d *Database) Strip() (err error) {
	_, err = d.db.Exec(`
		UPDATE strings SET value = '', version = 0;
		DELETE FROM strings WHERE NOT stringid IN (SELECT stringid FROM translationstrings);
		DELETE FROM files WHERE NOT fileid IN (SELECT fileid FROM strings);
		DELETE FROM archives WHERE NOT archiveid IN (SELECT archiveid FROM files);
		VACUUM;
		ANALYZE;
	`)
	return
}
