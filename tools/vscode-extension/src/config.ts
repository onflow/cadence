import { commands, window, workspace } from "vscode";
import { AccountsService, createAccountsService } from "./accounts";

const CONFIG_FLOW_COMMAND = "flowCommand";
const CONFIG_SERVICE_PRIVATE_KEY = "servicePrivateKey";
const CONFIG_SERVICE_KEY_SIGNATURE_ALGORITHM = "serviceKeySignatureAlgorithm";
const CONFIG_SERVICE_KEY_HASH_ALGORITHM = "serviceKeyHashAlgorithm";
const CONFIG_EMULATOR_ADDRESS = "emulatorAddress";
const CONFIG_NUM_ACCOUNTS = "numAccounts";

// An account that can be used to submit transactions.

// The subset of extension configuration used by the language server.
type ServerConfig = {
  servicePrivateKey: string;
  serviceKeySignatureAlgorithm: string;
  serviceKeyHashAlgorithm: string;
  emulatorAddress: string;
};

// The configuration used by the extension.
export class Config {
  // The name of the flow CLI executable
  flowCommand: string;
  serverConfig: ServerConfig;
  numAccounts: number;
  accounts: AccountsService;

  constructor(
    flowCommand: string,
    numAccounts: number,
    serverConfig: ServerConfig
  ) {
    this.flowCommand = flowCommand;
    this.serverConfig = serverConfig;
    this.numAccounts = numAccounts;
    this.accounts = createAccountsService(numAccounts);
  }
}

// Retrieves config from the workspace.
export function getConfig(): Config {
  const cadenceConfig = workspace.getConfiguration("cadence");

  const flowCommand: string | undefined = cadenceConfig.get(
    CONFIG_FLOW_COMMAND
  );
  if (!flowCommand) {
    throw new Error(`Missing ${CONFIG_FLOW_COMMAND} config`);
  }

  const servicePrivateKey: string | undefined = cadenceConfig.get(
    CONFIG_SERVICE_PRIVATE_KEY
  );
  if (!servicePrivateKey) {
    throw new Error(`Missing ${CONFIG_SERVICE_PRIVATE_KEY} config`);
  }

  const serviceKeySignatureAlgorithm: string | undefined = cadenceConfig.get(
    CONFIG_SERVICE_KEY_SIGNATURE_ALGORITHM
  );
  if (!serviceKeySignatureAlgorithm) {
    throw new Error(`Missing ${CONFIG_SERVICE_KEY_SIGNATURE_ALGORITHM} config`);
  }

  const serviceKeyHashAlgorithm: string | undefined = cadenceConfig.get(
    CONFIG_SERVICE_KEY_HASH_ALGORITHM
  );
  if (!serviceKeyHashAlgorithm) {
    throw new Error(`Missing ${CONFIG_SERVICE_KEY_HASH_ALGORITHM} config`);
  }

  const emulatorAddress: string | undefined = cadenceConfig.get(
    CONFIG_EMULATOR_ADDRESS
  );
  if (!emulatorAddress) {
    throw new Error(`Missing ${CONFIG_EMULATOR_ADDRESS} config`);
  }

  const numAccounts: number | undefined = cadenceConfig.get(
    CONFIG_NUM_ACCOUNTS
  );
  if (!numAccounts) {
    throw new Error(`Missing ${CONFIG_NUM_ACCOUNTS} config`);
  }

  const serverConfig: ServerConfig = {
    servicePrivateKey,
    serviceKeySignatureAlgorithm,
    serviceKeyHashAlgorithm,
    emulatorAddress,
  };

  return new Config(flowCommand, numAccounts, serverConfig);
}

// Adds an event handler that prompts the user to reload whenever the config
// changes.
export function handleConfigChanges() {
  workspace.onDidChangeConfiguration((e) => {
    // TODO: do something smarter for account/emulator config (re-send to server)
    const promptRestartKeys = [
      "languageServerPath",
      "accountKey",
      "accountAddress",
      "emulatorAddress",
    ];
    const shouldPromptRestart = promptRestartKeys.some((key) =>
      e.affectsConfiguration(`cadence.${key}`)
    );
    if (shouldPromptRestart) {
      window
        .showInformationMessage(
          "Server launch configuration change detected. Reload the window for changes to take effect",
          "Reload Window",
          "Not now"
        )
        .then((choice) => {
          if (choice === "Reload Window") {
            commands.executeCommand("workbench.action.reloadWindow");
          }
        });
    }
  });
}
