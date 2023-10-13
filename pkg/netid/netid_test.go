package netid

import (
	"fmt"
	"testing"
)

func TestGenerateNetID(t *testing.T) {
	g := NetID{}
	netID, err := g.GenerateNetID()
	if err != nil {
		t.Errorf("Error generating NetID: %v", err)
	}
	fmt.Printf("NetID: %s\n", netID)
}
