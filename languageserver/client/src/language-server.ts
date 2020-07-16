// The global `Go` is declared by `wasm_exec.js`.
// Instead of improving the that file, we use it as-is,
// because it is maintained by the Go team and

declare var Go

// Callbacks defines the functions that the language server calls
// and that need to be implemented by the client.

export interface Callbacks {
  // The callback that the language server calls
  // to write a message object to the client.
  // The object is a deserialized JSON-RPC request/response
  toClient(any): void

  // The callback that the language server calls
  // to get the code for an imported address, if any
  getAddressCode(string): string

  toServer(error: any, message: any): void
}

export async function startCadenceLanguageServer(callbacks: Callbacks) {
  const wasm = await fetch("./languageserver.wasm")
  const go = new Go()
  const result = await WebAssembly.instantiateStreaming(wasm, go.importObject)

  // The Go language server (WebAssembly environment) interacts with this JS environment
  // by calling global functions. There does not seem to be support yet
  // to directly import functions from the JS environment into the WebAssembly environment

  window['__CADENCE_LANGUAGE_SERVER_toClient__'] = (message: string) => {
    callbacks.toClient(JSON.parse(message))
  }

  window['__CADENCE_LANGUAGE_SERVER_getAddressCode__'] = (address: string): string => {
    return callbacks.getAddressCode(address)
  }

  window['__CADENCE_LANGUAGE_SERVER_close__'] = function() {
    // TODO:
  }

  callbacks.toServer = (error: any, message: any) => {
    window['__CADENCE_LANGUAGE_SERVER_toServer__'](error, JSON.stringify(message))
  }

  // For each file descriptor, buffer the written content until reaching a newline

  const outputBuffers = new Map<number, string>()
  const decoder = new TextDecoder("utf-8")

  // Implementing `writeSync` is mainly just for debugging purposes:
  // When the language server writes to a file, e.g. standard output or standard error,
  // then log the output in the console

  window['fs'].writeSync = function(fileDescriptor: number, buf: Uint8Array): number {
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

  go.run(result.instance)
}
