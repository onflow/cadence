import {
  TreeItem,
  TreeItemCollapsibleState,
  TreeDataProvider,
  ProviderResult,
  ExtensionContext,
  Event,
  EventEmitter,
} from "vscode";

import { Config } from "../config";

export class FlowAccountTreeItem extends TreeItem {
  constructor(
    public readonly label: string,
    public readonly collapsibleState: TreeItemCollapsibleState,
    public readonly accountAddress: string,
    public readonly isActive: string
  ) {
    super(label, collapsibleState);
  }

  get tooltip(): string {
    return `${this.label}`;
  }

  iconPath = {
    light: "",
    dark: "",
  };
}

export class FlowAccountDetailTreeItem extends TreeItem {
  constructor(
    public readonly label: string,
    public readonly collapsibleState: TreeItemCollapsibleState,
    public readonly isActive: boolean
  ) {
    super(label, collapsibleState);
  }

  get tooltip(): string {
    return `${this.label}`;
  }

  get description(): string {
    return `${this.label} ${this.isActive ? "(active)" : ""}`;
  }

  iconPath = {
    light: "",
    dark: "",
  };
}

export class FlowAccountDetailTreeDataProvider
  implements TreeDataProvider<TreeItem> {
  config: Config;
  constructor(ctx: ExtensionContext, config: Config) {
    this.config = config;
  }

  private getUpdatedAccountData(): FlowAccountTreeItem[] {
    return this.config.accounts.list.map((acct) => {
      const isServiceAccount = acct.index === 0;
      const label = isServiceAccount
        ? `${acct.address} (Service account)`
        : acct.address;
      return new FlowAccountTreeItem(label, 0, acct.address, "active");
    });
  }

  getTreeItem(element: TreeItem) {
    return element;
  }

  getChildren(element?: TreeItem): ProviderResult<TreeItem[]> {
    if (element === undefined) return this.getUpdatedAccountData();
    return undefined;
  }

  private _onDidChangeTreeData: EventEmitter<
    FlowAccountTreeItem | undefined
  > = new EventEmitter<FlowAccountTreeItem | undefined>();

  readonly onDidChangeTreeData: Event<FlowAccountTreeItem | undefined> = this
    ._onDidChangeTreeData.event;

  refresh(): void {
    this._onDidChangeTreeData.fire();
  }
}
