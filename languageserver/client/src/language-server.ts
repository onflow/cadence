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

// The global `Go` is declared by `wasm_exec.js`.
// Instead of improving the that file, we use it as-is,
// because it is maintained by the Go team and

declare var Go: any

// Callbacks defines the functions that the language server calls
// and that need to be implemented by the client.

export interface Callbacks {
  // The function that the language server calls
  // to write a message object to the client.
  // The object is a JSON-RPC request/response object
  toClient(message: Message): void

  // The function that the language server calls
  // to get the code for an imported address, if any
  getAddressCode(address: string): string | undefined

  // The function that the language client calls
  // to write a message object to the server.
  // The object is an JSON-RPC request/response object
  toServer(error: any, message: Message): void

  // The function that the language client can call
  // to notify the server that the client is closing
  onClientClose(): void

  // The function that the language server can call
  // to notify the client that the server is closing
  onServerClose(): void
}

declare global {
  interface Window {
    [index: string]: any;
  }
}

export class CadenceLanguageServer {

  static isLoaded = false

  private static async load() {
    if (this.isLoaded) {
      return
    }

    const wasm = await fetch("./languageserver.wasm")
    const go = new Go()
    const module = await WebAssembly.instantiateStreaming(wasm, go.importObject)

    // For each file descriptor, buffer the written content until reaching a newline

    const outputBuffers = new Map<number, string>()
    const decoder = new TextDecoder("utf-8")

    // Implementing `writeSync` is mainly just for debugging purposes:
    // When the language server writes to a file, e.g. standard output or standard error,
    // then log the output in the console

    window['fs'].writeSync = function (fileDescriptor: number, buf: Uint8Array): number {
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

    go.run(module.instance)

    this.isLoaded = true
  }

  private static functionNamePrefix = "CADENCE_LANGUAGE_SERVER"

  private static functionName(name: string): string {
    return `__${CadenceLanguageServer.functionNamePrefix}_${name}__`
  }

  private functionName(name: string): string {
    return `__${CadenceLanguageServer.functionNamePrefix}_${this.id}_${name}__`
  }

  static async create(callbacks: Callbacks): Promise<CadenceLanguageServer> {

    await this.load()

    return new CadenceLanguageServer(callbacks)
  }

  public readonly id: number
  private isClientClosed: boolean

  private constructor(callbacks: Callbacks) {

    // The language server, written in Go and compiled to WebAssembly, interacts with this JS environment
    // by calling global functions. There does not seem to be support yet to directly import functions
    // from the JS environment into the WebAssembly environment

    this.id = window[CadenceLanguageServer.functionName('start')]()

    window[this.functionName('toClient')] = (message: string): void => {
      callbacks.toClient(JSON.parse(message))
    }

    window[this.functionName('getAddressCode')] = (address: string): string | undefined => {
      if (!callbacks.getAddressCode) {
        return undefined
      }

      return callbacks.getAddressCode(address)
    }

    window[this.functionName('onServerClose')] = (): void => {
      if (!callbacks.onServerClose) {
        return
      }
      callbacks.onServerClose()
    }

    callbacks.toServer = (error: any, message: any) => {
      window[this.functionName('toServer')](error, JSON.stringify(message))
    }

    callbacks.onClientClose = () => {
      if (this.isClientClosed) {
        return
      }
      this.isClientClosed = true
      window[this.functionName('onClientClose')]()
    }
  }
}
