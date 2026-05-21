package interpreter_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestAtreeSlabSplit(t *testing.T) {
	t.Parallel()

	logFunction := stdlib.NewInterpreterLogFunction(stdlib.FunctionLogger(func(message string) error {
		fmt.Fprintln(os.Stderr, message)
		return nil
	}))

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(logFunction)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, logFunction)

	inter, err := test_utils.ParseCheckAndInterpretWithAtreeValidationsDisabled(
		t,
		`
    access(all) entitlement Withdraw
    access(all) resource Vault {
        access(all) var balance: UFix64

        init(balance: UFix64) {
            self.balance = balance
        }

        access(Withdraw) fun withdraw(amount: UFix64): @Vault {
            self.balance = self.balance - amount
            return <- create Vault(balance: amount)
        }

        access(all) fun deposit(from: @Vault) {
            self.balance = self.balance + from.balance
            destroy from
        }
    }

    access(all) attachment A1 for Vault {
        access(all) var a1: String; access(all) var a2: String
        access(all) var a3: String; access(all) var a4: String
        init() { self.a1 = ""; self.a2 = ""; self.a3 = ""; self.a4 = "" }
        access(all) fun inflate() {
            self.a1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            self.a2 = self.a1; self.a3 = self.a1; self.a4 = self.a1
        }
    }
    access(all) attachment A2 for Vault {
        access(all) var b1: String; access(all) var b2: String
        access(all) var b3: String; access(all) var b4: String
        init() { self.b1 = ""; self.b2 = ""; self.b3 = ""; self.b4 = "" }
        access(all) fun inflate() {
            self.b1 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
            self.b2 = self.b1; self.b3 = self.b1; self.b4 = self.b1
        }
    }
    access(all) attachment A3 for Vault {
        access(all) var d1: String; access(all) var d2: String
        access(all) var d3: String; access(all) var d4: String
        init() { self.d1 = ""; self.d2 = ""; self.d3 = ""; self.d4 = "" }
        access(all) fun inflate() {
            self.d1 = "dddddddddddddddddddddddddddddddddddddd"
            self.d2 = self.d1; self.d3 = self.d1; self.d4 = self.d1
        }
    }
    access(all) attachment A4 for Vault {
        access(all) var e1: String; access(all) var e2: String
        access(all) var e3: String; access(all) var e4: String
        init() { self.e1 = ""; self.e2 = ""; self.e3 = ""; self.e4 = "" }
        access(all) fun inflate() {
            self.e1 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
            self.e2 = self.e1; self.e3 = self.e1; self.e4 = self.e1
        }
    }
    access(all) attachment A5 for Vault {
        access(all) var g1: String; access(all) var g2: String
        access(all) var g3: String; access(all) var g4: String
        init() { self.g1 = ""; self.g2 = ""; self.g3 = ""; self.g4 = "" }
        access(all) fun inflate() {
            self.g1 = "gggggggggggggggggggggggggggggggggggggg"
            self.g2 = self.g1; self.g3 = self.g1; self.g4 = self.g1
        }
    }
    access(all) attachment A6 for Vault {
        access(all) var h1: String; access(all) var h2: String
        access(all) var h3: String; access(all) var h4: String
        init() { self.h1 = ""; self.h2 = ""; self.h3 = ""; self.h4 = "" }
        access(all) fun inflate() {
            self.h1 = "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
            self.h2 = self.h1; self.h3 = self.h1; self.h4 = self.h1
        }
    }

    access(all) fun double(_ original: @Vault): @Vault {
        let empty <- original.withdraw(amount: 0.0)
        let stash <- original.withdraw(amount: 0.0)

        // Preparatory step: Attach a bunch of small and empty attachments
        let r1 <- attach A1() to <-original
        let r2 <- attach A2() to <-r1
        let r3 <- attach A3() to <-r2
        let r4 <- attach A4() to <-r3
        let r5 <- attach A5() to <-r4
        let r  <- attach A6() to <-r5

        // Create two EphemeralReferenceValues pointing to two different CompositeValues
        // which point to the same underlying dictionary
        var arr: @[Vault] <- [<-r]
        let ref  = &arr[0] as auth(Withdraw) &Vault
        let ref2 = &arr[0] as auth(Withdraw) &Vault


        // Trigger an atree slab split on the underlying dictionary of Vault
        // by "inflating" those attachments
        ref[A1]!.inflate()
        ref[A2]!.inflate()
        ref[A3]!.inflate()
        ref[A4]!.inflate()
        ref[A5]!.inflate()
        ref[A6]!.inflate() 

        // At this point, the CompositeValue inside "ref" has been properly updated such
        // that its dictionary.root points to the newly created root slab. However,
        // the CompositeValue inside "ref2" did not get updated and its dictionary.root
        // still points to the old slab which is no longer the root node. The atree slab
        // split reassigned slab IDs such that the OLD root (still pointed to by ref2's
        // dictionary) now has a different slab ID, while the NEW root inherits the
        // original slab ID.

        // Do a conversion roundtrip to create an EphemeralReferenceValue from ref2.
        // Without the stable cached valueID on CompositeValue, this reference would be
        // tracked under the (now-different) live ValueID of the stale view, which would
        // let it bypass invalidation when the vault is moved.
        let immortalRef = (ref2 as auth(Withdraw) &AnyResource) as! auth(Withdraw) &Vault
        // Move the vault. Reference invalidation must void "immortalRef" alongside
        // "ref" and "ref2": the cached valueID ensures all three are tracked under the
        // same stable ID.
        var extracted <- arr[0] <- empty

        stash.deposit(from: <- extracted)
        // This second withdraw must panic with InvalidatedResourceReferenceError,
        // because immortalRef was invalidated when the vault was moved above.
        stash.deposit(from: <- immortalRef.withdraw(amount: immortalRef.balance))

        destroy arr
        return <- stash
    }

    access(all) fun main() {
        let original <- create Vault(balance: 100.0)
        let second <- double(<- double(<- original))
        log("Succesfully withdrawn: \(second.balance)")
        destroy second
    }
        `,
		test_utils.ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *activations.Activation[interpreter.Variable] {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	// After an atree slab split, a stale view of the resource's dictionary can
	// produce a different live ValueID than the original. Without the cached
	// valueID on CompositeValue, an EphemeralReferenceValue created from such
	// a stale view would register under a different ID, bypassing invalidation
	// when the resource is moved, and allow the balance to be withdrawn twice.
	// The cached valueID ensures the stale reference is invalidated alongside
	// the others, so the second withdraw must fail.
	_, err = inter.Invoke("main")
	RequireError(t, err)
	var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
	assert.ErrorAs(t, err, &invalidatedResourceReferenceError)
}
