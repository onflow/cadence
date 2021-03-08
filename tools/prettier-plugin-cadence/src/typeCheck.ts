import { AddressLocation, Location, StringLocation } from "./nodes"

export function isAddressLocation(location: any): location is AddressLocation {
	return location.Address !== undefined
}

export function isStringLocation(location: any): location is StringLocation {
	return location.String !== undefined
}
