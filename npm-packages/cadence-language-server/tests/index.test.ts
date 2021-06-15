import {CadenceLanguageServer} from "../src"
import * as fs from "fs"
import {Message, RequestMessage} from "vscode-jsonrpc/lib/messages"
import {Callbacks} from "../dist"

test("start", async () => {
  const binary = fs.readFileSync(require.resolve('../dist/cadence-language-server.wasm'))
  const messages: Message[] = []
  let callbacks: Callbacks = {
    toClient(message: Message) {
      messages.push(message)
    },
    toServer() {
      fail("toServer was not initialized")
    }
  }
  await CadenceLanguageServer.create(binary, callbacks)
  const request: RequestMessage = {
    jsonrpc: '2.0',
    id: 1,
    method: 'initialize',
    params: {}
  };

  callbacks.toServer?.(undefined, request)
  expect(messages.length).not.toEqual(0)
})
