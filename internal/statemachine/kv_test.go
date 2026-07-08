package statemachine

import "testing"

func TestSet(t *testing.T) {

	kv := New()

	err := kv.Apply(Command{
		Operation: OpSet,
		Key:       "name",
		Value:     "Soumya",
	})

	if err != nil {
		t.Fatal(err)
	}

	value, ok := kv.Get("name")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "Soumya" {
		t.Fatalf("expected Soumya got %s", value)
	}
}

func TestDelete(t *testing.T) {

	kv := New()

	_ = kv.Apply(Command{
		Operation: OpSet,
		Key:       "x",
		Value:     "10",
	})

	_ = kv.Apply(Command{
		Operation: OpDelete,
		Key:       "x",
	})

	_, ok := kv.Get("x")

	if ok {
		t.Fatal("key should have been deleted")
	}
}

func TestEncodeDecode(t *testing.T) {

	cmd := Command{
		Operation: OpSet,
		Key:       "language",
		Value:     "go",
	}

	data, err := cmd.Encode()
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeCommand(data)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Key != cmd.Key {
		t.Fatal("key mismatch")
	}

	if decoded.Value != cmd.Value {
		t.Fatal("value mismatch")
	}

	if decoded.Operation != cmd.Operation {
		t.Fatal("operation mismatch")
	}
}

func TestSnapshot(t *testing.T) {

	kv := New()

	_ = kv.Apply(Command{
		Operation: OpSet,
		Key:       "a",
		Value:     "1",
	})

	snapshot := kv.Snapshot()

	if snapshot["a"] != "1" {
		t.Fatal("snapshot mismatch")
	}
}
