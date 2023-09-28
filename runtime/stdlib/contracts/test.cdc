/// Test contract is the standard library that provides testing functionality in Cadence.
///
access(all) contract Test {

    /// backend emulates a real network.
    ///
    access(self) let backend: AnyStruct{BlockchainBackend}

    init(backend: AnyStruct{BlockchainBackend}) {
        self.backend = backend
    }

    /// Executes a script and returns the script return value and the status.
    /// `returnValue` field of the result will be `nil` if the script failed.
    ///
    access(all)
    fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult {
        return self.backend.executeScript(script, arguments)
    }

    /// Creates a signer account by submitting an account creation transaction.
    /// The transaction is paid by the service account.
    /// The returned account can be used to sign and authorize transactions.
    ///
    access(all)
    fun createAccount(): Account {
        return self.backend.createAccount()
    }

    /// Returns the account for the given address.
    ///
    access(all)
    fun getAccount(_ address: Address): Account {
        return self.backend.getAccount(address)
    }

    /// Add a transaction to the current block.
    ///
    access(all)
    fun addTransaction(_ tx: Transaction) {
        self.backend.addTransaction(tx)
    }

    /// Executes the next transaction in the block, if any.
    /// Returns the result of the transaction, or nil if no transaction was scheduled.
    ///
    access(all)
    fun executeNextTransaction(): TransactionResult? {
        return self.backend.executeNextTransaction()
    }

    /// Commit the current block.
    /// Committing will fail if there are un-executed transactions in the block.
    ///
    access(all)
    fun commitBlock() {
        self.backend.commitBlock()
    }

    /// Executes a given transaction and commit the current block.
    ///
    access(all)
    fun executeTransaction(_ tx: Transaction): TransactionResult {
        self.addTransaction(tx)
        let txResult = self.executeNextTransaction()!
        self.commitBlock()
        return txResult
    }

    /// Executes a given set of transactions and commit the current block.
    ///
    access(all)
    fun executeTransactions(_ transactions: [Transaction]): [TransactionResult] {
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
    access(all)
    fun deployContract(
        name: String,
        path: String,
        arguments: [AnyStruct]
    ): Error? {
        return self.backend.deployContract(
            name: name,
            path: path,
            arguments: arguments
        )
    }

    /// Returns all the logs from the blockchain, up to the calling point.
    ///
    access(all)
    fun logs(): [String] {
        return self.backend.logs()
    }

    /// Returns the service account of the blockchain. Can be used to sign
    /// transactions with this account.
    ///
    access(all)
    fun serviceAccount(): Account {
        return self.backend.serviceAccount()
    }

    /// Returns all events emitted from the blockchain.
    ///
    access(all)
    fun events(): [AnyStruct] {
        return self.backend.events(nil)
    }

    /// Returns all events emitted from the blockchain,
    /// filtered by type.
    ///
    access(all)
    fun eventsOfType(_ type: Type): [AnyStruct] {
        return self.backend.events(type)
    }

    /// Resets the state of the blockchain to the given height.
    ///
    access(all)
    fun reset(to height: UInt64) {
        self.backend.reset(to: height)
    }

    /// Moves the time of the blockchain by the given delta,
    /// which should be passed in the form of seconds.
    ///
    access(all)
    fun moveTime(by delta: Fix64) {
        self.backend.moveTime(by: delta)
    }

    /// Creates a snapshot of the blockchain, at the
    /// current ledger state, with the given name.
    ///
    access(all)
    fun createSnapshot(name: String) {
        let err = self.backend.createSnapshot(name: name)
        if err != nil {
            panic(err!.message)
        }
    }

    /// Loads a snapshot of the blockchain, with the
    /// given name, and updates the current ledger
    /// state.
    ///
    access(all)
    fun loadSnapshot(name: String) {
        let err = self.backend.loadSnapshot(name: name)
        if err != nil {
            panic(err!.message)
        }
    }

    access(all) struct Matcher {

        access(all) let test: ((AnyStruct): Bool)

        init(test: ((AnyStruct): Bool)) {
            self.test = test
        }

        /// Combine this matcher with the given matcher.
        /// Returns a new matcher that succeeds if this and the given matcher succeed.
        ///
        access(all)
        fun and(_ other: Matcher): Matcher {
            return Matcher(test: fun (value: AnyStruct): Bool {
                return self.test(value) && other.test(value)
            })
        }

        /// Combine this matcher with the given matcher.
        /// Returns a new matcher that succeeds if this or the given matcher succeed.
        /// If this matcher succeeds, then the other matcher would not be tested.
        ///
        access(all)
        fun or(_ other: Matcher): Matcher {
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
        /// The result status of an executed operation.
        ///
        access(all) let status: ResultStatus

        /// The optional error of an executed operation.
        ///
        access(all) let error: Error?
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
        access(all)
        fun executeScript(_ script: String, _ arguments: [AnyStruct]): ScriptResult

        /// Creates a signer account by submitting an account creation transaction.
        /// The transaction is paid by the service account.
        /// The returned account can be used to sign and authorize transactions.
        ///
        access(all)
        fun createAccount(): Account

        /// Returns the account for the given address.
        ///
        access(all)
        fun getAccount(_ address: Address): Account

        /// Add a transaction to the current block.
        ///
        access(all)
        fun addTransaction(_ tx: Transaction)

        /// Executes the next transaction in the block, if any.
        /// Returns the result of the transaction, or nil if no transaction was scheduled.
        ///
        access(all)
        fun executeNextTransaction(): TransactionResult?

        /// Commit the current block.
        /// Committing will fail if there are un-executed transactions in the block.
        ///
        access(all)
        fun commitBlock()

        /// Deploys a given contract, and initilizes it with the arguments.
        ///
        access(all)
        fun deployContract(
            name: String,
            path: String,
            arguments: [AnyStruct]
        ): Error?

        /// Returns all the logs from the blockchain, up to the calling point.
        ///
        access(all)
        fun logs(): [String]

        /// Returns the service account of the blockchain. Can be used to sign
        /// transactions with this account.
        ///
        access(all)
        fun serviceAccount(): Account

        /// Returns all events emitted from the blockchain, optionally filtered
        /// by type.
        ///
        access(all)
        fun events(_ type: Type?): [AnyStruct]

        /// Resets the state of the blockchain to the given height.
        ///
        access(all)
        fun reset(to height: UInt64)

        /// Moves the time of the blockchain by the given delta,
        /// which should be passed in the form of seconds.
        ///
        access(all)
        fun moveTime(by delta: Fix64)

        /// Creates a snapshot of the blockchain, at the
        /// current ledger state, with the given name.
        ///
        access(all)
        fun createSnapshot(name: String): Error?

        /// Loads a snapshot of the blockchain, with the
        /// given name, and updates the current ledger
        /// state.
        ///
        access(all)
        fun loadSnapshot(name: String): Error?
    }

    /// Returns a new matcher that negates the test of the given matcher.
    ///
    access(all)
    fun not(_ matcher: Matcher): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return !matcher.test(value)
        })
    }

    /// Returns a new matcher that checks if the given test value is either
    /// a ScriptResult or TransactionResult and the ResultStatus is succeeded.
    /// Returns false in any other case.
    ///
    access(all)
    fun beSucceeded(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return (value as! {Result}).status == ResultStatus.succeeded
        })
    }

    /// Returns a new matcher that checks if the given test value is either
    /// a ScriptResult or TransactionResult and the ResultStatus is failed.
    /// Returns false in any other case.
    ///
    access(all)
    fun beFailed(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return (value as! {Result}).status == ResultStatus.failed
        })
    }

    /// Returns a new matcher that checks if the given test value is nil.
    ///
    access(all)
    fun beNil(): Matcher {
        return Matcher(test: fun (value: AnyStruct): Bool {
            return value == nil
        })
    }

    /// Asserts that the result status of an executed operation, such as
    /// a script or transaction, has failed and contains the given error
    /// message.
    ///
    access(all)
    fun assertError(_ result: {Result}, errorMessage: String) {
        pre {
            result.status == ResultStatus.failed: "no error was found"
        }

        var found = false
        let msg = result.error!.message
        let msgLength = msg.length - errorMessage.length + 1
        var i = 0
        while i < msgLength {
            if msg.slice(from: i, upTo: i + errorMessage.length) == errorMessage {
                found = true
                break
            }
            i = i + 1
        }

        assert(found, message: "the error message did not contain the given sub-string")
    }

    /// Creates a matcher with a test function.
    /// The test function is of type '((T): Bool)',
    /// where 'T' is bound to 'AnyStruct'.
    ///
    access(all)
    native fun newMatcher<T: AnyStruct>(_ test: ((T): Bool)): Test.Matcher {}

    /// Wraps a function call in a closure, and expects it to fail with
    /// an error message that contains the given error message portion.
    ///
    access(all)
    native fun expectFailure(
        _ functionWrapper: ((): Void),
        errorMessageSubstring: String
    ) {}

    /// Expect function tests a value against a matcher
    /// and fails the test if it's not a match.
    ///
    access(all)
    native fun expect<T: AnyStruct>(_ value: T, _ matcher: Test.Matcher) {}

    /// Returns a matcher that succeeds if the tested
    /// value is equal to the given value.
    ///
    access(all)
    native fun equal<T: AnyStruct>(_ value: T): Test.Matcher {}

    /// Fails the test-case if the given values are not equal, and
    /// reports a message which explains how the two values differ.
    ///
    access(all)
    native fun assertEqual(_ expected: AnyStruct, _ actual: AnyStruct) {}

    /// Returns a matcher that succeeds if the tested value is
    /// an array or dictionary and the tested value contains
    /// no elements.
    ///
    access(all)
    native fun beEmpty(): Test.Matcher {}

    /// Returns a matcher that succeeds if the tested value is
    /// an array or dictionary and has the given number of elements.
    ///
    access(all)
    native fun haveElementCount(_ count: Int): Test.Matcher {}

    /// Returns a matcher that succeeds if the tested value is
    /// an array that contains a value that is equal to the given
    /// value, or the tested value is a dictionary that contains
    /// an entry where the key is equal to the given value.
    ///
    access(all)
    native fun contain(_ element: AnyStruct): Test.Matcher {}

    /// Returns a matcher that succeeds if the tested value
    /// is a number and greater than the given number.
    ///
    access(all)
    native fun beGreaterThan(_ value: Number): Test.Matcher {}

    /// Returns a matcher that succeeds if the tested value
    /// is a number and less than the given number.
    ///
    access(all)
    native fun beLessThan(_ value: Number): Test.Matcher {}

    /// Read a local file, and return the content as a string.
    ///
    access(all)
    native fun readFile(_ path: String): String {}

    /// Fails the test-case if the given condition is false,
    /// and reports a message which explains how the condition is false.
    ///
    access(all)
    native fun assert(_ condition: Bool, message: String): Void {}

    /// Fails the test-case with a message.
    ///
    access(all)
    native fun fail(message: String): Void {}

}
