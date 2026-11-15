package main

import (
	"math"
	"sort"

	qt "github.com/mappu/miqt/qt6"
)

// spinningLogo draws the logo as a solid: a cube with a square hole through each pair of faces, built
// from the 20 blocks of a 3×3×3 cube left once the centre and the six face centres are taken out. It's
// drawn with QPainter, outer faces only, back faces culled and the rest sorted far to near, which costs
// next to nothing at the size it's shown.

type vec3 struct{ x, y, z float64 }

func (a vec3) add(b vec3) vec3      { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec3) scale(s float64) vec3 { return vec3{a.x * s, a.y * s, a.z * s} }
func (a vec3) dot(b vec3) float64   { return a.x*b.x + a.y*b.y + a.z*b.z }

type logoFace struct {
	corners [4]vec3
	normal  vec3
}

// logoFaces are the faces of the blocks that aren't against another block
func logoFaces() []logoFace {
	present := func(x, y, z int) bool {
		if x < -1 || x > 1 || y < -1 || y > 1 || z < -1 || z > 1 {
			return false
		}
		zeros := 0
		for _, c := range []int{x, y, z} {
			if c == 0 {
				zeros++
			}
		}
		return zeros <= 1
	}
	dirs := []struct {
		d    [3]int
		u, v vec3
	}{
		{[3]int{1, 0, 0}, vec3{0, 1, 0}, vec3{0, 0, 1}},
		{[3]int{-1, 0, 0}, vec3{0, 0, 1}, vec3{0, 1, 0}},
		{[3]int{0, 1, 0}, vec3{0, 0, 1}, vec3{1, 0, 0}},
		{[3]int{0, -1, 0}, vec3{1, 0, 0}, vec3{0, 0, 1}},
		{[3]int{0, 0, 1}, vec3{1, 0, 0}, vec3{0, 1, 0}},
		{[3]int{0, 0, -1}, vec3{0, 1, 0}, vec3{1, 0, 0}},
	}
	var faces []logoFace
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			for z := -1; z <= 1; z++ {
				if !present(x, y, z) {
					continue
				}
				centre := vec3{float64(x), float64(y), float64(z)}
				for _, d := range dirs {
					if present(x+d.d[0], y+d.d[1], z+d.d[2]) {
						continue
					}
					n := vec3{float64(d.d[0]), float64(d.d[1]), float64(d.d[2])}
					c := centre.add(n.scale(0.5))
					u, v := d.u.scale(0.5), d.v.scale(0.5)
					faces = append(faces, logoFace{
						corners: [4]vec3{
							c.add(u.scale(-1)).add(v.scale(-1)),
							c.add(u).add(v.scale(-1)),
							c.add(u).add(v),
							c.add(u.scale(-1)).add(v),
						},
						normal: n,
					})
				}
			}
		}
	}
	return faces
}

// rotate turns p about y by yaw, then about x by pitch, then about z by roll
func rotate(p vec3, yaw, pitch, roll float64) vec3 {
	cy, sy := math.Cos(yaw), math.Sin(yaw)
	p = vec3{p.x*cy + p.z*sy, p.y, -p.x*sy + p.z*cy}
	cp, sp := math.Cos(pitch), math.Sin(pitch)
	p = vec3{p.x, p.y*cp - p.z*sp, p.y*sp + p.z*cp}
	cr, sr := math.Cos(roll), math.Sin(roll)
	return vec3{p.x*cr - p.y*sr, p.x*sr + p.y*cr, p.z}
}

type spinningLogo struct {
	*qt.QWidget
	faces []logoFace
	timer *qt.QTimer
	angle float64
	color accent
	still bool
}

// frameInterval is about 30 frames a second, smooth for something turning this slowly
const frameInterval = 33

func newSpinningLogo(parent *qt.QWidget, color accent, still bool) *spinningLogo {
	l := &spinningLogo{QWidget: qt.NewQWidget(parent), faces: logoFaces(), color: color, still: still, angle: 0.6}
	l.SetAttribute(qt.WA_TransparentForMouseEvents)
	l.timer = qt.NewQTimer2(l.QObject)
	l.timer.OnTimeout(func() {
		l.angle += 0.012
		l.Update()
	})
	l.OnPaintEvent(func(super func(*qt.QPaintEvent), _ *qt.QPaintEvent) { l.paint() })
	// nothing to animate while nobody can see it
	l.OnShowEvent(func(super func(*qt.QShowEvent), e *qt.QShowEvent) {
		super(e)
		if !l.still {
			l.timer.Start(frameInterval)
		}
	})
	l.OnHideEvent(func(super func(*qt.QHideEvent), e *qt.QHideEvent) {
		super(e)
		l.timer.Stop()
	})
	return l
}

func (l *spinningLogo) setAccent(color accent) {
	l.color = color
	l.Update()
}

func (l *spinningLogo) setStill(still bool) {
	l.still = still
	if still {
		l.timer.Stop()
	} else if l.IsVisible() {
		l.timer.Start(frameInterval)
	}
	l.Update()
}

type projected struct {
	points [4][2]float64
	depth  float64
	light  float64
}

func (l *spinningLogo) paint() {
	w, h := float64(l.Width()), float64(l.Height())
	size := math.Min(w, h)
	yaw := l.angle
	// a constant tilt like the logo's own, and a slow nod so the holes show their depth
	pitch := 0.42 + 0.12*math.Sin(l.angle*0.7)
	roll := 0.26
	light := vec3{-0.45, 0.65, 0.62}
	const camera = 9.0
	// the cube's corners reach about 1.7 from its centre: fill most of the widget with it
	focal := size * 0.19
	scale := 1.0 / 1.5

	shown := make([]projected, 0, len(l.faces))
	for _, f := range l.faces {
		n := rotate(f.normal, yaw, pitch, roll)
		if n.z <= 0.001 {
			continue
		}
		var p projected
		for i, c := range f.corners {
			r := rotate(c.scale(scale), yaw, pitch, roll)
			k := focal * camera / (camera - r.z)
			p.points[i] = [2]float64{w/2 + r.x*k, h/2 - r.y*k}
			p.depth += r.z
		}
		p.light = 0.32 + 0.68*math.Max(0, n.dot(light))
		shown = append(shown, p)
	}
	sort.Slice(shown, func(i, j int) bool { return shown[i].depth < shown[j].depth })

	painter := qt.NewQPainter2(l.QPaintDevice)
	defer painter.Delete()
	defer painter.End()
	painter.SetRenderHint(qt.QPainter__Antialiasing)

	// a soft glow under it, in the accent
	glow := qt.NewQRadialGradient5(w/2, h/2, size*0.5)
	inner := qt.NewQColor11(l.color.r, l.color.g, l.color.b, 70)
	outer := qt.NewQColor11(l.color.r, l.color.g, l.color.b, 0)
	glow.SetColorAt(0, inner)
	glow.SetColorAt(1, outer)
	glowBrush := qt.NewQBrush10(glow.QGradient)
	area := qt.NewQRectF4(0, 0, w, h)
	painter.FillRect(area, glowBrush)
	for _, done := range []interface{ Delete() }{area, glowBrush, glow, inner, outer} {
		done.Delete()
	}

	edge := qt.NewQColor11(255, 255, 255, 28)
	pen := qt.NewQPen3(edge)
	pen.SetWidthF(0.8)
	painter.SetPenWithPen(pen)
	pen.Delete()
	edge.Delete()
	for _, p := range shown {
		shade := func(c int) int { return int(math.Min(255, float64(c)*p.light+18*p.light)) }
		color := qt.NewQColor3(shade(l.color.r), shade(l.color.g), shade(l.color.b))
		brush := qt.NewQBrush3(color)
		path := qt.NewQPainterPath()
		for i, pt := range p.points {
			point := qt.NewQPointF3(pt[0], pt[1])
			if i == 0 {
				path.MoveTo(point)
			} else {
				path.LineTo(point)
			}
			point.Delete()
		}
		path.CloseSubpath()
		painter.SetBrush(brush)
		painter.DrawPath(path)
		path.Delete()
		brush.Delete()
		color.Delete()
	}
}
