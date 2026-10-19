package engine_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

func TestReviewFactsTokenChoice(t *testing.T) {
	s, err := engine.NewGameFromScenario(3, engine.Options{}, engine.NamedScenario("law-supremacy"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := engine.ParseKey("construct:academy")
	if err := engine.Apply(&s, a, nil); err != nil {
		t.Fatal(err)
	}
	law, _ := engine.ParseKey("token:law")
	agriculture, _ := engine.ParseKey("token:agriculture")
	if f := engine.ReviewFacts(&s, law); !f.Wins || f.Win.Kind != engine.VictoryScience {
		t.Fatalf("winning token fact missing: %+v", f)
	}
	if f := engine.ReviewFacts(&s, agriculture); f.Wins {
		t.Fatalf("nonwinning token called a win: %+v", f)
	}
}
