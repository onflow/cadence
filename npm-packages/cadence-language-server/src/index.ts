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

import {Message} from "vscode-jsonrpc/lib/messages";
import {go} from './go.js'
import WebAssemblyInstantiatedSource = WebAssembly.WebAssemblyInstantiatedSource

// Callbacks defines the functions that the language server calls
// and that need to be implemented by the client.

export interface Callbacks {
  // The function that the language server calls
  // to write a message object to the client.
  // The object is a JSON-RPC request/response object
  toClient?(message: Message): void

  // The function that the language server calls
  // to get the code for an imported address, if any
  getAddressCode?(address: string): string | undefined

  // The function that the language client calls
  // to write a message object to the server.
  // The object is an JSON-RPC request/response object
  toServer?(error: any, message: Message): void

  // The function that the language client can call
  // to notify the server that the client is closing
  onClientClose?(): void

  // The function that the language server can call
  // to notify the client that the server is closing
  onServerClose?(): void
}

const env: {
  [key: string]: any
} = typeof global !== 'undefined' ? global : window

export class CadenceLanguageServer {

  private static functionNamePrefix = "CADENCE_LANGUAGE_SERVER"
  private static loaded = false

  private static functionName(name: string): string {
    return `__${CadenceLanguageServer.functionNamePrefix}_${name}__`
  }

  private functionName(name: string): string {
    return `__${CadenceLanguageServer.functionNamePrefix}_${this.id}_${name}__`
  }

  public readonly callbacks: Callbacks
  public readonly id: number
  private isClientClosed: boolean = false

  public static async create(binaryLocation: string | BufferSource, callbacks: Callbacks): Promise<CadenceLanguageServer> {
    await this.ensureLoaded(binaryLocation)
    return new CadenceLanguageServer(callbacks)
  }

  private static async ensureLoaded(urlOrBinary: string | BufferSource) {
    if (this.loaded) {
      return
    }

    this.setWriteSync()

    await this.load(urlOrBinary)
    this.loaded = true
  }

  private static async load(urlOrBinary: string | BufferSource): Promise<void> {
    let instantiatedSource: WebAssemblyInstantiatedSource
    if (typeof urlOrBinary === 'string') {
      const binaryRequest = fetch(urlOrBinary)
      instantiatedSource = (await WebAssembly.instantiateStreaming(binaryRequest, go.importObject))
    } else {
      instantiatedSource = await WebAssembly.instantiate(urlOrBinary, go.importObject);
    }

    // NOTE: don't await the promise, just ignore it, as it is only resolved when the program exists
    go.run(instantiatedSource.instance).then(() => {})
  }

  private constructor(callbacks: Callbacks) {
    this.callbacks = callbacks

    // The language server, written in Go and compiled to WebAssembly, interacts with this JS environment
    // by calling global functions. There does not seem to be support yet to directly import functions
    // from the JS environment into the WebAssembly environment

    this.id = env[CadenceLanguageServer.functionName('start')]()

    env[this.functionName('toClient')] = (message: string): void => {
      if (!callbacks.toClient) {
        return
      }
      callbacks.toClient(JSON.parse(message))
    }

    env[this.functionName('getAddressCode')] = (address: string): string | undefined => {
      if (!callbacks.getAddressCode) {
        return undefined
      }

      return callbacks.getAddressCode(address)
    }

    env[this.functionName('onServerClose')] = (): void => {
      if (!callbacks.onServerClose) {
        return
      }
      callbacks.onServerClose()
    }

    callbacks.toServer = (error: any, message: any) => {
      env[this.functionName('toServer')](error, JSON.stringify(message))
    }

    callbacks.onClientClose = () => {
      if (this.isClientClosed) {
        return
      }
      this.isClientClosed = true
      env[this.functionName('onClientClose')]()
    }
  }

  close() {
    const { onClientClose } = this.callbacks;
    if (!onClientClose) {
      return
    }
    onClientClose()
  }

  // setWriteSync installs the writeSync filesystem handler that the Go WebAssembly binary calls
  private static setWriteSync() {
    // For each file descriptor, buffer the written content until reaching a newline

    const outputBuffers = new Map<number, string>()
    const decoder = new TextDecoder("utf-8")

    // Implementing `writeSync` is mainly just for debugging purposes:
    // When the language server writes to a file, e.g. standard output or standard error,
    // then log the output in the console

    env['fs'].writeSync = function (fileDescriptor: number, buf: Uint8Array): number {
      // Get the currently buffered output for the given file descriptor,
      // or initialize it, if there is no buffered output yet.

      let outputBuffer = outputBuffers.get(fileDescriptor)
      if (!outputBuffer) {
        outputBuffer = ""
      }

      // Decode the written data as UTF-8
      outputBuffer += decoder.decode(buf)

      // If the buffered output contains a newline,
      // log the contents up to the newline to the console

      const nl = outputBuffer.lastIndexOf("\n")
      if (nl != -1) {
        const lines = outputBuffer.substr(0, nl + 1)
        console.debug(`(FD ${fileDescriptor}):`, lines)
        // keep the remainder
        outputBuffer = outputBuffer.substr(nl + 1)
      }
      outputBuffers.set(fileDescriptor, outputBuffer)

      return buf.length
    }
  }
}
