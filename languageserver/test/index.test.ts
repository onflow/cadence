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

import { spawn, exec } from 'child_process'

beforeAll(() => {
  exec("go build ../cmd/languageserver")
})

async function withConnection(f: (connection: ProtocolConnection) => Promise<void>): Promise<void> {

  const child = spawn('./languageserver', ['-enableFlowClient=false'])

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
