import { createRoot } from 'react-dom/client'
import {createElement, ReactNode} from "react"
import CompositeValue from "./composite.tsx";
import {Value} from "./value.ts";
import DictionaryValue from "./dictionary.tsx";

function request(url: string) {
    return fetch(url, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json'
        }
    })
}

class AccountsView {
    private storageMapKeysView: StorageMapKeysView
    private selectElement: HTMLSelectElement

    constructor(storageMapKeysView: StorageMapKeysView) {
        this.storageMapKeysView = storageMapKeysView
        this.selectElement = document.querySelector('#accounts select')!
        this.load().then(() => {
            this.selectElement.addEventListener('change', this.onChange.bind(this))
        })
    }

    onChange() {
        this.storageMapKeysView.address = this.selectElement.value
    }

    async load() {
        const response = await request('/accounts')
        const addresses = await response.json()
        for (const address of addresses) {
            const option = document.createElement('option')
            option.value = address
            option.textContent = address
            this.selectElement.appendChild(option)
        }
    }
}

class StorageMapsView {
    private storageMapKeysView: StorageMapKeysView
    private selectElement: HTMLSelectElement

    constructor(storageMapKeysView: StorageMapKeysView) {
        this.storageMapKeysView = storageMapKeysView
        this.selectElement = document.querySelector('#storage-maps select')!
        this.load().then(() => {
            this.selectElement.addEventListener('change', this.onChange.bind(this))
        })
    }

    onChange() {
        this.storageMapKeysView.domain = this.selectElement.value
    }

    async load() {
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

class StorageMapKeysView {
    private selectElement: HTMLSelectElement
    private valuesElement: HTMLDivElement
    private _address: string | null = null
    private _domain: string | null = null

    constructor() {
        this.selectElement = document.querySelector('#storage-map-keys select')!
        this.selectElement.addEventListener('change', this.onChange.bind(this))
        this.valuesElement = document.querySelector('#values')!
    }

    async onChange() {
        const key = this.selectElement.value
        this.valuesElement.innerHTML = ""

        const response = await request(`/accounts/${this.address}/${this.domain}/${key}`)
        const value = await response.json()
        new ValueView([key], value, this.valuesElement)
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

    async update() {
        this.valuesElement.innerHTML = ""

        const address = this.address
        const domain = this.domain
        if (!address || !domain) {
            return
        }

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
}

class ValueView {

    constructor(keyPath: string[], value: Value, parentElement: HTMLElement) {
        const valueElement = document.createElement('div')
        valueElement.classList.add('value')
        parentElement.appendChild(valueElement)

        let valueComponent: ReactNode | null
        switch (value.kind) {
            case "composite":
                valueComponent = createElement(CompositeValue, {keyPath, value})
                break
            case "dictionary":
                valueComponent = createElement(DictionaryValue, {keyPath, value})
                break
        }

        if (valueComponent) {
            createRoot(valueElement).render(valueComponent)
        }

        // const headingElement = document.createElement('h2')
        // headingElement.textContent = keyPath[keyPath.length - 1]
        // valueElement.appendChild(headingElement)
        //
        // const contentsElement = document.createElement('pre')
        // contentsElement.innerText = JSON.stringify(value, null, '  ')
        // valueElement.appendChild(contentsElement)
    }
}

document.addEventListener("DOMContentLoaded", function () {
    const storageMapKeysView = new StorageMapKeysView()
    new AccountsView(storageMapKeysView)
    new StorageMapsView(storageMapKeysView)
})