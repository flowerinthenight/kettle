package share

import (
	"log"
	"testing"
)

func TestGen(t *testing.T) {
	s, err := New(nil, withVerbose(true))
	if err != nil {
		t.Fatal(err)
	}

	log.Println(s.IsVerbose())
}
