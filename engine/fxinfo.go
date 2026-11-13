package engine

import "sort"

// Look-and-feel data for the web shell's sprite-mode effects layer (prefix fx): weather, wind, time
// of day and the light sources in view. Read-only and cosmetic: it never draws g.R and changes
// nothing; the shell animates everything itself.

// FxLight is one light source on the map. R is the radius in cells; Flicker 0..1 is how much it wavers.
type FxLight struct {
	X       int     `json:"x"`
	Y       int     `json:"y"`
	R       float64 `json:"r"`
	Kind    string  `json:"k"`
	Flicker float64 `json:"fl"`
}

// FxInfo is what the effects layer needs for the current frame.
type FxInfo struct {
	Wx        string    `json:"wx,omitempty"`  // weather kind id (outdoors only)
	Overlay   string    `json:"ov,omitempty"`  // weather overlay: rain storm snow fog dust ash spore
	Intensity float64   `json:"in,omitempty"`  // 0..1
	Wind      int       `json:"wind"`          // -1 left, 0 calm, 1 right
	Phase     float64   `json:"ph"`            // 0..1 through the day, 0 = midnight
	Daylight  float64   `json:"dl"`            // 0 night .. 1 full day
	Indoors   bool      `json:"ind,omitempty"` // roofed tile, underground or a closed zone
	Dark      bool      `json:"dk,omitempty"`  // the zone is dark whatever the hour
	Depth     int       `json:"d,omitempty"`   // underground depth
	Lights    []FxLight `json:"lights,omitempty"`
}

func fxSmooth(a, b, x float64) float64 {
	t := (x - a) / (b - a)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

// fxDaylight is a smooth 0..1 brightness from the minute of day: dawn about 06:00, dusk about 20:00.
func fxDaylight(m float64) float64 {
	return fxSmooth(300, 420, m) * (1 - fxSmooth(1170, 1290, m))
}

var fxLightTerrain = map[Terrain]FxLight{
	TBrazier:    {R: 6, Kind: "torch", Flicker: 0.6},
	TLava:       {R: 2, Kind: "lava", Flicker: 0.25},
	TCrystal:    {R: 2, Kind: "crystal", Flicker: 0.08},
	TMushroom:   {R: 2, Kind: "glow", Flicker: 0.1},
	TAltar:      {R: 2, Kind: "altar", Flicker: 0.3},
	TRift:       {R: 3, Kind: "rift", Flicker: 0.4},
	TLantern:    {R: 3, Kind: "lantern", Flicker: 0.35},
	TChandelier: {R: 4, Kind: "lantern", Flicker: 0.3},
}

func fxIntensity(k *WeatherKind) float64 {
	switch k.ID {
	case "clear", "overcast":
		return 0
	case "heat":
		return 0.6
	case "frost":
		return 0.4
	}
	in := 0.45
	if k.Sight > 0 {
		in = 1 - float64(k.Sight)/26
	}
	if in < 0.3 {
		in = 0.3
	}
	if k.Bolt {
		in = 1
	}
	return in
}

// FxInfo describes the weather, time of day and lights for the map in view; nil on the world map.
func (g *Game) FxInfo() *FxInfo {
	if g.Cur == nil || g.P == nil || g.Mode == "world" || g.OnWorld {
		return nil
	}
	z := g.Cur
	min := float64((g.Turn + startMinutes) % DayLen)
	f := &FxInfo{Phase: min / DayLen, Daylight: fxDaylight(min), Depth: z.Z}
	f.Dark = z.Dark
	f.Indoors = !outdoors(z) || roofed(z.At(g.P.X, g.P.Y))
	if w := g.activeWeather(); w != nil && w.Kind != nil {
		f.Wx, f.Overlay, f.Intensity = w.Kind.ID, w.Kind.Overlay, fxIntensity(w.Kind)
		h := wxHash(g.W.wxSeed()^0x77d1, g.Turn/wxBucket, g.WX*31+g.WY)
		switch {
		case h < 0.3:
			f.Wind = -1
		case h > 0.7:
			f.Wind = 1
		}
	}
	// the player's own light first, as the field of view computes it
	p := g.P
	pr := p.LightRadius()
	if l := p.Pow("darkvision"); l > 0 {
		pr = maxi(pr, 7+l*2)
	}
	if !g.daylight() && z.Z == 0 {
		pr = maxi(pr, 4)
	}
	if pr < 2 {
		pr = 2
	}
	f.Lights = append(f.Lights, FxLight{X: p.X, Y: p.Y, R: float64(pr), Kind: "player", Flicker: 0.18})
	var cand []FxLight
	for y := 0; y < MH; y++ {
		for x := 0; x < MW; x++ {
			if !g.visible(x, y) {
				continue
			}
			i := y*MW + x
			l, ok := fxLightTerrain[z.T[i]]
			if z.Fire[i] > 0 {
				l, ok = FxLight{R: 3, Kind: "fire", Flicker: 0.8}, true
			}
			if !ok {
				continue
			}
			if (l.Kind == "lava" || l.Kind == "fire") && (x+y)%3 != 0 { // thin out big pools
				continue
			}
			l.X, l.Y = x, y
			cand = append(cand, l)
		}
	}
	for _, a := range z.Actors {
		if a.HP > 0 && a.St("burning") > 0 && g.visible(a.X, a.Y) {
			cand = append(cand, FxLight{X: a.X, Y: a.Y, R: 3, Kind: "fire", Flicker: 0.9})
		}
	}
	d := func(l FxLight) int { return (l.X-p.X)*(l.X-p.X) + (l.Y-p.Y)*(l.Y-p.Y) }
	sort.SliceStable(cand, func(i, j int) bool { return d(cand[i]) < d(cand[j]) })
	if len(cand) > 39 {
		cand = cand[:39]
	}
	f.Lights = append(f.Lights, cand...)
	return f
}
