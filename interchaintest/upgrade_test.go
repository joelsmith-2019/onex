package interchaintest

import (
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
)

// TestUpgrade tests the upgrade of the chain
func TestUpgrade(t *testing.T) {
	t.Parallel()

	cfg := OnexConfig

	chains := CreateConsumerChainsWithCustomConfig(t, 1, 0, cfg)
	ic, ctx, _, _ := BuildInitialChain(t, chains)
	onex := chains[1].(*cosmos.CosmosChain)

	testutil.WaitForBlocks(ctx, 15, onex)

	// Cleanup test
	t.Cleanup(func() {
		_ = ic.Close()
	})
}
