package engine

// Rng is splitmix64: tiny, fast, and its whole state is one saved number.
type Rng struct{ S uint64 }

func HashStr(s string) uint64 {
	h := uint64(1469598103934665603)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func NewRng(seed string) *Rng { return &Rng{S: HashStr(seed) | 1} }

func (r *Rng) next() uint64 {
	r.S += 0x9e3779b97f4a7c15
	z := r.S
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (r *Rng) Float() float64 { return float64(r.next()>>11) / float64(1<<53) }

// Int returns an integer in [lo, hi].
func (r *Rng) Int(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + int(r.next()%uint64(hi-lo+1))
}

func (r *Rng) Chance(p float64) bool { return r.Float() < p }

func (r *Rng) Dice(n, s int) int {
	t := 0
	for i := 0; i < n; i++ {
		t += r.Int(1, s)
	}
	return t
}

func Pick[T any](r *Rng, a []T) T { return a[r.Int(0, len(a)-1)] }

// value noise for zone terrain
func noise2(seed uint64, x, y float64) float64 {
	x0, y0 := int(floor(x)), int(floor(y))
	fx, fy := x-floor(x), y-floor(y)
	sx, sy := fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	v := func(ix, iy int) float64 {
		h := seed ^ uint64(ix)*0x9e3779b97f4a7c15 ^ uint64(iy)*0xc2b2ae3d27d4eb4f
		h = (h ^ (h >> 31)) * 0xbf58476d1ce4e5b9
		h ^= h >> 29
		return float64(h&0xffffff) / float64(0xffffff)
	}
	a := v(x0, y0) + (v(x0+1, y0)-v(x0, y0))*sx
	b := v(x0, y0+1) + (v(x0+1, y0+1)-v(x0, y0+1))*sx
	return a + (b-a)*sy
}

func fbm(seed uint64, x, y float64) float64 {
	return noise2(seed, x, y)*0.55 + noise2(seed+1, x*2.1, y*2.1)*0.3 + noise2(seed+2, x*4.3, y*4.3)*0.15
}

func floor(v float64) float64 {
	i := float64(int(v))
	if v < 0 && i != v {
		return i - 1
	}
	return i
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sign(a int) int {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}
