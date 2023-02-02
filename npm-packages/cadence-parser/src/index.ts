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

import { go } from './go'
import WebAssemblyInstantiatedSource = WebAssembly.WebAssemblyInstantiatedSource

declare global {
  namespace NodeJS {
    interface Global {
     [key: string]: any
    }
  }
}

export class CadenceParser {

  private static functionNamePrefix = "CADENCE_PARSER"
  private static loaded = false

  private static functionName(name: string): string {
    return `__${CadenceParser.functionNamePrefix}_${name}__`
  }

  public static async create(binaryLocation: string | BufferSource): Promise<CadenceParser> {
    await this.ensureLoaded(binaryLocation)
    return new CadenceParser()
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

  private constructor() {}

  public parse(code: string): any {
    const result = global[CadenceParser.functionName('parse')](code)
    return JSON.parse(result)
  }

  // setWriteSync installs the writeSync filesystem handler that the Go WebAssembly binary calls
  private static setWriteSync() {
    // For each file descriptor, buffer the written content until reaching a newline

    const outputBuffers = new Map<number, string>()
    const decoder = new TextDecoder("utf-8")

    // Implementing `writeSync` is mainly just for debugging purposes:
    // When the language server writes to a file, e.g. standard output or standard error,
    // then log the output in the console

    global.fs.writeSync = function (fileDescriptor: number, buf: Uint8Array): number {
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
