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

        pub fun executeScript(_ script: String): ScriptResult {
            return self.backend.executeScript(script)
        }

        pub fun createAccount(): Account {
            return self.backend.createAccount()
        }

        pub fun addTransaction(_ tx: Transaction) {
            self.backend.addTransaction(tx)
        }

        // Executes the next transaction, if any.
        // Returns the result of the transaction, or nil if no transaction was scheduled.
        //
        pub fun executeNextTransaction(): TransactionResult? {
            return self.backend.executeNextTransaction()
        }

        pub fun commitBlock() {
            self.backend.commitBlock()
        }

        //pub fun executeTransaction(_ transaction: Transaction): TransactionResult {
        //    self.addTransaction(transaction)
        //    let result = self.executeNextTransaction()!
        //    self.commitBlock()
        //    return result
        //}

        //pub fun executeTransactions(_ transactions: [Transaction]): [TransactionResult] {
        //    for transaction in transactions {
        //        self.addTransaction(transaction)
        //    }

        //    var results: [TransactionResult] = []
        //    for transaction in transactions {
        //        let result = self.executeNextTransaction()!
        //        results.append(result)
        //    }

        //    self.commitBlock()
        //    return results
        //}
    }

    pub enum ResultStatus: UInt8 {
        pub case succeeded
        pub case failed
    }

    pub struct TransactionResult {
        pub let status: ResultStatus

        init(_ status: ResultStatus) {
            self.status = status
        }
    }

    pub struct ScriptResult {
        pub let status:      ResultStatus
        pub let returnValue: AnyStruct?

        init(_ status: ResultStatus, _ returnValue: AnyStruct?) {
            self.status = status
            self.returnValue = returnValue
        }
    }

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

    pub struct Transaction {
        pub let code:       String
        pub let authorizer: Address?
        pub let signers:    [Account]

        init(_ code: String, _ authorizer: Address?, _ signers: [Account]) {
            self.code = code
            self.authorizer = authorizer
            self.signers = signers
        }
    }

    // BlockchainBackend is the interface to be implemented by the backend providers.
    //
    pub struct interface BlockchainBackend {

        pub fun executeScript(_ script: String): ScriptResult

        pub fun createAccount(): Account

        pub fun addTransaction(_ transaction: Transaction)

        pub fun executeNextTransaction(): TransactionResult?

        pub fun commitBlock()
    }
}
