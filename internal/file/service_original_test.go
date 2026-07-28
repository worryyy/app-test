package file

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestValidOOXMLRejectsGenericZip(t *testing.T) {
	docx := zipBytes(t, []string{"[Content_Types].xml", "word/document.xml"})
	if !validAcademicDocument(docx, ".docx", "application/zip") {
		t.Fatal("valid docx was rejected")
	}
	generic := zipBytes(t, []string{"readme.txt"})
	if validAcademicDocument(generic, ".docx", "application/zip") {
		t.Fatal("generic zip was accepted as docx")
	}
	if validAcademicDocument(generic, ".zip", "application/zip") {
		t.Fatal("zip extension was accepted")
	}
}

func zipBytes(t *testing.T, names []string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
