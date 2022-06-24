package integration

import (
	"fmt"
	"net/url"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/gateway"
	"github.com/onflow/flow-cli/pkg/flowkit/output"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
)

//go:generate go run github.com/vektra/mockery/cmd/mockery --name flowClient --filename mock_flow_test.go --inpkg
type flowClient interface {
	Initialize(configPath string, numberOfAccounts int) error
	GetClientAccount(name string) *ClientAccount
	GetActiveClientAccount() *ClientAccount
	SetActiveClientAccount(name string) error
	ExecuteScript(location *url.URL, args []cadence.Value) (cadence.Value, error)
	DeployContract(address flow.Address, name string, location *url.URL) (*flow.Account, error)
	SendTransaction(authorizers []flow.Address, location *url.URL, args []cadence.Value) (*flow.TransactionResult, error)
	GetAccount(address flow.Address) (*flow.Account, error)
	CreateAccount() (*flow.Account, error)
}

var _ flowClient = flowkitClient{}

// todo: check if we need this struct at all, could we just use the flow.Account returned from the flowkit

type ClientAccount struct {
	*flow.Account
	Name string
}

var names = []string{
	"Alice", "Bob", "Charlie",
	"Dave", "Eve", "Faythe",
	"Grace", "Heidi", "Ivan",
	"Judy", "Michael", "Niaj",
	"Olivia", "Oscar", "Peggy",
	"Rupert", "Sybil", "Ted",
	"Victor", "Walter",
}

type flowkitClient struct {
	services      *services.Services
	loader        flowkit.ReaderWriter
	state         *flowkit.State
	accounts      []*ClientAccount
	activeAccount *ClientAccount
}

func NewFlowkitClient(loader flowkit.ReaderWriter) *flowkitClient {
	return &flowkitClient{
		loader: loader,
	}
}

func (f flowkitClient) Initialize(configPath string, numberOfAccounts int) error {
	state, err := flowkit.Load([]string{configPath}, f.loader)
	if err != nil {
		return err
	}
	f.state = state

	logger := output.NewStdoutLogger(output.NoneLog)

	serviceAccount, err := state.EmulatorServiceAccount()
	if err != nil {
		return err
	}

	grpcGateway := gateway.NewEmulatorGateway(serviceAccount)
	if err != nil {
		return err
	}

	f.services = services.NewServices(grpcGateway, state, logger)

	if numberOfAccounts > len(names) {
		return fmt.Errorf(fmt.Sprintf("can only use up to %d accounts", len(names)))
	}

	f.accounts = make([]*ClientAccount, numberOfAccounts)
	for i := 0; i < numberOfAccounts; i++ {
		account, err := f.CreateAccount()
		if err != nil {
			return err
		}

		f.accounts[i] = &ClientAccount{
			Account: account,
			Name:    names[i],
		}
	}
	f.activeAccount = f.accounts[0] // make first active by default
	return nil
}

func (f flowkitClient) GetClientAccount(name string) *ClientAccount {
	for _, account := range f.accounts {
		if account.Name == name {
			return account
		}
	}
	return nil
}

func (f flowkitClient) SetActiveClientAccount(name string) error {
	account := f.GetClientAccount(name)
	if account == nil {
		return fmt.Errorf(fmt.Sprintf("account with a name %s not found", name))
	}
	f.activeAccount = account
	return nil
}

func (f flowkitClient) GetActiveClientAccount() *ClientAccount {
	return f.activeAccount
}

func (f flowkitClient) ExecuteScript(
	location *url.URL,
	args []cadence.Value,
) (cadence.Value, error) {
	code, err := f.state.ReadFile(location.Path)
	if err != nil {
		return nil, err
	}

	return f.services.Scripts.Execute(code, args, "", "") // todo check if it's ok that path is empty for resolving imports
}

func (f flowkitClient) DeployContract(
	address flow.Address,
	name string,
	location *url.URL,
) (*flow.Account, error) {
	code, err := f.state.ReadFile(location.Path)
	if err != nil {
		return nil, err
	}

	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}

	account := createSigner(address, service)
	return f.services.Accounts.AddContract(account, name, code, true)
}

func (f flowkitClient) SendTransaction(
	authorizers []flow.Address,
	location *url.URL,
	args []cadence.Value,
) (*flow.TransactionResult, error) {
	code, err := f.state.ReadFile(location.Path)
	if err != nil {
		return nil, err
	}

	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}
	// if no authorizers defined use the service as default
	if authorizers == nil {
		authorizers = []flow.Address{service.Address()}
	}

	tx, err := f.services.Transactions.Build(
		service.Address(),
		authorizers,
		service.Address(),
		service.Key().Index(),
		code,
		"",
		flow.DefaultTransactionGasLimit,
		args,
		"",
		true,
	)
	if err != nil {
		return nil, err
	}

	// sign with service as proposer
	tx, err = sign(service, tx)
	if err != nil {
		return nil, err
	}
	// sign with all authorizers
	for _, auth := range authorizers {
		tx, err = sign(createSigner(auth, service), tx)
		if err != nil {
			return nil, err
		}
	}
	// sign with service as payer
	tx, err = sign(service, tx)
	if err != nil {
		return nil, err
	}

	_, res, err := f.services.Transactions.SendSigned(tx.FlowTransaction().Encode(), true)
	return res, err
}

func (f flowkitClient) GetAccount(address flow.Address) (*flow.Account, error) {
	return f.services.Accounts.Get(address)
}

func (f flowkitClient) CreateAccount() (*flow.Account, error) {
	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, err
	}
	serviceKey, err := service.Key().PrivateKey()
	if err != nil {
		return nil, err
	}

	return f.services.Accounts.Create(
		service,
		[]crypto.PublicKey{(*serviceKey).PublicKey()},
		[]int{flow.AccountKeyWeightThreshold},
		[]crypto.SignatureAlgorithm{crypto.ECDSA_P256},
		[]crypto.HashAlgorithm{crypto.SHA3_256},
		nil,
	)
}

// Helpers
//

// createSigner creates a new flowkit account used for signing but using the key of the existing account.
func createSigner(address flow.Address, account *flowkit.Account) *flowkit.Account {
	signer := &flowkit.Account{}
	signer.SetAddress(address)
	signer.SetKey(account.Key())
	return signer
}

// sign sets the signer on a transaction and calls the sign method.
func sign(signer *flowkit.Account, tx *flowkit.Transaction) (*flowkit.Transaction, error) {
	err := tx.SetSigner(signer)
	if err != nil {
		return nil, err
	}

	tx, err = tx.Sign()
	if err != nil {
		return nil, err
	}

	return tx, nil
}
