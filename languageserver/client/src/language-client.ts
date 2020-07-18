/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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


import {CADENCE_LANGUAGE_ID} from './cadence'
import {Callbacks} from './language-server'
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
      callbacks.toServer(null, msg)
    },
    dispose() {
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
      callbacks.toClient = (message) => dataCallback(message)
    },
    dispose() {
      callbacks.onClientClose()
    }
  }

  const messageConnection = createMessageConnection(reader, writer, logger)

  return new MonacoLanguageClient({
    name: "Cadence Language Client",
    clientOptions: {
      documentSelector: [CADENCE_LANGUAGE_ID],
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
