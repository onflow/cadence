import * as monaco from "monaco-editor";
import {MonacoServices} from 'monaco-languageclient';
import {createCadenceLanguageClient} from "./language-client";
import {CADENCE_LANGUAGE_ID, configureCadence} from "./cadence";
import {Callbacks, startCadenceLanguageServer} from "./language-server";


document.addEventListener('DOMContentLoaded', async () => {

    const callbacks: Callbacks = {
      // The actual callback will be set as soon as the language server is initialized
      toServer: null,

      // The actual callback will be set as soon as the language client is initialized
      toClient: null,

      getAddressCode(address: string): string {
            // TODO:
            return `
                pub struct X {}
            `
        }
    }

    await startCadenceLanguageServer(callbacks);

    configureCadence()

    const editorElement = document.getElementById("editor");

    const model = monaco.editor.createModel(
      "import 0x1\n\nstruct S {\n    let n: Int\n}",
      CADENCE_LANGUAGE_ID,
      monaco.Uri.parse('inmemory://main.cdc')
    )

    const editor = monaco.editor.create(
        editorElement,
        {
            theme: 'vs-dark',
            language: CADENCE_LANGUAGE_ID,
            model: model,
            glyphMargin: true,
            minimap: {
                enabled: false
            },
        }
    );

    MonacoServices.install(editor);

  const languageClient = createCadenceLanguageClient(callbacks);
  languageClient.start()
})
