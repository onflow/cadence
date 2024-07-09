# Nov 16, 2022

## Interface Inheritance Meeting Notes

* FLIP: https://github.com/onflow/flips/pull/40

* Forum discussion: https://forum.onflow.org/t/flip-interface-inheritance-in-cadence/3750

* Open Questions:
  * Functions with conditions:
    * FLIP proposes to order them in a pre-determined order and run them all.
      * No overriding is supported (because of security concerns).
    * Is overriding needed? Potentially unsafe to do so.

  * Default functions: Two main concerns.
    * Should allow overriding of default functions?
      * Can be a security/safety concern.
      * Someone in a middle of an inheritance chain can override a default function, which would change the behavior for downstream contracts.
      * One solution is to make default functions to be 'view' only.
        * Reduce the depth/impact of security concerns of overriding.
        * Still going to need a way to resolve ambiguity. e.g: Two ‘getId()’ view functions are available; which of the two should be called?
    * How to resolve ambiguity, when two or more default implementations are available for functions?
      * Two potential solutions:
        * Ask the user to solve it by overriding the method inside the concrete-type/interface which faces ambiguity
          (This is what is proposed in the FLIP).
          * Ambiguity resolution of default functions in concrete types also uses the same approach.
            See: https://github.com/onflow/cadence/pull/1076#discussion_r675861413
        * Order/linearize the default functions and pick the one that is 'closest' to the current interface/concrete type.
          * It is 'safe' only if the default functions are view only.
          * Might be surprising to the user.
          * Already disregarded this option for default functions ambiguity resolution in concrete implementations.
      * Need to resolve ambiguity regardless of whether default function overriding is supported or not.
