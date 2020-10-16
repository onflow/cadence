import {
  StreamMessageReader,
  StreamMessageWriter,
  ProtocolConnection,
  createProtocolConnection,
  InitializeRequest,
  ExitNotification,
  ExecuteCommandRequest,
  DidOpenTextDocumentNotification,
  TextDocumentItem
} from "vscode-languageserver-protocol"

import { spawn, execSync } from 'child_process'
import * as path from "path"

beforeAll(() => {
  execSync("go build ../cmd/languageserver", {cwd: __dirname})
})

async function withConnection(f: (connection: ProtocolConnection) => Promise<void>): Promise<void> {

  const child = spawn(
    path.resolve(__dirname, './languageserver'),
    ['-enableFlowClient=false']
  )

  child.on('exit', (code) => {
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
