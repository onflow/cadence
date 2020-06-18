import { ExtensionContext, window, TreeItem, TreeView } from "vscode";
import { FlowAccountDetailTreeDataProvider } from "./accounts-data-provider";
import { Config } from "../config";

export interface AccountsTreeView {
  accountsTreeView: TreeView<TreeItem>;
  accountsTreeViewDataProvider: FlowAccountDetailTreeDataProvider;
}

export function createAccountsTreeView(
  ctx: ExtensionContext,
  config: Config
): AccountsTreeView {
  const accountsTreeViewDataProvider = new FlowAccountDetailTreeDataProvider(
    ctx,
    config
  );
  const accountsTreeView = window.createTreeView("flowAccounts", {
    treeDataProvider: accountsTreeViewDataProvider,
  });

  return { accountsTreeView, accountsTreeViewDataProvider };
}
