/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {CadenceLanguageServer, Callbacks} from '@onflow/cadence-language-server'
import {
  createMessageConnection,
  DataCallback,
  Disposable,
  Logger,
  Message,
  MessageReader,
  MessageWriter,
  PartialMessageInfo
} from "vscode-jsonrpc";
import {CloseAction, createConnection, ErrorAction, MonacoLanguageClient} from "monaco-languageclient";

export function createCadenceLanguageClient(callbacks: Callbacks) {
  const logger: Logger = {
    error(message: string) {
      console.error(message)
    },
    warn(message: string) {
      console.warn(message)
    },
    info(message: string) {
      console.info(message)
    },
    log(message: string) {
      console.log(message)
    },
  }

  const writer: MessageWriter = {
    onClose(_: (_: void) => void): Disposable {
      return Disposable.create(() => {
      })
    },
    onError(_: (error: [Error, Message, number]) => void): Disposable {
      return Disposable.create(() => {
      })
    },
    write(msg: Message) {
      if (!callbacks.toServer) {
        return
      }
      callbacks.toServer(null, msg)
    },
    dispose() {
      if (!callbacks.onClientClose) {
        return
      }
      callbacks.onClientClose()
    }
  }

  const reader: MessageReader = {
    onError(_: (error: Error) => void): Disposable {
      return Disposable.create(() => {
      })
    },
    onClose(_: (_: void) => void): Disposable {
      return Disposable.create(() => {
      })
    },
    onPartialMessage(_: (m: PartialMessageInfo) => void): Disposable {
      return Disposable.create(() => {
      })
    },
    listen(dataCallback: DataCallback) {
      callbacks.toClient = (message: Message) => dataCallback(message)
    },
    dispose() {
      if (!callbacks.onClientClose) {
        return
      }
      callbacks.onClientClose()
    }
  }

  const messageConnection = createMessageConnection(reader, writer, logger)

  return new MonacoLanguageClient({
    name: "Cadence Language Client",
    clientOptions: {
      documentSelector: ['cadence'],
      errorHandler: {
        error: () => ErrorAction.Continue,
        closed: () => CloseAction.DoNotRestart
      }
    },
    // Create a language client connection from the JSON-RPC connection on demand
    connectionProvider: {
      get: (errorHandler, closeHandler) => {
        return Promise.resolve(createConnection(messageConnection, errorHandler, closeHandler))
      }
    }
  });
}

export async function createServer(
  binaryLocation: string,
  getAddressCode?: (address: string) => string | undefined
): Promise<CadenceLanguageServer> {

  const callbacks: Callbacks = {
    getAddressCode: getAddressCode,
  }

  const server = CadenceLanguageServer.create(binaryLocation, callbacks);
  const languageClient = createCadenceLanguageClient(callbacks);
  languageClient.start()
  return server
}
