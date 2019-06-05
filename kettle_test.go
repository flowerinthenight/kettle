package kettle

import (
	"log"
	"testing"
)

func TestGen(t *testing.T) {
	s, err := New(withVerbose(true))
	if err != nil {
		t.Fatal(err)
	}

	log.Println(s.IsVerbose())
}
