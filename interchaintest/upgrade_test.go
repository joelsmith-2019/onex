package interchaintest

import (
	"context"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v4"
	cosmos "github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"github.com/stretchr/testify/require"
)

func TestRandom(t *testing.T) {
	t.Parallel()

	// Setup chains, build interchain
	chains := CreateChainsWithCustomConsumerConfig(t, 1, 0, OnexConfig)
	onomy, onex := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
	ic, ctx, relayer, eRep, _, _ := BuildInitialChain(t, onomy, onex)

	testutil.WaitForBlocks(ctx, 5, onomy, onex)

	// Start relayer
	t.Log("Starting relayer............................................")
	require.NoError(t, relayer.StartRelayer(ctx, eRep, "ics-path"))
	testutil.WaitForBlocks(ctx, 5, onomy, onex)

	funds := int64(10_000_000_000)
	onexUser := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), funds, onex)[0]
	coins, err := onex.AllBalances(ctx, onexUser.FormattedAddress())
	require.NoError(t, err)
	t.Log("Onex Coins", coins)

	// Ensure chains are properly producing blocks
	testutil.WaitForBlocks(ctx, 500, onex)

	t.Log("Length of Chains", len(chains))

	// Cleanup test
	t.Cleanup(func() {
		_ = ic.Close()
	})
}

// TestUpgrade tests the upgrade of the chain
func TestUpgrade(t *testing.T) {
	t.Parallel()

	// Setup chains, build interchain
	chains := CreateChainsWithCustomConsumerConfig(t, 1, 0, OnexConfig)
	onomy, onex := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
	ic, ctx, _, _, _, _ := BuildInitialChain(t, onomy, onex)

	// Ensure chains are properly producing blocks
	testutil.WaitForBlocks(ctx, 5, onex)

	// Fund test user on onex
	funds := int64(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), funds, onex)
	onexUser := users[0]

	// Calculate onex height, halt, height & submit proposal
	height, err := onex.Height(ctx)
	require.NoError(t, err)
	haltHeight := height + 10
	propId := submitUpgradeProposal(t, ctx, onex, onexUser, "upgrade-1", haltHeight)
	t.Log("Waiting for proposal to pass", propId)

	// Notify onex validators to vote on proposal
	err = onex.VoteOnProposalAllValidators(ctx, propId, cosmos.ProposalVoteYes)
	require.NoError(t, err)

	// Wait for proposal to pass
	res, err := cosmos.PollForProposalStatus(ctx, onex, height, haltHeight, propId, cosmos.ProposalStatusPassed)
	require.NoError(t, err, "error polling for proposal status")

	t.Log("Proposal passed:", res)

	// TODO: UPGRADE NODES

	// Cleanup test
	t.Cleanup(func() {
		_ = ic.Close()
	})
}

// submitUpgradeProposal submits a software upgrade proposal on chain
func submitUpgradeProposal(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, upgradeName string, haltHeight uint64) string {
	upgradeMsg := cosmos.SoftwareUpgradeProposal{
		Deposit:     fmt.Sprintf(`500000000%s`, chain.Config().Denom),
		Title:       "Software Upgrade",
		Name:        upgradeName,
		Description: "Software Upgrade",
		Height:      haltHeight,
		Info:        "ipfs://CID",
	}

	tx, err := chain.UpgradeProposal(ctx, user.KeyName(), upgradeMsg)
	require.NoError(t, err, "error submitting proposal")
	t.Log("Proposal ID", tx.ProposalID)

	return tx.ProposalID
}

func upgradeNodes(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, upgradeName string, haltHeight uint64) {
	t.Log("Upgrading chain to version: ", upgradeName)
}
