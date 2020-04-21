import {commands, ExtensionContext, Position, Range, window, workspace} from "vscode";
import {Extension, renderExtension} from "./extension";
import {LanguageServerAPI} from "./language-server";
import {createTerminal} from "./terminal";
import {shortAddress, stripAddressPrefix} from "./address";

// Command identifiers for locally handled commands
export const RESTART_SERVER = "cadence.restartServer";
export const START_EMULATOR = "cadence.runEmulator";
export const STOP_EMULATOR = "cadence.stopEmulator";
export const CREATE_ACCOUNT = "cadence.createAccount";
export const SWITCH_ACCOUNT = "cadence.switchActiveAccount";

// Command identifies for commands handled by the Language server
export const CREATE_ACCOUNT_SERVER = "cadence.server.createAccount";
export const CREATE_DEFAULT_ACCOUNTS_SERVER = "cadence.server.createDefaultAccounts";
export const SWITCH_ACCOUNT_SERVER = "cadence.server.switchActiveAccount";

// Registers a command with VS Code so it can be invoked by the user.
function registerCommand(ctx: ExtensionContext, command: string, callback: (...args: any[]) => any) {
    ctx.subscriptions.push(commands.registerCommand(command, callback));
}

// Registers all commands that are handled by the extension (as opposed to
// those handled by the Language Server).
export function registerCommands(ext: Extension) {
    registerCommand(ext.ctx, RESTART_SERVER, restartServer(ext));
    registerCommand(ext.ctx, START_EMULATOR, startEmulator(ext));
    registerCommand(ext.ctx, STOP_EMULATOR, stopEmulator(ext));
    registerCommand(ext.ctx, CREATE_ACCOUNT, createAccount(ext));
    registerCommand(ext.ctx, SWITCH_ACCOUNT, switchActiveAccount(ext));
}

// Restarts the language server, updating the client in the extension object.
const restartServer = (ext: Extension) => async () => {
    await ext.api.client.stop();
    ext.api = new LanguageServerAPI(ext.ctx, ext.config);
};

// Starts the emulator in a terminal window.
const startEmulator = (ext: Extension) => async () => {
    // Start the emulator with the root key we gave to the language server.
    const {serverConfig} = ext.config

    ext.terminal.sendText(
        [
            ext.config.flowCommand,
            `emulator`, `start`, `--init`, `--verbose`,
            `--root-priv-key`, serverConfig.rootPrivateKey,
            `--root-sig-algo`, serverConfig.rootKeySignatureAlgorithm,
            `--root-hash-algo`, serverConfig.rootKeyHashAlgorithm,
        ].join(" ")
    );
    ext.terminal.show();

    // create default accounts after the emulator has started
    // skip root account since it is already created
    setTimeout(async () => {
        try {
            const accounts = await ext.api.createDefaultAccounts(ext.config.numAccounts - 1);
            accounts.forEach(address => ext.config.addAccount(address));
        } catch (err) {
            console.error("Failed to create default accounts", err);
            window.showWarningMessage("Failed to create default accounts");
        }
    }, 3000);
};

// Stops emulator, exits the terminal, and removes all config/db files.
const stopEmulator = (ext: Extension) => async () => {
    ext.terminal.dispose();
    ext.terminal = createTerminal(ext.ctx);

    // Clear accounts and restart language server to ensure account
    // state is in sync.
    ext.config.resetAccounts();
    renderExtension(ext);
    await ext.api.client.stop();
    ext.api = new LanguageServerAPI(ext.ctx, ext.config);
};

// Creates a new account by requesting that the Language Server submit
// a "create account" transaction from the currently active account.
const createAccount = (ext: Extension) => async () => {
    try {
        const addr = await ext.api.createAccount();
        ext.config.addAccount(addr);
    } catch (err) {
        window.showErrorMessage("Failed to create account: " + err);
        return;
    }
};

// Switches the active account to the option selected by the user. The selection
// is propagated to the Language Server.
const switchActiveAccount = (ext: Extension) => async () => {
    // Suffix to indicate which account is active
    const activeSuffix = "(active)";
    // Create the options (mark the active account with an 'active' prefix)
    const accountOptions = Object
        .keys(ext.config.accounts)
        // Mark the active account with a `*` in the dialog
        .map(addr => addr === ext.config.activeAccount ? `${shortAddress(addr)} ${activeSuffix}` : shortAddress(addr));

    window.showQuickPick(accountOptions)
        .then(selected => {
            // `selected` is undefined if the QuickPick is dismissed, and the
            // string value of the selected option otherwise.
            if (selected === undefined) {
                return;
            }
            // If the user selected the active account, remove the `*` prefix
            if (selected.endsWith(activeSuffix)) {
                selected = selected.slice(0, -activeSuffix.length).trim();
            }
            if (!ext.config.accounts[selected]) {
                console.error('Switched to invalid account: ', selected);
                return;
            }

            try {
                ext.api.switchActiveAccount(stripAddressPrefix(selected));
                window.visibleTextEditors.forEach(editor => {
                    if (!editor.document.lineCount) {
                        return;
                    }
                    // NOTE: We add a space to the end of the last line to force
                    // Codelens to refresh.
                    const lineCount = editor.document.lineCount;
                    const lastLine = editor.document.lineAt(lineCount-1);
                    editor.edit(edit => {
                        if (lastLine.isEmptyOrWhitespace) {
                            edit.insert(new Position(lineCount-1, 0), ' ');
                            edit.delete(new Range(lineCount-1, 0, lineCount-1, 1000));
                        } else {
                            edit.insert(new Position(lineCount-1, 1000), '\n');
                        }
                    });
                });
            } catch (err) {
                window.showWarningMessage("Failed to switch active account");
                console.error(err);
                return;
            }
            ext.config.activeAccount = selected;
            window.showInformationMessage(`Switched to account ${selected}`);
            renderExtension(ext);
        });
};
