package payment_test

import (
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
)

func TestRegistry(t *testing.T) {
	r := payment.NewRegistry()
	if err := r.Register(mock.New(mock.Options{Name: "mock", Secret: "s"})); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(mock.New(mock.Options{Name: "vnpay-sandbox", Secret: "s"})); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(mock.New(mock.Options{Name: "mock", Secret: "s"})); err == nil {
		t.Fatal("duplicate provider accepted")
	}
	if err := r.Register(mock.New(mock.Options{Name: "Bad Name", Secret: "s"})); err == nil {
		t.Fatal("invalid name accepted")
	}
	if err := r.SetDefault("momo"); err == nil {
		t.Fatal("default on a disabled provider accepted")
	}
	if err := r.SetDefault("vnpay-sandbox"); err != nil || r.Default() != "vnpay-sandbox" {
		t.Fatalf("default = %q, err %v", r.Default(), err)
	}
	names := []string{}
	for _, p := range r.List() {
		names = append(names, p.Name())
	}
	if len(names) != 2 || names[0] != "mock" || names[1] != "vnpay-sandbox" {
		t.Fatalf("order = %v", names)
	}
	if _, ok := r.Get("momo"); ok {
		t.Fatal("unknown provider found")
	}
}
