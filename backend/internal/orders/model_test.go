package orders

import "testing"

func TestCreateInputValidation(t *testing.T) {
	cases := []struct {
		name  string
		input CreateInput
		valid bool
	}{
		{"empty items", CreateInput{Customer: "Customer"}, false},
		{"empty customer", CreateInput{Items: []ItemInput{{SKU: "A", Qty: 1}}}, false},
		{"zero quantity", CreateInput{Customer: "Customer", Items: []ItemInput{{SKU: "A", Qty: 0}}}, false},
		{"duplicate", CreateInput{Customer: "Customer", Items: []ItemInput{{SKU: "A", Qty: 1}, {SKU: " A ", Qty: 2}}}, false},
		{"valid", CreateInput{Customer: " Customer ", Items: []ItemInput{{SKU: " A ", Qty: 1}}}, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.Validate()
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
			if test.valid && (test.input.Customer != "Customer" || test.input.Items[0].SKU != "A") {
				t.Fatal("input was not normalized")
			}
		})
	}
}
