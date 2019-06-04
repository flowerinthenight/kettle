package share

import (
	"log"
	"testing"
)

func TestGen(t *testing.T) {
	s, err := New(WithName("hello"))
	if err != nil {
		t.Fatal(err)
	}

	log.Println(s.Name())
}
