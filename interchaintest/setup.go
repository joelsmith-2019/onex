package interchaintest

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	daotypes "github.com/onomyprotocol/onomy/x/dao/types"
)

var (
	ProviderName    = "onomy"
	ProviderVersion = "v1.1.4"
	ProviderDenom   = "anom"

	ConsumerName    = "onex"
	ConsumerVersion = "local"
	ConsumerDenom   = ProviderDenom

	Bech32Prefix = "onomy"

	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"
	MinDepositAount  = "1000000"

	// Returns genesisAmount & genesisSelfDelegationAmount
	// Onomy/Onex requires significantly more than the default amount
	ModifyGenesisAmounts = func() (types.Coin, types.Coin) {
		return types.Coin{
				Amount: types.NewInt(9_000_000_000_000_000_000),
				Denom:  ProviderDenom,
			},
			types.Coin{
				Amount: types.NewInt(5_000_000_000_000_000_000),
				Denom:  ProviderDenom,
			}
	}

	OnomyGenesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.voting_params.voting_period",
			Value: VotingPeriod,
		},
		{
			Key:   "app_state.gov.deposit_params.max_deposit_period",
			Value: MaxDepositPeriod,
		},
		{
			Key:   "app_state.gov.deposit_params.min_deposit.0.denom",
			Value: ProviderDenom,
		},
		{
			Key:   "app_state.gov.deposit_params.min_deposit.0.amount",
			Value: MinDepositAount,
		},
	}

	OnomyImage = ibc.DockerImage{
		Repository: ProviderName,
		Version:    ProviderVersion,
		UidGid:     "1025:1025",
	}

	OnomyConfig = ibc.ChainConfig{
		Type:                 "cosmos",
		Name:                 ProviderName,
		ChainID:              "onomy-1",
		Images:               []ibc.DockerImage{OnomyImage},
		Bin:                  ProviderName + "d",
		Bech32Prefix:         Bech32Prefix,
		Denom:                ProviderDenom,
		CoinType:             "118",
		GasPrices:            fmt.Sprintf("0%s", ProviderDenom),
		GasAdjustment:        1.0,
		TrustingPeriod:       "504h",
		NoHostMount:          false,
		ConfigFileOverrides:  nil,
		EncodingConfig:       OnomyEncoding(),
		ModifyGenesis:        cosmos.ModifyGenesis(OnomyGenesisKV),
		ModifyGenesisAmounts: ModifyGenesisAmounts,
	}

	OnexGenesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.voting_params.voting_period",
			Value: VotingPeriod,
		},
		{
			Key:   "app_state.gov.deposit_params.max_deposit_period",
			Value: MaxDepositPeriod,
		},
		{
			Key:   "app_state.gov.deposit_params.min_deposit.0.denom",
			Value: ConsumerDenom,
		},
		{
			Key:   "app_state.gov.deposit_params.min_deposit.0.amount",
			Value: MinDepositAount,
		},
	}

	OnexImage = ibc.DockerImage{
		Repository: ConsumerName,
		Version:    ConsumerVersion,
		UidGid:     "1025:1025",
	}

	OnexConfig = ibc.ChainConfig{
		Type:                 "cosmos",
		Name:                 ConsumerName,
		ChainID:              "onex-1",
		Images:               []ibc.DockerImage{OnexImage},
		Bin:                  ConsumerName + "d",
		Bech32Prefix:         Bech32Prefix,
		Denom:                ConsumerDenom,
		CoinType:             "118",
		GasPrices:            fmt.Sprintf("0%s", ConsumerDenom),
		GasAdjustment:        1.0,
		TrustingPeriod:       "48h",
		NoHostMount:          false,
		ConfigFileOverrides:  nil,
		EncodingConfig:       OnexEncoding(),
		ModifyGenesis:        cosmos.ModifyGenesis(OnexGenesisKV),
		ModifyGenesisAmounts: ModifyGenesisAmounts,
	}
)

// OnomyEncoding returns the encoding config for the onomy chain
func OnomyEncoding() *params.EncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// Add custom encoding overrides here
	daotypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

// OnexEncoding returns the encoding config for the onex chain
func OnexEncoding() *params.EncodingConfig {
	cfg := cosmos.DefaultEncoding()
	return &cfg
}

// CreateChainsWithCustomConsumerConfig creates chain(s) with custom configuration. It will always
// set the first chain to the onomy provider chain and the second chain to the onex consumer chain.
func CreateChainsWithCustomConsumerConfig(t *testing.T, numVals, numFull int, consumerConfig ibc.ChainConfig) []ibc.Chain {

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          ProviderName,
			ChainName:     ProviderName,
			Version:       ProviderVersion,
			ChainConfig:   OnomyConfig,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
		{
			Name:          ConsumerName,
			ChainName:     ConsumerName,
			Version:       ConsumerVersion,
			ChainConfig:   consumerConfig,
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	// Assert length is 2 (provider & consumer)
	require.Len(t, chains, 2)

	// Assert position 0 is provider and position 1 is consumer
	require.Equal(t, chains[0].Config().Name, ProviderName)
	require.Equal(t, chains[1].Config().Name, ConsumerName)

	return chains
}

// BuildInitialChain creates a new interchain object and builds the chains.
func BuildInitialChain(t *testing.T, providerChain ibc.Chain, consumerChain ibc.Chain) (*interchaintest.Interchain, context.Context, ibc.Relayer, *testreporter.RelayerExecReporter, *client.Client, string) {
	// Relayer Factory
	client, network := interchaintest.DockerSetup(t)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
	).Build(t, client, network)

	const ibcPath = "ics-path"

	ic := interchaintest.NewInterchain().
		AddChain(providerChain).
		AddChain(consumerChain).
		AddRelayer(r, "relayer").
		AddProviderConsumerLink(interchaintest.ProviderConsumerLink{
			Provider: providerChain,
			Consumer: consumerChain,
			Relayer:  r,
			Path:     ibcPath,
		})

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

	return ic, ctx, r, eRep, client, network
}
