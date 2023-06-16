/// Test contract is the standard library that provides testing functionality in Cadence.
///
access(all) contract Test {

    /// Blockchain emulates a real network.
    ///
    access(all) struct Blockchain {

        access(all) let backend: {BlockchainBackend}

        init(backend: {BlockchainBackend}) {
            self.backend = backend
        }

        /// Executes a script and returns the script return value and the status.
        /// `returnValue` field of the result will be `nil` if the script failed.
        ///
        access(all) fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult {
            return self.backend.executeScript(script, arguments)
        }

        /// Creates a signer account by submitting an account creation transaction.
        /// The transaction is paid by the service account.
        /// The returned account can be used to sign and authorize transactions.
        ///
        access(all) fun createAccount(): Account {
            return self.backend.createAccount()
        }

        /// Add a transaction to the current block.
        ///
        access(all) fun addTransaction(_ tx: Transaction) {
            self.backend.addTransaction(tx)
        }

        /// Executes the next transaction in the block, if any.
        /// Returns the result of the transaction, or nil if no transaction was scheduled.
        ///
        access(all) fun executeNextTransaction(): TransactionResult? {
            return self.backend.executeNextTransaction()
        }

        /// Commit the current block.
        /// Committing will fail if there are un-executed transactions in the block.
        ///
        access(all) fun commitBlock() {
            self.backend.commitBlock()
        }

        /// Executes a given transaction and commit the current block.
        ///
        access(all) fun executeTransaction(_ tx: Transaction): TransactionResult {
            self.addTransaction(tx)
            let txResult = self.executeNextTransaction()!
            self.commitBlock()
            return txResult
        }

        /// Executes a given set of transactions and commit the current block.
        ///
        access(all) fun executeTransactions(_ transactions: [Transaction]): [TransactionResult] {
            for tx in transactions {
                self.addTransaction(tx)
            }

            var results: [TransactionResult] = []
            for tx in transactions {
                let txResult = self.executeNextTransaction()!
                results.append(txResult)
            }

            self.commitBlock()
            return results
        }

        /// Deploys a given contract, and initilizes it with the arguments.
        ///
        access(all) fun deployContract(
            name: String,
            code: String,
            account: Account,
            arguments: [AnyStruct]
        ): Error? {
            return self.backend.deployContract(
                name: name,
                code: code,
                account: account,
                arguments: arguments
            )
        }

        /// Set the configuration to be used by the blockchain.
        /// Overrides any existing configuration.
        ///
        access(all) fun useConfiguration(_ configuration: Configuration) {
            self.backend.useConfiguration(configuration)
        }
    }

    access(all) struct Matcher {

        access(all) let test: fun(AnyStruct): Bool

        access(all) init(test: fun(AnyStruct): Bool) {
            self.test = test
        }

        /// Combine this matcher with the given matcher.
        /// Returns a new matcher that succeeds if this and the given matcher succeed.
        ///
        access(all) fun and(_ other: Matcher): Matcher {
            return Matcher(test: fun (value: AnyStruct): Bool {
                return self.test(value) && other.test(value)
            })
        }

        /// Combine this matcher with the given matcher.
        /// Returns a new matcher that succeeds if this or the given matcher succeed.
        /// If this matcher succeeds, then the other matcher would not be tested.
        ///
        access(all) fun or(_ other: Matcher): Matcher {
            return Matcher(test: fun (value: AnyStruct): Bool {
                return self.test(value) || other.test(value)
            })
        }
    }

    /// ResultStatus indicates status of a transaction or script execution.
    ///
    access(all) enum ResultStatus: UInt8 {
        access(all) case succeeded
        access(all) case failed
    }

    /// Result is the interface to be implemented by the various execution
    /// operations, such as transactions and scripts.
    ///
    access(all) struct interface Result {
        /// The resulted status of an executed operation.
        ///
        access(all) let status: ResultStatus
    }

    /// The result of a transaction execution.
    ///
    access(all) struct TransactionResult: Result {
        access(all) let status: ResultStatus
        access(all) let error: Error?

        init(status: ResultStatus, error: Error?) {
            self.status = status
            self.error = error
        }
    }

    /// The result of a script execution.
    ///
    access(all) struct ScriptResult: Result {
        access(all) let status: ResultStatus
        access(all) let returnValue: AnyStruct?
        access(all) let error: Error?

        init(status: ResultStatus, returnValue: AnyStruct?, error: Error?) {
            self.status = status
            self.returnValue = returnValue
            self.error = error
        }
    }

    // Error is returned if something has gone wrong.
    //
    access(all) struct Error {
        access(all) let message: String

        init(_ message: String) {
            self.message = message
        }
    }

    /// Account represents info about the account created on the blockchain.
    ///
    access(all) struct Account {
        access(all) let address: Address
        access(all) let publicKey: PublicKey

        init(address: Address, publicKey: PublicKey) {
            self.address = address
            self.publicKey = publicKey
        }
    }

    /// Configuration to be used by the blockchain.
    /// Can be used to set the address mappings.
    ///
    access(all) struct Configuration {
        access(all) let addresses: {String: Address}

        init(addresses: {String: Address}) {
            self.addresses = addresses
        }
    }

    /// Transaction that can be submitted and executed on the blockchain.
    ///
    access(all) struct Transaction {
        access(all) let code: String
        access(all) let authorizers: [Address]
        access(all) let signers: [Account]
        access(all) let arguments: [AnyStruct]

        init(code: String, authorizers: [Address], signers: [Account], arguments: [AnyStruct]) {
            self.code = code
            self.authorizers = authorizers
            self.signers = signers
            self.arguments = arguments
        }
    }

    /// BlockchainBackend is the interface to be implemented by the backend providers.
    ///
    access(all) struct interface BlockchainBackend {

        /// Executes a script and returns the script return value and the status.
        /// `returnValue` field of the result will be `nil` if the script failed.
        ///
        access(all) fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult

        /// Creates a signer account by submitting an account creation transaction.
        /// The transaction is paid by the service account.
        /// The returned account can be used to sign and authorize transactions.
        ///
        access(all) fun createAccount(): Account

        /// Add a transaction to the current block.
        ///
        access(all) fun addTransaction(_ tx: Transaction)

        /// Executes the next transaction in the block, if any.
        /// Returns the result of the transaction, or nil if no transaction was scheduled.
        ///
        access(all) fun executeNextTransaction(): TransactionResult?

        /// Commit the current block.
        /// Committing will fail if there are un-executed transactions in the block.
        ///
        access(all) fun commitBlock()

        /// Deploys a given contract, and initilizes it with the arguments.
        ///
        access(all) fun deployContract(
            name: String,
            code: String,
            account: Account,
            arguments: [AnyStruct]
        ): Error?

        /// Set the configuration to be used by the blockchain.
        /// Overrides any existing configuration.
        ///
        access(all) fun useConfiguration(_ configuration: Configuration)
    }

    /// Returns a new matcher that negates the test of the given matcher.
    ///
    access(all) fun not(_ matcher: Matcher): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return !matcher.test(value)
        })
    }

    /// Returns a new matcher that checks if the given test value is either
    /// a ScriptResult or TransactionResult and the ResultStatus is succeeded.
    /// Returns false in any other case.
    ///
    access(all) fun beSucceeded(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return (value as! {Result}).status == ResultStatus.succeeded
        })
    }

    /// Returns a new matcher that checks if the given test value is either
    /// a ScriptResult or TransactionResult and the ResultStatus is failed.
    /// Returns false in any other case.
    ///
    access(all) fun beFailed(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return (value as! {Result}).status == ResultStatus.failed
        })
    }

    /// Returns a new matcher that checks if the given test value is nil.
    ///
    access(all) fun beNil(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return value == nil
        })
    }

}
