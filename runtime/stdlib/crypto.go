package stdlib

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser2"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/internal"
	"github.com/onflow/cadence/runtime/trampoline"
)

var CryptoChecker = func() *sema.Checker {

	code := internal.MustAssetString("contracts/crypto.cdc")

	program, err := parser2.ParseProgram(code)
	if err != nil {
		panic(err)
	}

	location := ast.IdentifierLocation("Crypto")

	var checker *sema.Checker
	checker, err = sema.NewChecker(
		program,
		location,
		sema.WithPredeclaredValues(BuiltinFunctions.ToValueDeclarations()),
		sema.WithPredeclaredTypes(BuiltinTypes.ToTypeDeclarations()),
	)
	if err != nil {
		panic(err)
	}

	err = checker.Check()
	if err != nil {
		panic(err)
	}

	return checker
}()

var cryptoSignatureVerifierInterfaceType = CryptoChecker.GlobalTypes["SignatureVerifier"].Type.(*sema.InterfaceType)

var cryptoSignatureVerifierRestrictedType = &sema.RestrictedType{
	Type: &sema.AnyType{},
	Restrictions: []*sema.InterfaceType{
		cryptoSignatureVerifierInterfaceType,
	},
}

var cryptoContractInitializerTypes = []sema.Type{
	cryptoSignatureVerifierRestrictedType,
}

var cryptoContractVerifySignatureFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) trampoline.Trampoline {
		// TODO:
		panic("TODO")
	},
)

var cryptoContractSignatureVerifier = func() *interpreter.CompositeValue {
	implementationTypeID :=
		sema.TypeID(string(cryptoSignatureVerifierInterfaceType.ID()) + "Impl")

	signatureVerifier := interpreter.NewCompositeValue(
		CryptoChecker.Location,
		implementationTypeID,
		common.CompositeKindStructure,
		nil,
		nil,
	)

	signatureVerifier.Functions = map[string]interpreter.FunctionValue{
		"verify": cryptoContractVerifySignatureFunction,
	}

	return signatureVerifier
}()

var cryptoContractInitializerArguments = []interpreter.Value{
	cryptoContractSignatureVerifier,
}

func NewCryptoContract(
	inter *interpreter.Interpreter,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {

	value, err := inter.InvokeFunctionValue(
		constructor,
		cryptoContractInitializerArguments,
		cryptoContractInitializerTypes,
		cryptoContractInitializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	return compositeValue, nil
}
