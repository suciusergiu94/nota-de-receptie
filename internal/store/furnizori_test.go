package store

import "testing"

func TestFurnizoriiPornescGol(t *testing.T) {
	s := deschide(t)
	f, err := s.ListFurnizori()
	if err != nil {
		t.Fatalf("ListFurnizori: %v", err)
	}
	if len(f) != 0 {
		t.Errorf("len = %d, vrem 0", len(f))
	}
}

func TestRetineFurnizorIgnoraDuplicateleSiGolul(t *testing.T) {
	s := deschide(t)
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	for _, nume := range []string{"Alfa SRL", "alfa srl", "  ", "", "Beta SA"} {
		if err := retineFurnizor(tx, nume); err != nil {
			t.Fatalf("retineFurnizor(%q): %v", nume, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := s.ListFurnizori()
	if err != nil {
		t.Fatalf("ListFurnizori: %v", err)
	}
	if len(got) != 2 || got[0] != "Alfa SRL" || got[1] != "Beta SA" {
		t.Errorf("furnizori = %v, vrem [Alfa SRL Beta SA]", got)
	}
}

func TestDeleteFurnizor(t *testing.T) {
	s := deschide(t)
	tx, _ := s.db.Begin()
	retineFurnizor(tx, "Alfa SRL")
	retineFurnizor(tx, "Beta SA")
	tx.Commit()

	if err := s.DeleteFurnizor("alfa srl"); err != nil {
		t.Fatalf("DeleteFurnizor: %v", err)
	}
	got, _ := s.ListFurnizori()
	if len(got) != 1 || got[0] != "Beta SA" {
		t.Errorf("furnizori = %v, vrem [Beta SA]", got)
	}
}

func TestDeleteFurnizorInexistentNuEEroare(t *testing.T) {
	s := deschide(t)
	if err := s.DeleteFurnizor("Cineva"); err != nil {
		t.Errorf("DeleteFurnizor pe un nume inexistent: %v", err)
	}
}
