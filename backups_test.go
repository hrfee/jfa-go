package main

import (
	"testing"
	"time"
)

func testBackupParse(f string, a Backup, t *testing.T) {
	b := Backup{}
	err := b.FromString(f)
	if err != nil {
		t.Fatalf("error: %+v", err)
	}
	if !b.Equals(a) {
		t.Fatalf("not equal: %+v != %+v", b, a)
	}
}

func TestBackupParserOld(t *testing.T) {
	Q1 := BACKUP_PREFIX_OLD + "2023-12-21T21-08-00" + BACKUP_SUFFIX
	A1 := Backup{}
	A1.Date, _ = time.Parse(BACKUP_DATEFMT, "2023-12-21T21-08-00")
	testBackupParse(Q1, A1, t)
}
func TestBackupParserOldUpload(t *testing.T) {
	Q2 := BACKUP_UPLOAD_PREFIX + BACKUP_PREFIX_OLD + "2023-12-21T21-08-00" + BACKUP_SUFFIX
	A2 := Backup{
		Upload: true,
	}
	A2.Date, _ = time.Parse(BACKUP_DATEFMT, "2023-12-21T21-08-00")
	testBackupParse(Q2, A2, t)
}
func TestBackupParserUploadDate(t *testing.T) {
	Q3 := BACKUP_UPLOAD_PREFIX + BACKUP_PREFIX + BACKUP_DATE_PREFIX + "2023-12-21T21-08-00" + BACKUP_SUFFIX
	A3 := Backup{
		Upload: true,
	}
	A3.Date, _ = time.Parse(BACKUP_DATEFMT, "2023-12-21T21-08-00")
	testBackupParse(Q3, A3, t)
}
func TestBackupParserUploadCommitDate(t *testing.T) {
	Q4 := BACKUP_UPLOAD_PREFIX + BACKUP_PREFIX + BACKUP_COMMIT_PREFIX + "testcommit" + BACKUP_DATE_PREFIX + "2023-12-21T21-08-00" + BACKUP_SUFFIX
	A4 := Backup{
		Commit: "testcommit",
		Upload: true,
	}
	A4.Date, _ = time.Parse(BACKUP_DATEFMT, "2023-12-21T21-08-00")
	testBackupParse(Q4, A4, t)
}
func TestBackupParserDateCommit(t *testing.T) {
	Q5 := BACKUP_PREFIX + BACKUP_DATE_PREFIX + "2023-12-21T21-08-00" + BACKUP_COMMIT_PREFIX + "testcommit" + BACKUP_SUFFIX
	A5 := Backup{
		Commit: "testcommit",
	}
	A5.Date, _ = time.Parse(BACKUP_DATEFMT, "2023-12-21T21-08-00")
	testBackupParse(Q5, A5, t)
}
