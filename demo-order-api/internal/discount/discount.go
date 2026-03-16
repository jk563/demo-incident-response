// Package discount provides promotional discount code lookups.
package discount

import "github.com/example/demo-incident-response/demo-order-api/internal/model"

// tiers defines the available discount tiers.
// There are 3 tiers at indices 0, 1, and 2.
var tiers = []model.DiscountTier{
	{Name: "bronze", Rate: 0.05},
	{Name: "silver", Rate: 0.10},
	{Name: "gold", Rate: 0.15},
}

// codeTierIndex maps promotional codes to tier indices.
// NOTE: WELCOME maps to index 3 which is out of bounds — this is intentional.
var codeTierIndex = map[string]int{
	"SAVE5":   0,
	"SAVE10":  1,
	"SAVE15":  2,
	"WELCOME": 3, // BUG: index out of range — do not fix
}

// Lookup returns the discount tier for the given promo code.
// Returns the tier and true if found, or a zero tier and false if the code
// is not recognised. Panics if WELCOME is used (intentional demo bug).
func Lookup(code string) (model.DiscountTier, bool) {
	idx, ok := codeTierIndex[code]
	if !ok {
		return model.DiscountTier{}, false
	}
	return tiers[idx], true // panics for WELCOME: index out of range [3] with length 3
}
