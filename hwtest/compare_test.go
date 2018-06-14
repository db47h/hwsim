package hwtest_test

import (
	"testing"

	hw "github.com/db47h/hwsim"
	hl "github.com/db47h/hwsim/hwlib"
	"github.com/db47h/hwsim/hwtest"
)

func TestComparePart(t *testing.T) {
	or, err := hw.Chip("custom_or", "a,b", "out",
		hl.Nand("a=a, b=a, out=notA"),
		hl.Nand("a=b, b=b, out=notB"),
		hl.Nand("a=notA, b=notB, out=out"),
	)
	if err != nil {
		t.Fatal(err)
	}
	hwtest.ComparePart(t, 4, hl.Or, or)
}
