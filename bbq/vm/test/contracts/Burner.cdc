/// Burner is a contract that can facilitate the destruction of any resource on flow.
///
/// Contributors
/// - Austin Kline - https://twitter.com/austin_flowty
/// - Deniz Edincik - https://twitter.com/bluesign
/// - Bastian Müller - https://twitter.com/turbolent
access(all) contract Burner {
    /// When Crescendo (Cadence 1.0) is released, custom destructors will be removed from cadece.
    /// Burnable is an interface meant to replace this lost feature, allowing anyone to add a callback
    /// method to ensure they do not destroy something which is not meant to be,
    /// or to add logic based on destruction such as tracking the supply of a FT Collection
    ///
    /// NOTE: The only way to see benefit from this interface
    /// is to always use the burn method in this contract. Anyone who owns a resource can always elect **not**
    /// to destroy a resource this way
    access(all) resource interface Burnable {
        access(contract) fun burnCallback()
    }

    /// burn is a global method which will destroy any resource it is given.
    /// If the provided resource implements the Burnable interface,
    /// it will call the burnCallback method and then destroy afterwards.
    access(all) fun burn(_ toBurn: @AnyResource?) {
        if toBurn == nil {
            destroy toBurn
            return
        }
        let r <- toBurn!

        if let s <- r as? @{Burnable} {
            s.burnCallback()
            destroy s
        } else if let arr <- r as? @[AnyResource] {
            while arr.length > 0 {
                let item <- arr.removeFirst()
                self.burn(<-item)
            }
            destroy arr
        } else if let dict <- r as? @{HashableStruct: AnyResource} {
            let keys = dict.keys
            while keys.length > 0 {
                let item <- dict.remove(key: keys.removeFirst())!
                self.burn(<-item)
            }
            destroy dict
        } else {
            destroy r
        }
    }
}
