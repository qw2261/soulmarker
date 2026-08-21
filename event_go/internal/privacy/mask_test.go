package privacy

import "testing"

func TestMaskNameAndContact(t *testing.T) {
	tests := []struct {
		name, input, want string
		mask              func(string) string
	}{
		{"Chinese name", "张三", "张*", MaskName},
		{"long name", "Alice", "A***", MaskName},
		{"single name", "李", "*", MaskName},
		{"email", "member@example.com", "m***@example.com", MaskContact},
		{"phone", "138-0013-8000", "138****8000", MaskContact},
		{"other", "wechat-id", "w***", MaskContact},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.mask(test.input); got != test.want {
				t.Fatalf("got %q want %q", got, test.want)
			}
		})
	}
}
