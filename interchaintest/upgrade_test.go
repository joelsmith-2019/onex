package interchaintest

import (
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
)

// TestUpgrade tests the upgrade of the chain
func TestUpgrade(t *testing.T) {
	t.Parallel()

	cfg := OnomyConfig

	chains := CreateChainsWithCustomConfig(t, 1, 0, cfg)

	fmt.Println("We made it here!")

	ic, ctx, _, _ := BuildInitialChain(t, chains)

	onomy := chains[0].(*cosmos.CosmosChain)

	testutil.WaitForBlocks(ctx, 15, onomy)

	// Cleanup test
	t.Cleanup(func() {
		_ = ic.Close()
	})
}
