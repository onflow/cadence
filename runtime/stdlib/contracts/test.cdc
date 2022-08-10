// Test contract is the standard library that provides testing functionality in Cadence.
//
// TODO: Name is yet to be finalized
//
pub contract Test {

    pub struct Blockchain {

        pub let backend: AnyStruct{BlockchainBackend}

        init(backend: AnyStruct{BlockchainBackend}) {
            self.backend = backend
        }

        // Executes a script and returns the script return value and the status.
        // `returnValue` field of the result will be `nil` if the script failed.
        //
        pub fun executeScript(_ script: String, _ args: [AnyStruct]): ScriptResult {
            return self.backend.executeScript(script, args)
        }

        // Creates a signer account by submitting an account creation transaction.
        // The transaction is paid by the service account.
        // The returned account can be used to sign and authorize transactions.
        //
        pub fun createAccount(): Account {
            return self.backend.createAccount()
        }

        // Add a transaction to the current block.
        //
        pub fun addTransaction(_ tx: Transaction) {
            self.backend.addTransaction(tx)
        }

        // Executes the next transaction in the block, if any.
        // Returns the result of the transaction, or nil if no transaction was scheduled.
        //
        pub fun executeNextTransaction(): TransactionResult? {
            return self.backend.executeNextTransaction()
        }

        // Commit the current block.
        // Committing will fail if there are un-executed transactions in the block.
        //
        pub fun commitBlock() {
            self.backend.commitBlock()
        }

        // Executes a given transaction and commit the current block.
        //
        pub fun executeTransaction(_ transaction: Transaction): TransactionResult {
            self.addTransaction(transaction)
            let txResult = self.executeNextTransaction()!
            self.commitBlock()
            return txResult
        }

        // Executes a given set of transactions and commit the current block.
        //
        pub fun executeTransactions(_ transactions: [Transaction]): [TransactionResult] {
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
        pub fun deployContract(
            _ name: String,
            _ code: String,
            _ account: Account,
            _ args: [AnyStruct]
        ): Error? {
            return self.backend.deployContract(name, code, account, args)
        }
    }

    // ResultStatus indicates status of a transaction or script execution.
    //
    pub enum ResultStatus: UInt8 {
        pub case succeeded
        pub case failed
    }

    // The result of a transaction execution.
    //
    pub struct TransactionResult {
        pub let status: ResultStatus
        pub let error:  Error?

        init(_ status: ResultStatus, _ error: Error) {
            self.status = status
            self.error = error
        }
    }

    // The result of a script execution.
    //
    pub struct ScriptResult {
        pub let status:      ResultStatus
        pub let returnValue: AnyStruct?
        pub let error:       Error?

        init(_ status: ResultStatus, _ returnValue: AnyStruct?, _ error: Error?) {
            self.status = status
            self.returnValue = returnValue
            self.error = error
        }
    }

    // Error is returned if something has gone wrong.
    //
    pub struct Error {
        pub let message: String

        init(_ message: String) {
            self.message = message
        }
    }

    // Account represents a user account in the blockchain.
    //
    pub struct Account {
        pub let address:    Address
        pub let accountKey: AccountKey
        pub let privateKey: [UInt8]

        init(_ address: Address, _ accountKey: AccountKey, _ privateKey: [UInt8]) {
            self.address = address
            self.accountKey = accountKey
            self.privateKey = privateKey
        }
    }

    // Transaction that can be submitted and executed on the blockchain.
    //
    pub struct Transaction {
        pub let code:       String
        pub let authorizer: Address?
        pub let signers:    [Account]
        pub let args:       [AnyStruct]

        init(_ code: String, _ authorizer: Address?, _ signers: [Account], _ args: [AnyStruct]) {
            self.code = code
            self.authorizer = authorizer
            self.signers = signers
            self.args = args
        }
    }

    // BlockchainBackend is the interface to be implemented by the backend providers.
    //
    pub struct interface BlockchainBackend {

        pub fun executeScript(_ script: String, _ args: [AnyStruct]): ScriptResult

        pub fun createAccount(): Account

        pub fun addTransaction(_ transaction: Transaction)

        pub fun executeNextTransaction(): TransactionResult?

        pub fun commitBlock()

        pub fun deployContract(
            _ name: String,
            _ code: String,
            _ account: Account,
            _ args: [AnyStruct]
        ): Error?
    }
}
