package interchaintest

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"
	Denom            = "unom"

	OnomyImage = ibc.DockerImage{
		Repository: "onomy",
		Version:    "local",
		UidGid:     "1025:1025",
	}

	defaultGenesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.params.voting_period",
			Value: VotingPeriod,
		},
		{
			Key:   "app_state.gov.params.max_deposit_period",
			Value: MaxDepositPeriod,
		},
		{
			Key:   "app_state.gov.params.min_deposit.0.denom",
			Value: Denom,
		},
	}

	OnomyConfig = ibc.ChainConfig{
		Type:                "cosmos",
		Name:                "onomy",
		ChainID:             "onomy-ic-1",
		Images:              []ibc.DockerImage{OnomyImage},
		Bin:                 "onexd",
		Bech32Prefix:        "onomy",
		Denom:               Denom,
		CoinType:            "118",
		GasPrices:           fmt.Sprintf("0%s", Denom),
		GasAdjustment:       2.0,
		TrustingPeriod:      "112h", //TODO: DETERMINE THE TRUSTING PERIOD
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      OnomyEncoding(),
		ModifyGenesis:       cosmos.ModifyGenesis(defaultGenesisKV),
	}
)

// onomyEncoding returns the encoding config for the onomy chain
func OnomyEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// TODO: ADD CHAIN-SPECIFIC ENCODING HERE

	return &cfg
}

// CreateChainsWithCustomConfig creates chain(s) with custom configuration
func CreateChainsWithCustomConfig(t *testing.T, numVals, numFull int, config ibc.ChainConfig) []ibc.Chain {
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "onomy",
			ChainName:     "onomy",
			Version:       config.Images[0].Version,
			ChainConfig:   config,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

// BuildInitialChain creates a new interchain object and builds the chains
func BuildInitialChain(t *testing.T, chains []ibc.Chain) (*interchaintest.Interchain, context.Context, *client.Client, string) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic = ic.AddChain(chain)
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	return ic, ctx, client, network
}
