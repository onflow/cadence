package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/interpreter"
)

func TestInterpretCapabilityBorrow(t *testing.T) {

	inter, _ := testAccount(
		t,
		true,
		`
          resource R {
              let foo: Int

              init() {
                  self.foo = 42
              }
          }

          fun saveAndLink() {
              let r <- create R()
              account.save(<-r, to: /storage/r)

              account.link<&R>(/public/single, target: /storage/r)
              
              account.link<&R>(/public/double, target: /public/single)
              
              account.link<&R>(/public/nonExistent, target: /storage/nonExistent)
              
              account.link<&R>(/public/loop1, target: /public/loop2)
              account.link<&R>(/public/loop2, target: /public/loop1)
          }

          fun foo(_ path: Path): Int {
              return account.getCapability(path)!.borrow<&R>()!.foo
          }

          fun single(): Int { 
              return foo(/public/single)
          }

          fun double(): Int { 
              return foo(/public/double)
          }

          fun nonExistent(): Int { 
              return foo(/public/nonExistent)
          }

          fun loop(): Int { 
              return foo(/public/loop1)
          }
        `,
	)

	// save

	_, err := inter.Invoke("saveAndLink")
	require.NoError(t, err)

	t.Run("single", func(t *testing.T) {

		value, err := inter.Invoke("single")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewIntValue(42), value)
	})

	t.Run("double", func(t *testing.T) {

		value, err := inter.Invoke("double")
		require.NoError(t, err)

		require.Equal(t, interpreter.NewIntValue(42), value)
	})

	t.Run("nonExistent", func(t *testing.T) {

		_, err := inter.Invoke("nonExistent")
		require.Error(t, err)

		require.IsType(t, &interpreter.ForceNilError{}, err)
	})

	t.Run("loop", func(t *testing.T) {

		_, err := inter.Invoke("loop")
		require.Error(t, err)

		require.IsType(t, &interpreter.CyclicLinkError{}, err)

		require.Equal(t,
			err.Error(),
			"cyclic link in account 0x2a: /public/loop1 -> /public/loop2 -> /public/loop1",
		)
	})
}
