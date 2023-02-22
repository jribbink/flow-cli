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

package accounts

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/config"
	"github.com/onflow/flow-cli/pkg/flowkit/gateway"
	"github.com/onflow/flow-cli/pkg/flowkit/output"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/onflow/flow-cli/pkg/flowkit/util"
)

// createInteractive is used when user calls a default account create command without any provided values.
//
// This process takes the user through couple of steps with prompts asking for them to provide name and network,
// and it then uses account creation APIs to automatically create the account on the network as well as save it.
func createInteractive(state *flowkit.State) error {
	log := output.NewStdoutLogger(output.InfoLog)
	name := output.AccountNamePrompt(state.Accounts()) // todo check for duplicate names
	networkName, selectedNetwork := output.CreateAccountNetworkPrompt()
	privateFile := fmt.Sprintf("%s.pkey", name)

	// create new gateway based on chosen network
	gw, err := gateway.NewGrpcGateway(selectedNetwork.Host)
	if err != nil {
		return err
	}
	service := services.NewServices(gw, state, output.NewStdoutLogger(output.NoneLog))

	key, err := service.Keys.Generate("", crypto.ECDSA_P256)
	if err != nil {
		return err
	}

	log.StartProgress(fmt.Sprintf("Creating account %s on %s...", name, networkName))

	var account *flowkit.Account
	if selectedNetwork == config.DefaultEmulatorNetwork() {
		account, err = createEmulatorAccount(state, service, name, key)
		log.StopProgress()
		log.Info(output.Italic("\nPlease note that the newly-created account will only be available while you keep the emulator service running. If you restart the emulator service, all accounts will be reset. If you want to persist accounts between restarts, please use the '--persist' flag when starting the flow emulator.\n"))
	} else {
		account, err = createNetworkAccount(state, service, name, key, privateFile, selectedNetwork)
		log.StopProgress()
	}
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf(
		"%s New account created with address %s and name %s on %s network.\n",
		output.SuccessEmoji(),
		output.Bold(fmt.Sprintf("0x%s", account.Address().String())),
		output.Bold(name),
		output.Bold(networkName)),
	)

	state.Accounts().AddOrUpdate(account)
	err = state.SaveDefault()
	if err != nil {
		return err
	}

	items := []string{
		"Here’s a summary of all the actions that were taken",
		fmt.Sprintf("Added the new account to %s.", output.Bold("flow.json")),
	}
	if selectedNetwork != config.DefaultEmulatorNetwork() {
		items = append(items,
			fmt.Sprintf("Saved the private key to %s.", output.Bold(privateFile)),
			fmt.Sprintf("Added %s to %s.", output.Bold(privateFile), output.Bold(".gitignore")),
		)
	}
	outputList(log, items, false)

	return nil
}

// createNetworkAccount using the account creation API and return the newly created account address.
func createNetworkAccount(
	state *flowkit.State,
	services *services.Services,
	name string,
	key crypto.PrivateKey,
	privateFile string,
	network config.Network,
) (*flowkit.Account, error) {
	networkAccount := &lilicoAccount{
		PublicKey: strings.TrimPrefix(key.PublicKey().String(), "0x"),
	}

	id, err := networkAccount.create(network.Name)
	if err != nil {
		return nil, err
	}

	result, err := getAccountCreationResult(services, id)
	if err != nil {
		return nil, err
	}

	events := flowkit.EventsFromTransaction(result)
	address := events.GetCreatedAddresses()
	if len(address) == 0 {
		return nil, fmt.Errorf("account creation error")
	}

	err = util.AddToGitIgnore(privateFile, state.ReaderWriter())
	if err != nil {
		return nil, err
	}

	err = state.ReaderWriter().WriteFile(privateFile, []byte(key.String()), os.FileMode(0644))
	if err != nil {
		return nil, fmt.Errorf("failed saving private key: %w", err)
	}

	return flowkit.NewAccount(name).SetAddress(*address[0]).SetKey(
		flowkit.NewFileAccountKey(privateFile, 0, crypto.ECDSA_P256, crypto.SHA3_256),
	), nil
}

func createEmulatorAccount(
	state *flowkit.State,
	service *services.Services,
	name string,
	key crypto.PrivateKey,
) (*flowkit.Account, error) {
	signer, err := state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}

	networkAccount, err := service.Accounts.Create(
		signer,
		[]crypto.PublicKey{key.PublicKey()},
		[]int{flow.AccountKeyWeightThreshold},
		[]crypto.SignatureAlgorithm{crypto.ECDSA_P256},
		[]crypto.HashAlgorithm{crypto.SHA3_256},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return flowkit.NewAccount(name).SetAddress(networkAccount.Address).SetKey(
		flowkit.NewHexAccountKeyFromPrivateKey(0, crypto.SHA3_256, key),
	), nil
}

func getAccountCreationResult(services *services.Services, id flow.Identifier) (*flow.TransactionResult, error) {
	_, result, err := services.Transactions.GetStatus(id, true)
	if err != nil {
		if status.Code(err) == codes.NotFound { // if transaction not yet propagated, wait for it
			time.Sleep(1 * time.Second)
			return getAccountCreationResult(services, id)
		}
		return nil, err
	}

	return result, nil
}

// lilicoAccount contains all the data needed for interaction with lilico account creation API.
type lilicoAccount struct {
	PublicKey          string `json:"publicKey"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	HashAlgorithm      string `json:"hashAlgorithm"`
	Weight             int    `json:"weight"`
}

type lilicoResponse struct {
	Data struct {
		TxId string `json:"txId"`
	} `json:"data"`
}

var accountToken = ""

// create a new account using the lilico API and parsing the response, returning account creation transaction ID.
func (l *lilicoAccount) create(network string) (flow.Identifier, error) {
	// fix to the defaults as we don't support other values
	l.HashAlgorithm = crypto.SHA3_256.String()
	l.SignatureAlgorithm = crypto.ECDSA_P256.String()
	l.Weight = flow.AccountKeyWeightThreshold

	data, err := json.Marshal(l)
	if err != nil {
		return flow.EmptyID, err
	}

	apiNetwork := ""
	if network == config.DefaultTestnetNetwork().Name {
		apiNetwork = "/testnet"
	}

	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("https://openapi.lilico.org/v1/address%s", apiNetwork),
		bytes.NewReader(data),
	)
	if err != nil {
		return flow.EmptyID, fmt.Errorf("could not create an account: %w", err)
	}

	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	request.Header.Add("Authorization", accountToken)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // lilico api doesn't yet have a valid cert, todo reevaluate
		},
	}
	res, err := client.Do(request)
	if err != nil {
		return flow.EmptyID, fmt.Errorf("could not create an account: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var lilicoRes lilicoResponse
	err = json.Unmarshal(body, &lilicoRes)
	if err != nil {
		return flow.EmptyID, fmt.Errorf("could not create an account: %w", err)
	}

	return flow.HexToID(lilicoRes.Data.TxId), nil
}

// outputList helper for printing lists
func outputList(log *output.StdoutLogger, items []string, numbered bool) {
	log.Info(fmt.Sprintf("%s:", items[0]))
	items = items[1:]
	for n, item := range items {
		sep := " -"
		if numbered {
			sep = fmt.Sprintf(" %d.", n+1)
		}
		log.Info(fmt.Sprintf("%s %s", sep, item))
	}
	log.Info("")
}
