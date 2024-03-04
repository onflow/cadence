import { createRoot } from 'react-dom/client'
import React, { createElement, ReactNode } from "react"
import CompositeValue from "./composite.tsx"
import DictionaryValue from "./dictionary.tsx"
import PrimitiveValue from "./primitive.tsx"
import { Value } from "./value.ts"
import ArrayValue from "./array.tsx"
import FallbackValue from "./fallback.tsx"

function request(url: string, method: string = 'GET', body?: BodyInit) {
    return fetch(url, {
        method,
        headers: {
            'Content-Type': 'application/json'
        },
        body
    })
}

class AccountsView {
    private readonly storageMapKeysView: StorageMapKeysView
    private readonly selectElement: HTMLSelectElement

    constructor(storageMapKeysView: StorageMapKeysView) {
        this.storageMapKeysView = storageMapKeysView
        this.selectElement = document.querySelector('#accounts select')!

        this.addKeyboardEventListeners()

        this.load().then(addresses => {
            this.selectElement.addEventListener('change', this.onChange.bind(this))

            if (addresses.length) {
                this.selectElement.value = addresses[0]
                this.onChange()
            }
        })
    }

    private onChange() {
        this.storageMapKeysView.address = this.selectElement.value
    }

    private async load(): Promise<string[]> {
        const response = await request('/accounts')
        const addresses = await response.json()
        for (const address of addresses) {
            const option = document.createElement('option')
            option.value = address
            option.textContent = address
            this.selectElement.appendChild(option)
        }
        return addresses
    }

    private addKeyboardEventListeners() {
        this.selectElement.addEventListener('keydown', event => {
            if (event.key === 'ArrowRight') {
                event.preventDefault()
                this.storageMapKeysView.focus()
            }
        })
    }

    focus() {
        this.selectElement.focus()
    }
}

class StorageMapsView {
    private readonly storageMapKeysView: StorageMapKeysView
    private readonly selectElement: HTMLSelectElement

    constructor(storageMapKeysView: StorageMapKeysView) {
        this.storageMapKeysView = storageMapKeysView
        this.selectElement = document.querySelector('#storage-maps select')!

        this.load().then(() => {
            this.selectElement.addEventListener('change', this.onChange.bind(this))
            this.selectElement.value = 'storage'
            this.onChange()
        })
    }

    private onChange() {
        this.storageMapKeysView.domain = this.selectElement.value
    }

    private async load() {
        const response = await request('/known_storage_maps')
        const domains = await response.json()
        for (const domain of domains) {
            const option = document.createElement('option')
            option.value = domain
            option.textContent = domain
            this.selectElement.appendChild(option)
        }
    }
}

class KeyPath {
    constructor(
        public address: string,
        public domain: string,
        public key: string,
        public nested: any[]
    ) {}

    with(nested: any): KeyPath {
        const newNested = this.nested.slice()
        newNested.push(nested)
        return new KeyPath(
            this.address,
            this.domain,
            this.key,
            newNested
        )
    }
}

interface FocusableView {
    focus(): void
}

class StorageMapKeysView implements FocusableView {
    private readonly headingElement: HTMLHeadingElement
    private readonly selectElement: HTMLSelectElement
    private readonly valuesElement: HTMLDivElement
    private _address: string | null = null
    private _domain: string | null = null
    accountsView: AccountsView | null = null
    private valueView: ValueView | null = null

    constructor() {
        this.headingElement = document.querySelector('#storage-map-keys h2')!
        this.selectElement = document.querySelector('#storage-map-keys select')!
        this.selectElement.addEventListener('change', this.onChange.bind(this))
        this.valuesElement = document.querySelector('#values')!

        this.addKeyboardEventListeners()
    }

    private async onChange() {
        const key = this.selectElement.value
        this.valuesElement.innerHTML = ""

        const address = this.address
        const domain = this.domain
        if (!address || !domain) {
            return
        }

        this.valueView = new ValueView(
            new KeyPath(address, domain, key, []),
            this.valuesElement,
            this
        )
    }

    get address(): string | null {
        return this._address
    }

    set address(address: string) {
        this._address = address
        this.update()
    }

    get domain(): string | null {
        return this._domain
    }

    set domain(domain: string) {
        this._domain = domain
        this.update()
    }

    private async update() {
        this.valuesElement.innerHTML = ""

        const address = this.address
        const domain = this.domain
        if (!address || !domain) {
            return
        }

        this.headingElement.textContent = `${address}/${domain}`

        const response = await request(`/accounts/${address}/${domain}`)
        const keys = await response.json()
        this.selectElement.innerHTML = ""
        for (const key of keys) {
            const option = document.createElement('option')
            option.value = key
            option.textContent = key
            this.selectElement.appendChild(option)
        }
    }

    focus() {
        const select = this.selectElement
        const options = select.options

        if (!select.value && options.length) {
            select.value = options[0].value
            this.onChange()
        }

        select.focus()
    }

    private addKeyboardEventListeners() {
        this.selectElement.addEventListener('keydown', event => {
            switch (event.key) {
                case 'ArrowLeft':
                    event.preventDefault()
                    this.accountsView?.focus()
                    break
                case 'ArrowRight':
                    event.preventDefault()
                    this.valueView?.focus()
                    break
            }
        })
    }
}

class ValueView implements FocusableView {
    private readonly keyPath: KeyPath
    private readonly valueElement: HTMLDivElement
    private readonly parentElement: HTMLElement
    private nextValueView: ValueView | null = null
    private previousElement: FocusableView

    constructor(keyPath: KeyPath, parentElement: HTMLElement, previousElement: FocusableView) {
        this.keyPath = keyPath
        this.previousElement = previousElement

        this.valueElement = document.createElement('div')
        this.valueElement.classList.add('value')
        parentElement.appendChild(this.valueElement)

        this.parentElement = parentElement

        this.load()
    }

    async load() {
        const {address, domain, key, nested} = this.keyPath
        const response = await request(
            `/accounts/${address}/${domain}/${key}`,
            'POST',
            JSON.stringify(nested)
        )
        const value: Value = await response.json()

        let valueComponent: ReactNode | null
        switch (value.kind) {
            case "composite":
                valueComponent = createElement(
                    CompositeValue,
                    {
                        value,
                        onChange: this.onChange.bind(this),
                        onKeyDown: this.onKeyDown.bind(this)
                    }
                )
                break

            case "dictionary":
                valueComponent = createElement(
                    DictionaryValue,
                    {
                        value,
                        onChange: this.onChange.bind(this),
                        onKeyDown: this.onKeyDown.bind(this)
                    }
                )
                break

            case "array":
                valueComponent = createElement(
                    ArrayValue,
                    {
                        value,
                        onChange: this.onChange.bind(this),
                        onKeyDown: this.onKeyDown.bind(this)
                    }
                )
                break

            case "primitive":
                valueComponent = createElement(
                    PrimitiveValue,
                    {
                        value
                    }
                )
                break

            case "fallback":
                valueComponent = createElement(
                    FallbackValue,
                    {
                        value
                    }
                )
                break
        }

        if (valueComponent) {
            createRoot(this.valueElement).render(valueComponent)
        } else {
            const contentsElement = document.createElement('pre')
            contentsElement.innerText = JSON.stringify(value, null, '  ')
            this.valueElement.appendChild(contentsElement)
        }
    }

    private onChange(nested: any) {
        // clear trailing value views
        while (true) {
            const lastChild = this.parentElement.lastChild
            if (lastChild === this.valueElement) {
                break
            }
            lastChild?.remove()
        }

        // append new value view
        this.nextValueView = new ValueView(
            this.keyPath.with(nested),
            this.parentElement,
            this
        )
    }

    private onKeyDown(event: React.KeyboardEvent) {
        switch (event.key) {
            case 'ArrowLeft':
                event.preventDefault()
                this.previousElement.focus()
                break
            case 'ArrowRight':
                event.preventDefault()
                this.nextValueView?.focus()
                break
        }
    }

    focus() {
        const select = this.valueElement.querySelector("select")
        if (!select) {
            return
        }

        const options = select.options

        if (!select.value && options.length) {
            select.value = options[0].value
        }

        select.focus()
    }
}

document.addEventListener("DOMContentLoaded", function () {
    const storageMapKeysView = new StorageMapKeysView()
    const accountsView = new AccountsView(storageMapKeysView)
    storageMapKeysView.accountsView = accountsView
    new StorageMapsView(storageMapKeysView)
    accountsView.focus()
})