package blockchain

import "math/big"

type Validator struct {
	Address []byte
	Stake   *big.Int
}

func (b *Blockchain) SelectValidator() []byte {
	// Упрощенный выбор по максимальному стейку
	var selectedValidator []byte
	maxStake := big.NewInt(0)

	for _, validator := range b.Validators {
		if validator.Stake.Cmp(maxStake) > 0 {
			maxStake = validator.Stake
			selectedValidator = validator.Address
		}
	}
	return selectedValidator
}

func (b *Blockchain) AddStake(address []byte, amount *big.Int) {
	for i, validator := range b.Validators {
		if bytes.Equal(validator.Address, address) {
			b.Validators[i].Stake.Add(validator.Stake, amount)
			return
		}
	}
	b.Validators = append(b.Validators, Validator{address, amount})
}