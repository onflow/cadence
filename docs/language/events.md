---
title: Events
---

Events are special values that can be emitted during the execution of a program.

An event type can be declared with the `event` keyword.

```cadence
event FooEvent(x: Int, y: Int)
```

The syntax of an event declaration is similar to that of
a [function declaration](functions#function-declarations);
events contain named parameters, each of which has an optional argument label.
Types that can be in event definitions are restricted
to booleans, strings, integer, and arrays or dictionaries of these types.

Events can only be declared within a [contract](contracts) body.
Events cannot be declared globally or within resource or struct types.

Resource argument types are not allowed because when a resource is used as
an argument, it is moved.  A piece of code would not want to move a resource
to emit an event, so it is not allowed as a parameter.

```cadence
// Invalid: An event cannot be declared globally
//
event GlobalEvent(field: Int)

pub contract Events {
    // Event with explicit argument labels
    //
    event BarEvent(labelA fieldA: Int, labelB fieldB: Int)

    // Invalid: A resource type is not allowed to be used
    // because it would be moved and lost
    //
    event ResourceEvent(resourceField: @Vault)
}

```

### Emitting events

To emit an event from a program, use the `emit` statement:

```cadence
pub contract Events {
    event FooEvent(x: Int, y: Int)

    // Event with argument labels
    event BarEvent(labelA fieldA: Int, labelB fieldB: Int)

    fun events() {
        emit FooEvent(x: 1, y: 2)

        // Emit event with explicit argument labels
        // Note that the emitted event will only contain the field names,
        // not the argument labels used at the invocation site.
        emit FooEvent(labelA: 1, labelB: 2)
    }
}
```

Emitting events has the following restrictions:

- Events can only be invoked in an `emit` statement.

  This means events cannot be assigned to variables or used as function parameters.

- Events can only be emitted from the location in which they are declared.
