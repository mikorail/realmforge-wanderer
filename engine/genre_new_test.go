package engine

import (
	"os"
	"reflect"
	"testing"
)

var gnGenres = []string{"wasteland", "twilight"}

func TestGenreNewSkinComplete(t *testing.T) {
	for _, genre := range gnGenres {
		sk := SkinFor(genre)
		v := reflect.ValueOf(*sk)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			name := v.Type().Field(i).Name
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					t.Errorf("%s: empty %s", genre, name)
				}
			case reflect.Slice, reflect.Map:
				if name != "Creatures" && f.Len() == 0 {
					t.Errorf("%s: empty %s", genre, name)
				}
			case reflect.Array:
				for j := 0; j < f.Len(); j++ {
					if f.Index(j).Kind() == reflect.String && f.Index(j).String() == "" {
						t.Errorf("%s: empty %s[%d]", genre, name, j)
					}
				}
			}
		}
		for _, a := range archetypes {
			if sk.Callings[a] == "" {
				t.Errorf("%s: no calling %s", genre, a)
			}
		}
		for _, pd := range PowerDefs {
			if sk.PowerName(pd.ID) == "" {
				t.Errorf("%s: no power name %s", genre, pd.ID)
			}
		}
		if sk.Callings["warrior"] == baseSkin().Callings["warrior"] {
			t.Errorf("%s: callings not re-skinned", genre)
		}
	}
}

func TestGenreNewCreatureNames(t *testing.T) {
	for _, genre := range gnGenres {
		for i := range Templates {
			tp := &Templates[i]
			n, ok := tp.Names[genre]
			if !ok || n == "" {
				t.Errorf("%s: template %s has no %s name", tp.ID, tp.ID, genre)
			}
		}
	}
}

func TestGenreNewTables(t *testing.T) {
	for _, genre := range gnGenres {
		if _, ok := augSkins[genre]; !ok {
			t.Errorf("%s: no augSkin", genre)
		}
		if len(genreSigs[genre]) == 0 || len(genreWeather[genre]) == 0 || attireOverlay[genre] == "" {
			t.Errorf("%s: missing sigs/weather/attire entry", genre)
		}
		if len(muCasteNames[genre]) != len(muCasteNames[""]) || len(muHunterNames[genre]) == 0 || catSkin[genre] == nil {
			t.Errorf("%s: missing caste/hunter/category entry", genre)
		}
		for _, sw := range genreWeather[genre] {
			if WeatherByID(sw.To) == nil {
				t.Errorf("%s: unknown weather %s", genre, sw.To)
			}
		}
	}
}

func TestGenreNewPlay(t *testing.T) {
	b, err := os.ReadFile("../public/worlds/ember-crown.json")
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{"ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", ">", "<", "s", "g", "i", "a", "b", "c", "Enter", "Escape", "f", "t", "x", "j", "r", "m", "?", "e", "d", "z", "w", "Tab", "T", "C", "W", "O", "K", "V", "P", "Q", "o", "H", "F2", "1", "2", " "}
	for _, genre := range gnGenres {
		w, err := ParseWorld(string(b))
		if err != nil {
			t.Fatal(err)
		}
		w.Genre = genre
		_ = CharOptions(w)
		g := NewGame(w, CharSpec{Name: "Gn", Species: "human", Calling: "warrior", Attr: [6]int{16, 16, 16, 16, 16, 16}, Powers: map[string]int{"flames": 1, "wings": 1}})
		g.P.MaxHP, g.P.HP = 100000, 100000
		g.Render()
		r := NewRng("gn:" + genre)
		for i := 0; i < 2000; i++ {
			g.Key(keys[r.Int(0, len(keys)-1)])
			g.Render()
			if i%250 == 0 {
				g.Warp(r.Int(0, w.W-1), r.Int(0, w.H-1), r.Int(0, 2), r.Int(0, 2), r.Int(-1, 3))
			}
			if g.Dead {
				g = NewGame(w, CharSpec{Name: "Gn2", Species: "human", Calling: "scout", Attr: [6]int{16, 16, 16, 16, 16, 16}})
				g.P.MaxHP, g.P.HP = 100000, 100000
			}
		}
		if _, err := LoadGame(w, g.Save()); err != nil {
			t.Errorf("%s: save round trip: %v", genre, err)
		}
	}
}
