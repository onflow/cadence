package runtime

type Interface interface {
	// ResolveImport resolves an import of a program.
	ResolveImport(Location) ([]byte, error)
	// GetValue gets a value for the given key in the storage, controlled and owned by the given accounts.
	GetValue(owner, controller, key []byte) (value []byte, err error)
	// SetValue sets a value for the given key in the storage, controlled and owned by the given accounts.
	SetValue(owner, controller, key, value []byte) (err error)
	// CreateAccount creates a new account with the given public keys and code.
	CreateAccount(publicKeys [][]byte) (address Address, err error)
	// AddAccountKey appends a key to an account.
	AddAccountKey(address Address, publicKey []byte) error
	// RemoveAccountKey removes a key from an account by index.
	RemoveAccountKey(address Address, index int) (publicKey []byte, err error)
	// CheckCode checks the validity of the code.
	CheckCode(address Address, code []byte) (err error)
	// UpdateAccountCode updates the code associated with an account.
	UpdateAccountCode(address Address, code []byte, checkPermission bool) (err error)
	// GetSigningAccounts returns the signing accounts.
	GetSigningAccounts() []Address
	// Log logs a string.
	Log(string)
	// EmitEvent is called when an event is emitted by the runtime.
	EmitEvent(Event)
}

type EmptyRuntimeInterface struct{}

func (i *EmptyRuntimeInterface) ResolveImport(location Location) ([]byte, error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) GetValue(controller, owner, key []byte) (value []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) SetValue(controller, owner, key, value []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) CreateAccount(publicKeys [][]byte) (address Address, err error) {
	return Address{}, nil
}

func (i *EmptyRuntimeInterface) AddAccountKey(address Address, publicKey []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) RemoveAccountKey(address Address, index int) (publicKey []byte, err error) {
	return nil, nil
}

func (i *EmptyRuntimeInterface) CheckCode(address Address, code []byte) error {
	return nil
}

func (i *EmptyRuntimeInterface) UpdateAccountCode(address Address, code []byte, checkPermission bool) error {
	return nil
}

func (i *EmptyRuntimeInterface) GetSigningAccounts() []Address {
	return nil
}

func (i *EmptyRuntimeInterface) Log(message string) {}

func (i *EmptyRuntimeInterface) EmitEvent(event Event) {}
