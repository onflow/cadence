import {
  createProtocolConnection,
  DidOpenTextDocumentNotification,
  ExecuteCommandRequest,
  ExitNotification,
  InitializeRequest,
  ProtocolConnection,
  StreamMessageReader,
  StreamMessageWriter,
  TextDocumentItem,
  PublishDiagnosticsNotification,
  PublishDiagnosticsParams, DidChangeTextDocumentNotification
} from "vscode-languageserver-protocol"

import {execSync, spawn} from 'child_process'
import * as path from "path"
import * as fs from "fs";
import {VersionedTextDocumentIdentifier} from "vscode-languageserver-types";
import {TextDocumentContentChangeEvent} from "vscode-languageserver-protocol/lib/protocol";

beforeAll(() => {
  execSync("go build ../cmd/languageserver", {cwd: __dirname})
})

async function withConnection(f: (connection: ProtocolConnection) => Promise<void>): Promise<void> {

  const child = spawn(
    path.resolve(__dirname, './languageserver'),
    ['-enableFlowClient=false']
  )

  let stderr = ""
  child.stderr.setEncoding('utf8')
  child.stderr.on('data', (data) => {
    stderr += data
  });

  child.on('exit', (code) => {
    if (code !== 0) {
      console.error(stderr)
    }
    expect(code).toBe(0)
  })

  const connection = createProtocolConnection(
    new StreamMessageReader(child.stdout),
    new StreamMessageWriter(child.stdin),
    null
  );

  connection.listen()

  await connection.sendRequest(InitializeRequest.type,
    {
      capabilities: {},
      processId: process.pid,
      rootUri: '/',
      workspaceFolders: null,
    }
  )

  await f(connection)

  await connection.sendNotification(ExitNotification.type)
}

async function createTestDocument(connection: ProtocolConnection, code: string): Promise<string> {
  const uri = "file:///test.cdc"

  await connection.sendNotification(DidOpenTextDocumentNotification.type, {
    textDocument: TextDocumentItem.create(
      uri,
      "cadence",
      1,
      code,
    )
  })

  return uri
}

describe("getEntryPointParameters command", () => {

  async function testCode(code: string, expectedParameters: object[]) {
    return withConnection(async (connection) => {

      const uri = await createTestDocument(connection, code)

      const result = await connection.sendRequest(ExecuteCommandRequest.type, {
        command: "cadence.server.getEntryPointParameters",
        arguments: [uri]
      })

      expect(result).toEqual(expectedParameters)
    })
  }

  test("script", async() =>
    testCode(
        `pub fun main(a: Int) {}`,
        [{name: 'a', type: 'Int'}]
    )
  )

  test("transaction", async() =>
    testCode(
        `transaction(a: Int) {}`,
        [{name: 'a', type: 'Int'}]
    )
  )
})

describe("getContractInitializerParameters command", () => {

  async function testCode(code: string, expectedParameters: object[]) {
    return withConnection(async (connection) => {

      const uri = await createTestDocument(connection, code)

      const result = await connection.sendRequest(ExecuteCommandRequest.type, {
        command: "cadence.server.getContractInitializerParameters",
        arguments: [uri]
      })

      expect(result).toEqual(expectedParameters)
    })
  }

  test("no contract", async() =>
      testCode(
          ``,
          []
      )
  )

  test("one contract, no parameters", async() =>
      testCode(
          `
          pub contract C {
              init() {}
          }
          `,
          [],
      )
  )

  test("one contract, one parameter", async() =>
      testCode(
          `
          pub contract C {
              init(a: Int) {}
          }
          `,
          [{name: 'a', type: 'Int'}],
      )
  )

  test("many contracts", async() =>
      testCode(
          `
          pub contract C1 {
              init(a: Int) {}
          }

          pub contract C2 {
              init(b: Int) {}
          }
          `,
          []
      )
  )
})

describe("parseEntryPointArguments command", () => {

  async function testCode(code: string) {
    return withConnection(async (connection) => {

      const uri = await createTestDocument(connection, code)

      const result = await connection.sendRequest(ExecuteCommandRequest.type, {
        command: "cadence.server.parseEntryPointArguments",
        arguments: [uri, ['0x42']]
      })

      expect(result).toEqual([{value: '0x0000000000000042', type: 'Address'}])
    })
  }

  test("script", async() =>
    testCode("pub fun main(a: Address) {}"))

  test("transaction", async() =>
    testCode("transaction(a: Address) {}"))
})

describe("diagnostics", () => {

  async function testCode(code: string) {
    return withConnection(async (connection) => {

      const notificationPromise = new Promise<PublishDiagnosticsParams>((resolve) => {
        connection.onNotification(PublishDiagnosticsNotification.type, resolve)
      })

      const uri = await createTestDocument(connection, code)

      const notification = await notificationPromise

      expect(notification.uri).toEqual(uri)
      expect(notification.diagnostics).toHaveLength(1)
      expect(notification.diagnostics[0].message).toEqual("cannot find variable in this scope: `X`. not found in this scope")
    })
  }

  test("script", async() =>
    testCode(
      `pub fun main() { X }`,
    )
  )

  test("transaction", async() =>
    testCode(
      `transaction() { execute { X } }`,
    )
  )

  type TestDoc = {
    name: string
    code: string
  }

  type DocNotification = {
    name: string
    notification: Promise<PublishDiagnosticsParams>
  }

  const fooContractCode = fs.readFileSync('./foo.cdc', 'utf8')

  async function testImports(docs: TestDoc[]): Promise<DocNotification[]> {
    return new Promise<DocNotification[]>(resolve => {

      withConnection(async (connection) => {
        let docsNotifications: DocNotification[] = []

        for (let doc of docs) {
          const notification = new Promise<PublishDiagnosticsParams>((resolve) => {
            connection.onNotification(PublishDiagnosticsNotification.type, (notification) => {
              if (notification.uri == `file://${doc.name}.cdc`) {
                resolve(notification)
              }
            })
          })
          docsNotifications.push({
            name: doc.name,
            notification: notification
          })

          await connection.sendNotification(DidOpenTextDocumentNotification.type, {
            textDocument: TextDocumentItem.create(`file://${doc.name}.cdc`, "cadence", 1, doc.code)
          })
        }

        resolve(docsNotifications)
      })

    })
  }

  async function getDiagnostics(connection: ProtocolConnection, numberOfDiagnostics: number): Promise<PublishDiagnosticsParams[]> {
    let diagnostics: PublishDiagnosticsParams[] = []

    return new Promise<PublishDiagnosticsParams[]>((resolve) => {
      connection.onNotification(
          PublishDiagnosticsNotification.type,
          notification => {
            diagnostics.push(notification)
            if (diagnostics.length == numberOfDiagnostics) {
              resolve(diagnostics)
            }
          }
      )
    })
  }

  test("script with import", async() => {
    const contractName = "foo"
    const scriptName = "script"
    const scriptCode = `
      import Foo from "./foo.cdc"
      pub fun main() { log(Foo.bar) }
    `

    let docNotifications = await testImports([
      { name: contractName, code: fooContractCode },
      { name: scriptName, code: scriptCode }
    ])

    let script = await docNotifications.find(n => n.name == scriptName).notification
    expect(script.uri).toEqual(`file://${scriptName}.cdc`)
    expect(script.diagnostics).toHaveLength(0)
  })

  test("script import failure", async() => {
    const contractName = "foo"
    const scriptName = "script"
    const scriptCode = `
      import Foo from "./foo.cdc"
      pub fun main() { log(Foo.zoo) }
    `

    let docNotifications = await testImports([
      { name: contractName, code: fooContractCode },
      { name: scriptName, code: scriptCode }
    ])

    let script = await docNotifications.find(n => n.name == scriptName).notification
    expect(script.uri).toEqual(`file://${scriptName}.cdc`)
    expect(script.diagnostics).toHaveLength(1)
    expect(script.diagnostics[0].message).toEqual("value of type `Foo` has no member `zoo`. unknown member")
  })

  test("changes to imported contracts are visible", async() => {

    const contractAPath = "./fixtures/contractA.cdc"
    const contractA = `
      pub contract contractA {
         pub let alpha: String
         init() {
           self.alpha = "hello"
         }
      }
    `
    const contractA1 = `
      pub contract contractA {
         pub let alpha: String
         pub let bravo: String
         init() {
           self.alpha = "hello"
           self.bravo = "world"
         }
      }
    `

    const contractBPath = "./fixtures/contractB.cdc"
    const contractB = `
      import contractA from "./contractA.cdc"
      pub contract contractB {
         init() {
            log(contractA.alpha)
            log(contractA.bravo)
         }
      }
    `

    let diagnosticsPromise
    await withConnection(async (connection) => {
      diagnosticsPromise = getDiagnostics(connection, 3)

      fs.writeFileSync(contractAPath, contractA)

      connection.sendNotification(DidOpenTextDocumentNotification.type, {
        textDocument: TextDocumentItem.create(`file://${contractAPath}`, "cadence", 1, contractA)
      })

      connection.sendNotification(DidOpenTextDocumentNotification.type, {
        textDocument: TextDocumentItem.create(`file://${contractBPath}`, "cadence", 1, contractB)
      })

      await new Promise((r) => setTimeout(r, 200))
      fs.writeFileSync(contractAPath, contractA1)

      connection.sendNotification(DidChangeTextDocumentNotification.type, {
        textDocument: VersionedTextDocumentIdentifier.create(`file://${contractAPath}`, 1),
        contentChanges: [ {
          text: contractA1
        } ],
      })

      connection.sendNotification(DidOpenTextDocumentNotification.type, {
        textDocument: TextDocumentItem.create(`file://${contractBPath}`, "cadence", 1, contractB)
      })

      return diagnosticsPromise
    })

    let diagnostics = await diagnosticsPromise
    expect(diagnostics[0].uri).toEqual(`file://${contractAPath}`)
    expect(diagnostics[1].uri).toEqual(`file://${contractBPath}`)
    expect(diagnostics[1].diagnostics.length).toEqual(1)
    expect(diagnostics[1].diagnostics[0].message).toEqual("value of type `contractA` has no member `bravo`. unknown member")
    expect(diagnostics[2].uri).toEqual(`file://${contractAPath}`)
    expect(diagnostics[2].diagnostics.length).toEqual(0) // no diagnostic problems after member has been actually added
  })

  test("referencing same type in multiple places gets correctly resolved", async() => {

  })

})
