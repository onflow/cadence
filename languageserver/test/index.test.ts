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

  async function testCode(code: string) {
    return withConnection(async (connection) => {

      const uri = await createTestDocument(connection, code)

      const result = await connection.sendRequest(ExecuteCommandRequest.type, {
        command: "cadence.server.getEntryPointParameters",
        arguments: [uri]
      })

      expect(result).toEqual([{name: 'a', type: 'Int'}])
    })
  }

  test("script", async() =>
    testCode("pub fun main(a: Int) {}"))

  test("transaction", async() =>
    testCode("transaction(a: Int) {}"))
})
