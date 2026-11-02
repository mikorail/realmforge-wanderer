package engine

import "testing"

func TestBestiaryCatalogue(t *testing.T) {
	gr, er := bsCounts()
	t.Logf("species %d groups %v eras %v", len(bsAll), gr, er)
}
