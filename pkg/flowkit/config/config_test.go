/*
 * Flow CLI
 *
 * Copyright 2019 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package config_test

import (
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/onflow/flow-cli/pkg/flowkit/config"
)

func generateComplexConfig() config.Config {
	var keys []crypto.PrivateKey
	key, _ := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, "dd72967fd2bd75234ae9037dd4694c1f00baad63a10c35172bf65fbb8ad74b47")
	keys = append(keys, key)
	key, _ = crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, "388e3fbdc654b765942610679bb3a66b74212149ab9482187067ee116d9a8118")
	keys = append(keys, key)
	key, _ = crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, "27bbeba308e501f8485ddaab77e285c0bc0d611096a79b4f0b4ccc927c6dbf04")
	keys = append(keys, key)

	return config.Config{
		Emulators: config.Emulators{{
			Name:           "default",
			Port:           9000,
			ServiceAccount: "emulator-account",
		}},
		Contracts: config.Contracts{{
			Name:     "NonFungibleToken",
			Location: "../hungry-kitties/cadence/contracts/NonFungibleToken.cdc",
			Network:  "emulator",
		}, {
			Name:     "FungibleToken",
			Location: "../hungry-kitties/cadence/contracts/FungibleToken.cdc",
			Network:  "emulator",
		}, {
			Name:     "Kibble",
			Location: "./cadence/kibble/contracts/Kibble.cdc",
			Network:  "emulator",
		}, {
			Name:     "KittyItems",
			Location: "./cadence/kittyItems/contracts/KittyItems.cdc",
			Network:  "emulator",
		}, {
			Name:     "KittyItemsMarket",
			Location: "./cadence/kittyItemsMarket/contracts/KittyItemsMarket.cdc",
			Network:  "emulator",
		}, {
			Name:     "KittyItemsMarket",
			Location: "0x123123123",
			Network:  "testnet",
		}},
		Deployments: config.Deployments{{
			Network: "emulator",
			Account: "emulator-account",
			Contracts: []config.ContractDeployment{
				{Name: "KittyItems", Args: []cadence.Value{}},
				{Name: "KittyItemsMarket", Args: []cadence.Value{}},
			},
		}, {
			Network: "emulator",
			Account: "account-4",
			Contracts: []config.ContractDeployment{
				{Name: "FungibleToken", Args: []cadence.Value{}},
				{Name: "NonFungibleToken", Args: []cadence.Value{}},
				{Name: "Kibble", Args: []cadence.Value{}},
				{Name: "KittyItems", Args: []cadence.Value{}},
				{Name: "KittyItemsMarket", Args: []cadence.Value{}},
			},
		}, {
			Network: "testnet",
			Account: "account-2",
			Contracts: []config.ContractDeployment{
				{Name: "FungibleToken", Args: []cadence.Value{}},
				{Name: "NonFungibleToken", Args: []cadence.Value{}},
				{Name: "Kibble", Args: []cadence.Value{}},
				{Name: "KittyItems", Args: []cadence.Value{}},
			},
		}},
		Accounts: config.Accounts{{
			Name:    "emulator-account",
			Address: flow.ServiceAddress(flow.Emulator),
			Key: config.AccountKey{
				Type:       config.KeyTypeHex,
				Index:      0,
				SigAlgo:    crypto.ECDSA_P256,
				HashAlgo:   crypto.SHA3_256,
				PrivateKey: keys[0],
			},
		}, {
			Name:    "account-2",
			Address: flow.HexToAddress("2c1162386b0a245f"),
			Key: config.AccountKey{
				Type:       config.KeyTypeHex,
				Index:      0,
				SigAlgo:    crypto.ECDSA_P256,
				HashAlgo:   crypto.SHA3_256,
				PrivateKey: keys[1],
			},
		}, {
			Name:    "account-4",
			Address: flow.HexToAddress("f8d6e0586b0a20c1"),
			Key: config.AccountKey{
				Type:       config.KeyTypeHex,
				Index:      0,
				SigAlgo:    crypto.ECDSA_P256,
				HashAlgo:   crypto.SHA3_256,
				PrivateKey: keys[2],
			},
		}},
		Networks: config.Networks{{
			Name: "emulator",
			Host: "127.0.0.1.3569",
		}, {
			Name: "testnet",
			Host: "access.devnet.nodes.onflow.org:9000",
			Key:  "5000676131ad3e22d853a3f75a5b5d0db4236d08dd6612e2baad771014b5266a242bccecc3522ff7207ac357dbe4f225c709d9b273ac484fed5d13976a39bdcd",
		}},
	}
}

func Test_GetContractsForNetworkComplex(t *testing.T) {
	conf := generateComplexConfig()
	kitty, err := conf.Contracts.ByName("KittyItems")
	assert.NoError(t, err)
	market, err := conf.Contracts.ByName("KittyItemsMarket")
	assert.NoError(t, err)

	assert.Equal(t, "KittyItems", kitty.Name)
	assert.Equal(t, "./cadence/kittyItemsMarket/contracts/KittyItemsMarket.cdc", market.Location)
}

func Test_GetContractsByNameAndNetworkComplex(t *testing.T) {
	conf := generateComplexConfig()
	market, err := conf.Contracts.ByNameAndNetwork("KittyItemsMarket", "testnet")
	assert.NoError(t, err)

	assert.Equal(t, "0x123123123", market.Location)
}

func Test_GetContractsByNetworkComplex(t *testing.T) {
	conf := generateComplexConfig()
	contracts := conf.Contracts.ByNetwork("emulator")

	assert.Len(t, contracts, 5)
	assert.Equal(t, "NonFungibleToken", contracts[0].Name)
	assert.Equal(t, "FungibleToken", contracts[1].Name)
	assert.Equal(t, "Kibble", contracts[2].Name)
	assert.Equal(t, "KittyItems", contracts[3].Name)
	assert.Equal(t, "KittyItemsMarket", contracts[4].Name)
}

func Test_GetAccountByNameComplex(t *testing.T) {
	conf := generateComplexConfig()
	acc, err := conf.Accounts.ByName("account-4")
	assert.NoError(t, err)

	assert.Equal(t, "f8d6e0586b0a20c1", acc.Address.String())
}

func Test_GetDeploymentsByNetworkComplex(t *testing.T) {
	conf := generateComplexConfig()
	deployments := conf.Deployments.ByAccountAndNetwork("account-2", "testnet")

	assert.Equal(t,
		[]config.ContractDeployment{
			{Name: "FungibleToken", Args: []cadence.Value{}},
			{Name: "NonFungibleToken", Args: []cadence.Value{}},
			{Name: "Kibble", Args: []cadence.Value{}},
			{Name: "KittyItems", Args: []cadence.Value{}},
		},
		deployments[0].Contracts,
	)
}

func Test_GetNetworkByNameComplex(t *testing.T) {
	conf := generateComplexConfig()
	network, err := conf.Networks.ByName("emulator")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1.3569", network.Host)

	network, err = conf.Networks.ByName("testnet")
	assert.NoError(t, err)
	assert.Equal(t, network.Host, "access.devnet.nodes.onflow.org:9000")
	assert.Equal(t, network.Key, "5000676131ad3e22d853a3f75a5b5d0db4236d08dd6612e2baad771014b5266a242bccecc3522ff7207ac357dbe4f225c709d9b273ac484fed5d13976a39bdcd")
}
