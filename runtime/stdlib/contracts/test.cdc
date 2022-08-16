/// Test contract is the standard library that provides testing functionality in Cadence.
///
/// TODO: Name is yet to be finalized
///
pub contract Test {

    /// Convenient function to fail a test.
    /// Is equivalent to calling `assert(false)`.
    ///
    pub fun fail() {
        assert(false)
    }

    pub struct interface Matcher {

        pub fun test(_ value: Any): Bool

        pub fun and(_ other: AnyStruct{Matcher}): AnyStruct{Matcher}

        pub fun or(_ other: AnyStruct{Matcher}): AnyStruct{Matcher}
    }

    /// Blockchain emulates a real network.
    ///
    pub struct Blockchain {

        pub let backend: AnyStruct{BlockchainBackend}

        init(backend: AnyStruct{BlockchainBackend}) {
            self.backend = backend
        }

        /// Executes a script and returns the script return value and the status.
        /// `returnValue` field of the result will be `nil` if the script failed.
        ///
        pub fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult {
            return self.backend.executeScript(script, arguments)
        }

        /// Creates a signer account by submitting an account creation transaction.
        /// The transaction is paid by the service account.
        /// The returned account can be used to sign and authorize transactions.
        ///
        pub fun createAccount(): Account {
            return self.backend.createAccount()
        }

        /// Add a transaction to the current block.
        ///
        pub fun addTransaction(_ tx: Transaction) {
            self.backend.addTransaction(tx)
        }

        /// Executes the next transaction in the block, if any.
        /// Returns the result of the transaction, or nil if no transaction was scheduled.
        ///
        pub fun executeNextTransaction(): TransactionResult? {
            return self.backend.executeNextTransaction()
        }

        /// Commit the current block.
        /// Committing will fail if there are un-executed transactions in the block.
        ///
        pub fun commitBlock() {
            self.backend.commitBlock()
        }

        /// Executes a given transaction and commit the current block.
        ///
        pub fun executeTransaction(_ transaction: Transaction): TransactionResult {
            self.addTransaction(transaction)
            let txResult = self.executeNextTransaction()!
            self.commitBlock()
            return txResult
        }

        /// Executes a given set of transactions and commit the current block.
        ///
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
    }

    /// ResultStatus indicates status of a transaction or script execution.
    ///
    pub enum ResultStatus: UInt8 {
        pub case succeeded
        pub case failed
    }

    /// The result of a transaction execution.
    ///
    pub struct TransactionResult {
        pub let status: ResultStatus

        init(status: ResultStatus) {
            self.status = status
        }
    }

    /// The result of a script execution.
    ///
    pub struct ScriptResult {
        pub let status: ResultStatus
        pub let returnValue: AnyStruct?

        init(status: ResultStatus, returnValue: AnyStruct?) {
            self.status = status
            self.returnValue = returnValue
        }
    }

    /// Account represents info about the account created on the blockchain.
    ///
    pub struct Account {
        pub let address: Address
        pub let publicKey: PublicKey

        init(address: Address, publicKey: PublicKey) {
            self.address = address
            self.publicKey = publicKey
        }
    }

    /// Transaction that can be submitted and executed on the blockchain.
    ///
    pub struct Transaction {
        pub let code: String
        pub let authorizers: [Address]
        pub let signers: [Account]
        pub let arguments: [AnyStruct]

        init(code: String, authorizers: [Address], signers: [Account], arguments: [AnyStruct]) {
            self.code = code
            self.authorizers = authorizers
            self.signers = signers
            self.arguments = arguments
        }
    }

    /// BlockchainBackend is the interface to be implemented by the backend providers.
    ///
    pub struct interface BlockchainBackend {

        pub fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult

        pub fun createAccount(): Account

        pub fun addTransaction(_ transaction: Transaction)

        pub fun executeNextTransaction(): TransactionResult?

        pub fun commitBlock()
    }
}
