package integration

import (
	"github.com/onflow/cadence"
	"github.com/onflow/flow-cli/pkg/flowkit"
	"github.com/onflow/flow-cli/pkg/flowkit/services"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
)

type flowClient interface {
	ExecuteScript(
		code []byte,
		args []cadence.Value,
		scriptPath string,
		network string,
	) (cadence.Value, error)

	DeployContract(
		account *flowkit.Account,
		contractName string,
		contractSource []byte,
		update bool,
	) (*flow.Account, error)

	SendTransaction(
		authorizers []flow.Address,
		code []byte,
		args []cadence.Value,
	) (*flow.Transaction, *flow.TransactionResult, error)

	GetAccount(address flow.Address) (*flow.Account, error)

	CreateAccount() (*flow.Account, error)
}

var _ flowClient = flowkitClient{}

type flowkitClient struct {
	services *services.Services
	state    *flowkit.State
}

func (f flowkitClient) ExecuteScript(
	code []byte,
	args []cadence.Value,
	scriptPath string,
	network string,
) (cadence.Value, error) {
	return f.services.Scripts.Execute(code, args, scriptPath, network)
}

func (f flowkitClient) DeployContract(
	account *flowkit.Account,
	contractName string,
	contractSource []byte,
	update bool,
) (*flow.Account, error) {
	return f.services.Accounts.AddContract(account, contractName, contractSource, update)
}

func (f flowkitClient) SendTransaction(
	authorizers []flow.Address,
	code []byte,
	args []cadence.Value,
) (*flow.Transaction, *flow.TransactionResult, error) {
	service, err := f.state.EmulatorServiceAccount()
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	// sign with service as proposer
	tx, err = sign(service, tx)
	if err != nil {
		return nil, nil, err
	}
	// sign with all authorizers
	for _, auth := range authorizers {
		signer := &flowkit.Account{}
		signer.SetAddress(auth)
		signer.SetKey(service.Key())

		tx, err = sign(signer, tx)
		if err != nil {
			return nil, nil, err
		}
	}
	// sign with service as payer
	tx, err = sign(service, tx)
	if err != nil {
		return nil, nil, err
	}

	return f.services.Transactions.SendSigned(tx.FlowTransaction().Encode(), true)
}

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
