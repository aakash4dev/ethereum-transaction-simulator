package transaction

import (
	"math/big"
	"testing"
)

func TestParallelConfig(t *testing.T) {
	t.Run("ConfigCreation", func(t *testing.T) {
		config := &ParallelConfig{
			Value:           big.NewInt(100),
			GasLimit:        21000,
			Data:            []byte("test"),
			MaxTransactions: 10,
		}

		if config.Value.Cmp(big.NewInt(100)) != 0 {
			t.Error("Value should be 100")
		}

		if config.GasLimit != 21000 {
			t.Error("GasLimit should be 21000")
		}

		if config.MaxTransactions != 10 {
			t.Error("MaxTransactions should be 10")
		}
	})
}

