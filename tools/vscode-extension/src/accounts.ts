import { addAddressPrefix } from "./address";
import { Extension } from "./extension";
export const SERVICE_ADDR: string = "f8d6e0586b0a20c7";

export class AccountsService {
  numAccounts: number;
  // Set of created accounts for which we can submit transactions.
  // Mapping from account address to account object.
  list: Array<Account>;
  // Index of the currently active account.
  activeAccount: number;

  ext: Extension | undefined;

  constructor(numAccounts: number) {
    this.numAccounts = numAccounts;
    this.list = [new Account(0, SERVICE_ADDR)];
    this.activeAccount = 0;
    this.ext = undefined;
  }

  init(ext: Extension) {
    this.ext = ext;
  }

  addAccount(address: string) {
    const index = this.list.length;
    const account = new Account(index, address);
    this.list.push(account);
    this.ext &&
      this.ext.accountsTreeView.accountsTreeViewDataProvider.refresh();
  }

  setActiveAccount(index: number) {
    this.activeAccount = index;
  }

  getActiveAccount(): Account {
    return this.list[this.activeAccount];
  }

  getAccount(index: number): Account | null {
    if (index < 0 || index >= this.list.length) {
      return null;
    }

    return this.list[index];
  }

  // Resets account state
  resetAccounts() {
    this.list = [new Account(0, SERVICE_ADDR)];
    this.activeAccount = 0;
  }
}

export class Account {
  index: number;
  address: string;

  constructor(index: number, address: string) {
    this.index = index;
    this.address = address;
  }

  name(): string {
    return this.index === 0 ? "Service Account" : `Account ${this.index}`;
  }

  fullName(): string {
    return `${this.name()} (${addAddressPrefix(this.address)})`;
  }
}

export function createAccountsService(numAccounts: number) {
  const accountsService = new AccountsService(numAccounts);
  return accountsService;
}
