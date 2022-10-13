
# Capabilities Meeting Notes

## Capability bootstrapping proposal

* FLIP: https://github.com/onflow/flow/pull/1122

* Publishing capabilities

* Last problem: Spam

    * Off-chain or on-chain (events, iteration API)

    * Quantity

    * Message could be harassment

    * Ideas:

        * Maybe no default/built-in/system event, but a custom event?

        * Not concern of protocol, but level above:

            * Allow/Ignore-list in user agent (e.g wallet software)

            * User agent could default to everything, but have harassment/filtering feature

        * Events are for making systems aware of on-chain state changes

        * Warning in documentation: 

            * Events should not be presented as-is to user

        * Having no events will result in user agents to repeatedly pull

        * Block explorer shows everything (today; could be extended, just like a user agent like a wallet)

    * Capabilities only?

        * Arbitrary publishing will result in avoiding capabilities

        * Should only be used to bridge to capabilities world

        * → design API so it can be extended to allow resources in the future in a backwards-compatible way

    * → only publish / claim / unpublish

    * Claim:

        * Event?

            * → Emit:

                * Not much harm

                * Useful in typical user agent (dashboard)

                * Need for reacting stays

            * Don’t emit:

                * Why?

        * Removal/unpublish? 

            * Don’t remove:

                * No harm in staying

                * Removing seems implicit

                * Receiver may rely on cap existing in publisher

            * → Remove:

                * Optimize for standard case

                * No need to constantly clean up

                * Mailbox analogy: someone shows up and claims → gone, no further action required by publisher

                * Forces receiver to "move" the capability to their account

    * → Need good documentation with examples

        * E.g. Claim cap, get resource, store resource, throw away cap

    * Related: event API needs filtering and pagination

    * Concern: Capabilities can be copied

## Capability Controllers

* FLIP: https://github.com/onflow/flow/pull/798

* Revocation problem example (admin interface)

* Capability creation only in publisher’s account?

* Admins should be able to undo, even each others’ decisions

* Not a problem of the language / API (should make it possible, something else can make it easier); capabilities API should be as simple (basic?) as possible

* Problems can mostly be solved with (better) attenuation

* Need delegation without changing type

* Attenuation allows custom code → security issues?

* Use cases for both concrete type and interface

* Tags?

* Mapping of names to IDs in contract?

* Use own name for capability. Does it require re-lookup based on ID?

* Keep reference count?

* Remove retarget? 

    * Security issue: retarget after check

    * Can only retarget to same type

    * Safety footgun

    * Problematic in two-step process

    * But always a problem

    * Examples:

        * Caps on interfaces, don’t care about concrete type, care about functionality

        * Concrete vault vs e.g pooling/split abstraction

* Remove restore?

* Struct, but non-storable

* Don’t return status flags. Either idempotent or panic

* → Attenuation out of scope

* → Example for delegation

* → Example for cap petnames

* → Default function petname should be default → easy to add later, move ahead without

