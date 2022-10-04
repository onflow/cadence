# Extensions Meeting Notes

FLIP: https://github.com/onflow/flow/pull/1101

## Two Paradigms
### Statically Checked
* Multiple different people could have the same extensions that do the same thing
* What if someone deletes the contract defining an extension?
### Dynamically Enforced
* Extensions would function like a “sub-resource”, method lookup would first happen on the extension, then fall back to the parent


# Discussion 
* Microplatforms that can be extended
    * Allow people to add features/functionality to their code without changing it, as this is the primary paradigm of smart contract development
        * Better for composability
        * People who add functionality should not be able to break the rules originally encoded into the base object
    * This paradigm benefits from thinking about extensions as an attachment rather than a wrapper; this is independent of whether or not we have a static representation
    * Owner of the resource **must** have control over what extensions are attached to the resource

* Are these two proposals even that different? 
    * Static version may be the same as the dynamic version just with added static typing
* Should it be possible to override the behavior of the base resource?
    * Potential to be dangerous, but also potential to be very powerful
    * E.g. Vault that does automatic currency conversion
        * This could also be done by just creating a different type that wraps a Vault (e.g. implementing Provider and Receiver), rather than extending the Vault type
    * Assumption up to this point has been: extension can add new functionality but cannot change any old functionality 
    * Extended version of a type is no longer an instance of the old type, which makes overriding the old type safe
        * Explicit casting 
* Metadata 
    * Metadata should reflect the extensions that are present on the type, rather than only showing the base resource
* Extension adds method that is later added by the base resource
    * What happens here?
    * In static model, the extension’s method would need to be removed
    * Could also require users to be explicit about which type (base or extension) they wish to use the method from (static dispatch)
        * Type.foo(instance)
        * Cadence’s flavor is more Pythonic than this very C++/Rust solution? Syntax is fundamentally different to what Cadence currently uses
    * Could also have the composed version default to the extension’s method and must be casted to the base type to use the base type’s method
        * Including the self pointer inside the definition of the extension itself
        * Or super keyword for extensions to refer to its base type
* Benefit of extensions over a purely compositional model is extensions have information about what base type an extension is attached to (e.g. via a self pointer)
* Signature extensions should not be removable and attachable to another resource
    * But it would make sense for a hat to be removable and given to another kitty
    * Need to allow extensions to decide whether they can be transferable or not
* Having extensions be their own resources/NFTs that can be traded is a nice feature if it’s easy, but is not necessary 
    * Extension could just be a manager for a resource type, rather than the extension being the resource itself
* Object and its extensions should be independent types that can define conflicting methods
    * E.g. metadata case
    * Need some method to go from extension to base and vice versa
        * While you have the extension, you only have access to those methods, and while you have the base you only have access to its methods
            * Extension has a reference to its parent?
            * Like trait objects in Rust
* Is this still extensions? Current proposal feels more like “attachments” rather than an extension of the base type
* Not the same as a dictionary of owned resources on the parent type
    * Must have a backpointer from attachment to parent
    * Use cases where the attachment should not be transferable
        * For transferable cases, the attachment can be a manager that handles the resources, rather than the resource itself
