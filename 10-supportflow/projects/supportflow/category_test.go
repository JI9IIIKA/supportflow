package supportflow

import "testing"

func TestCategory(t *testing.T) {
	for text, want := range map[string]string{"Ошибка входа": "technical", "Не прошла оплата": "billing", "Hello": "general"} {
		if got := Category(text); got != want {
			t.Fatalf("%q: got %s want %s", text, got, want)
		}
	}
}
