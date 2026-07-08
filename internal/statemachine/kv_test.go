package statemachine

import "testing"

func TestSet(t *testing.T) {
	kv := New()

	err := kv.Apply(Command{
		Operation: Set,
		Key:       "name",
		Value:     "Soumya",
	})

	if err != nil {
		t.Fatal(err)
	}

	value, ok := kv.Get("name")

	if !ok {
		t.Fatal("key not found")
	}

	if value != "Soumya" {
		t.Fatalf("expected Soumya got %s", value)
	}
}

func TestDelete(t *testing.T) {
	kv := New()

	_ = kv.Apply(Command{
		Operation: Set,
		Key:       "x",
		Value:     "10",
	})

	_ = kv.Apply(Command{
		Operation: Delete,
		Key:       "x",
	})

	_, ok := kv.Get("x")

	if ok {
		t.Fatal("key should have been deleted")
	}
}
