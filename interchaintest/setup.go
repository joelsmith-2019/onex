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
	ProviderName    = "onomy"
	ProviderVersion = "v1.1.4"
	ConsumerName    = "onex"
	ConsumerVersion = "local"

	Denom        = "unom"
	Bech32Prefix = "onomy"

	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"

	OnomyImage = ibc.DockerImage{
		Repository: ProviderName,
		Version:    ProviderVersion,
		UidGid:     "1025:1025",
	}

	OnomyConfig = ibc.ChainConfig{
		Type:           "cosmos",
		Name:           ProviderName,
		ChainID:        "onomy-1",
		Images:         []ibc.DockerImage{OnomyImage},
		Bin:            "onomyd",
		Bech32Prefix:   Bech32Prefix,
		Denom:          Denom,
		CoinType:       "118",
		GasPrices:      fmt.Sprintf("0%s", Denom),
		GasAdjustment:  1.0,
		TrustingPeriod: "168h",
		NoHostMount:    false,
		EncodingConfig: OnomyEncoding(),
	}

	OnexImage = ibc.DockerImage{
		Repository: ConsumerName,
		Version:    ConsumerVersion,
		UidGid:     "1025:1025",
	}

	// defaultGenesisKV = []cosmos.GenesisKV{
	// {
	// 	Key:   "app_state.gov.params.voting_period",
	// 	Value: VotingPeriod,
	// },
	// {
	// 	Key:   "app_state.gov.params.max_deposit_period",
	// 	Value: MaxDepositPeriod,
	// },
	// {
	// 	Key:   "app_state.gov.params.min_deposit.0.denom",
	// 	Value: Denom,
	// },
	// }

	OnexConfig = ibc.ChainConfig{
		Type:                "cosmos",
		Name:                ConsumerName,
		ChainID:             "local-1",
		Images:              []ibc.DockerImage{OnexImage},
		Bin:                 "onexd",
		Bech32Prefix:        Bech32Prefix,
		Denom:               Denom,
		CoinType:            "118",
		GasPrices:           fmt.Sprintf("0%s", Denom),
		GasAdjustment:       1.0,
		TrustingPeriod:      "168h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      OnexEncoding(),
		ModifyGenesis:       nil, //cosmos.ModifyGenesis(defaultGenesisKV),
	}
)

// OnomyEncoding returns the encoding config for the onomy chain
func OnomyEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()
	return &cfg
}

// OnexEncoding returns the encoding config for the onex chain
func OnexEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// TODO: ADD CHAIN-SPECIFIC ENCODING HERE

	return &cfg
}

// CreateConsumerChainsWithCustomConfig creates chain(s) with custom configuration. It will always
// set the first chain to the onomy provider chain and all other chains to the onex consumer chain.
func CreateConsumerChainsWithCustomConfig(t *testing.T, numVals, numFull int, config ibc.ChainConfig) []ibc.Chain {

	providerVals, providerFull := 1, 0

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          ProviderName,
			ChainName:     ProviderName,
			Version:       ProviderVersion,
			ChainConfig:   OnomyConfig,
			NumValidators: &providerVals,
			NumFullNodes:  &providerFull,
		},
		{
			Name:          ConsumerName,
			ChainName:     ConsumerName,
			Version:       ConsumerVersion,
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

// BuildInitialChain creates a new interchain object and builds the chains. chains[0] will always be the provider chain.
func BuildInitialChain(t *testing.T, chains []ibc.Chain) (*interchaintest.Interchain, context.Context, *client.Client, string) {
	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain()

	// Relayer Factory
	client, network := interchaintest.DockerSetup(t)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
	).Build(t, client, network)

	ic.AddRelayer(r, "relayer")

	const ibcPath = "ics-path"

	// Provider is always the first chain
	provider := chains[0]
	ic.AddChain(provider)

	for i := 1; i < len(chains); i++ {
		consumer := chains[i]
		ic.AddChain(consumer).
			AddProviderConsumerLink(interchaintest.ProviderConsumerLink{
				Provider: provider,
				Consumer: consumer,
				Relayer:  r,
				Path:     ibcPath,
			})
	}

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	ctx := context.Background()

	err := ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	})
	require.NoError(t, err)

	return ic, ctx, client, network
}
