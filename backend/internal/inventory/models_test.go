package inventory

import "testing"

// TestProductPriceRequired permits free items but rejects absent and negative prices.
func TestProductPriceRequired(t *testing.T) {
	zero, negative := 0, -1
	for _, test := range []struct {
		name  string
		price *int
		valid bool
	}{{"missing", nil, false}, {"free", &zero, true}, {"negative", &negative, false}} {
		t.Run(test.name, func(t *testing.T) {
			input := ProductInput{SKU: "A", Name: "Product", Price: test.price}
			if (input.Validate() == nil) != test.valid {
				t.Fatalf("unexpected validity for %s", test.name)
			}
		})
	}
}
