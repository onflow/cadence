package vm

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type testAccountHandler struct {
	accountIDs                 map[common.Address]uint64
	generateAccountID          func(address common.Address) (uint64, error)
	getAccountBalance          func(address common.Address) (uint64, error)
	getAccountAvailableBalance func(address common.Address) (uint64, error)
	commitStorageTemporarily   func(inter *interpreter.Interpreter) error
	getStorageUsed             func(address common.Address) (uint64, error)
	getStorageCapacity         func(address common.Address) (uint64, error)
	validatePublicKey          func(key *stdlib.PublicKey) error
	verifySignature            func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm sema.SignatureAlgorithm,
		hashAlgorithm sema.HashAlgorithm,
	) (
		bool,
		error,
	)
	blsVerifyPOP     func(publicKey *stdlib.PublicKey, signature []byte) (bool, error)
	hash             func(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error)
	getAccountKey    func(address common.Address, index int) (*stdlib.AccountKey, error)
	accountKeysCount func(address common.Address) (uint64, error)
	emitEvent        func(
		inter *interpreter.Interpreter,
		eventType *sema.CompositeType,
		values []interpreter.Value,
		locationRange interpreter.LocationRange,
	)
	addAccountKey func(
		address common.Address,
		key *stdlib.PublicKey,
		algo sema.HashAlgorithm,
		weight int,
	) (
		*stdlib.AccountKey,
		error,
	)
	revokeAccountKey       func(address common.Address, index int) (*stdlib.AccountKey, error)
	getAccountContractCode func(location common.AddressLocation) ([]byte, error)
	parseAndCheckProgram   func(
		code []byte,
		location common.Location,
		getAndSetProgram bool,
	) (
		*interpreter.Program,
		error,
	)
	updateAccountContractCode func(location common.AddressLocation, code []byte) error
	recordContractUpdate      func(location common.AddressLocation, value *interpreter.CompositeValue)
	contractUpdateRecorded    func(location common.AddressLocation) bool
	interpretContract         func(
		location common.AddressLocation,
		program *interpreter.Program,
		name string,
		invocation stdlib.DeployedContractConstructorInvocation,
	) (
		*interpreter.CompositeValue,
		error,
	)
	temporarilyRecordCode     func(location common.AddressLocation, code []byte)
	removeAccountContractCode func(location common.AddressLocation) error
	recordContractRemoval     func(location common.AddressLocation)
	getAccountContractNames   func(address common.Address) ([]string, error)
}

var _ stdlib.AccountHandler = &testAccountHandler{}

func (t *testAccountHandler) GenerateAccountID(address common.Address) (uint64, error) {
	if t.generateAccountID == nil {
		if t.accountIDs == nil {
			t.accountIDs = map[common.Address]uint64{}
		}
		t.accountIDs[address]++
		return t.accountIDs[address], nil
	}
	return t.generateAccountID(address)
}

func (t *testAccountHandler) GetAccountBalance(address common.Address) (uint64, error) {
	if t.getAccountBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountBalance"))
	}
	return t.getAccountBalance(address)
}

func (t *testAccountHandler) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	if t.getAccountAvailableBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountAvailableBalance"))
	}
	return t.getAccountAvailableBalance(address)
}

func (t *testAccountHandler) CommitStorageTemporarily(inter *interpreter.Interpreter) error {
	if t.commitStorageTemporarily == nil {
		panic(errors.NewUnexpectedError("unexpected call to CommitStorageTemporarily"))
	}
	return t.commitStorageTemporarily(inter)
}

func (t *testAccountHandler) GetStorageUsed(address common.Address) (uint64, error) {
	if t.getStorageUsed == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageUsed"))
	}
	return t.getStorageUsed(address)
}

func (t *testAccountHandler) GetStorageCapacity(address common.Address) (uint64, error) {
	if t.getStorageCapacity == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageCapacity"))
	}
	return t.getStorageCapacity(address)
}

func (t *testAccountHandler) ValidatePublicKey(key *stdlib.PublicKey) error {
	if t.validatePublicKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to ValidatePublicKey"))
	}
	return t.validatePublicKey(key)
}

func (t *testAccountHandler) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm sema.SignatureAlgorithm,
	hashAlgorithm sema.HashAlgorithm,
) (
	bool,
	error,
) {
	if t.verifySignature == nil {
		panic(errors.NewUnexpectedError("unexpected call to VerifySignature"))
	}
	return t.verifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (t *testAccountHandler) BLSVerifyPOP(publicKey *stdlib.PublicKey, signature []byte) (bool, error) {
	if t.blsVerifyPOP == nil {
		panic(errors.NewUnexpectedError("unexpected call to BLSVerifyPOP"))
	}
	return t.blsVerifyPOP(publicKey, signature)
}

func (t *testAccountHandler) Hash(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error) {
	if t.hash == nil {
		panic(errors.NewUnexpectedError("unexpected call to Hash"))
	}
	return t.hash(data, tag, algorithm)
}

func (t *testAccountHandler) GetAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	if t.getAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountKey"))
	}
	return t.getAccountKey(address, index)
}

func (t *testAccountHandler) AccountKeysCount(address common.Address) (uint64, error) {
	if t.accountKeysCount == nil {
		panic(errors.NewUnexpectedError("unexpected call to AccountKeysCount"))
	}
	return t.accountKeysCount(address)
}

func (t *testAccountHandler) EmitEvent(
	inter *interpreter.Interpreter,
	eventType *sema.CompositeType,
	values []interpreter.Value,
	locationRange interpreter.LocationRange,
) {
	if t.emitEvent == nil {
		panic(errors.NewUnexpectedError("unexpected call to EmitEvent"))
	}
	t.emitEvent(
		inter,
		eventType,
		values,
		locationRange,
	)
}

func (t *testAccountHandler) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (
	*stdlib.AccountKey,
	error,
) {
	if t.addAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to AddAccountKey"))
	}
	return t.addAccountKey(
		address,
		key,
		algo,
		weight,
	)
}

func (t *testAccountHandler) RevokeAccountKey(address common.Address, index int) (*stdlib.AccountKey, error) {
	if t.revokeAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to RevokeAccountKey"))
	}
	return t.revokeAccountKey(address, index)
}

func (t *testAccountHandler) GetAccountContractCode(location common.AddressLocation) ([]byte, error) {
	if t.getAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractCode"))
	}
	return t.getAccountContractCode(location)
}

func (t *testAccountHandler) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	if t.parseAndCheckProgram == nil {
		panic(errors.NewUnexpectedError("unexpected call to ParseAndCheckProgram"))
	}
	return t.parseAndCheckProgram(code, location, getAndSetProgram)
}

func (t *testAccountHandler) UpdateAccountContractCode(location common.AddressLocation, code []byte) error {
	if t.updateAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to UpdateAccountContractCode"))
	}
	return t.updateAccountContractCode(location, code)
}

func (t *testAccountHandler) RecordContractUpdate(location common.AddressLocation, value *interpreter.CompositeValue) {
	if t.recordContractUpdate == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractUpdate"))
	}
	t.recordContractUpdate(location, value)
}

func (t *testAccountHandler) ContractUpdateRecorded(location common.AddressLocation) bool {
	if t.contractUpdateRecorded == nil {
		panic(errors.NewUnexpectedError("unexpected call to ContractUpdateRecorded"))
	}
	return t.contractUpdateRecorded(location)
}

func (t *testAccountHandler) InterpretContract(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	*interpreter.CompositeValue,
	error,
) {
	if t.interpretContract == nil {
		panic(errors.NewUnexpectedError("unexpected call to InterpretContract"))
	}
	return t.interpretContract(
		location,
		program,
		name,
		invocation,
	)
}

func (t *testAccountHandler) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	if t.temporarilyRecordCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to TemporarilyRecordCode"))
	}
	t.temporarilyRecordCode(location, code)
}

func (t *testAccountHandler) RemoveAccountContractCode(location common.AddressLocation) error {
	if t.removeAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to RemoveAccountContractCode"))
	}
	return t.removeAccountContractCode(location)
}

func (t *testAccountHandler) RecordContractRemoval(location common.AddressLocation) {
	if t.recordContractRemoval == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractRemoval"))
	}
	t.recordContractRemoval(location)
}

func (t *testAccountHandler) GetAccountContractNames(address common.Address) ([]string, error) {
	if t.getAccountContractNames == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractNames"))
	}
	return t.getAccountContractNames(address)
}

func (t *testAccountHandler) StartContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) EndContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) IsContractBeingAdded(common.AddressLocation) bool {
	// NO-OP
	return false
}
