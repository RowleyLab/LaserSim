package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 600
	screenHeight = 300
	cavityxdim   = 401                                    // The cavity x dimension
	cavityydim   = 101                                    // The cavity y dimension
	cavityxpos   = 150                                    // The cavity x position
	cavityypos   = 50                                     // The cavity y position
	cavityframes = 2*cavityxdim + 2                       // frames to traverse the cavity
	mediumxdim   = 51                                     // The lasing medium x dimension
	mediumydim   = 21                                     // The lasing medium y dimension
	mediumxpos   = cavityxpos + (cavityxdim-mediumxdim)/2 // The medium x position
	mediumypos   = cavityypos + (cavityydim-mediumydim)/2 // The medium y position
)

var (
	pump     = 0.002                   // Pump intensity
	isc      = 0.020                   // Rate of inter-system crossing
	decay    = 0.00004                 // Rate of spontaneous decay from the lasing state
	cross    = 0.15                    // Optical cross-section of the fluorophores
	qval     = 0.9                     // Current qualtiy factor of the cavity (reflectivity of end-mirrors)
	highqval = 0.9                     // Quality factor when cavity isn't being spoiled
	lowqval  = 0.05                    // Quality factor for a spoiled cavity
	autoq    = true                    // Automatic modulation of the q value
	highq    = true                    // quality factor is currently high
	photons  []*photon                 // All the photons currently in the simulation
	fluoros  [][]fluoro                // 2-d array of the lasing medium fluorophore molecules
	gnum     = mediumxdim * mediumydim // The number of fluorophores in the ground state
	enum     = 0                       // The number of fluorophores in the excited state
	lnum     = 0                       // The number of fluorophores in the lasing state
	cpnum    = 0                       // The number of photons in the cavity
	lpnum    = 0                       // The number of photons in the laser output
	framenum = 0                       // Frame number (increments each frame)
	gImage   *ebiten.Image             // Sprite for ground state fluorophores
	eImage   *ebiten.Image             // Sprite for excited state fluorophores
	lImage   *ebiten.Image             // Sprite for lasing state fluorophores
	pImage   *ebiten.Image             // Sprite for photons
	hqmImage *ebiten.Image             // Sprite for the high Q cavity end mirrors
	aomImage *ebiten.Image             // Sprite to indicate that automatic Q modulation is on
	lqmImage *ebiten.Image             // Sprite for the low Q cavity end mirrors
	lpImage  *ebiten.Image             // Sprite to graph the number of lasing photons (output)
	txtImage *ebiten.Image             // Instruction text
	opts     = &ebiten.DrawImageOptions{}
	r        = rand.New(rand.NewSource(time.Now().UnixNano())) // Random number generator
)

type photon struct {
	xpos int
	ypos int
	xvel int
	yvel int
}

type fluoro struct {
	state int // 0 => ground state, 1 => excited state, 2 => lasing state
	xpos  int // xposition (actual pixel position, not index)
	ypos  int // yposition (actual pixol position, not index)
}

func (f *fluoro) updateStates() int {
	if f.state == 0 {
		if r.Float64() < pump {
			f.state = 1
			gnum = gnum - 1
			enum = enum + 1
		}
	} else if f.state == 1 {
		if r.Float64() < pump {
			f.state = 0
			enum = enum - 1
			gnum = gnum + 1
		} else if r.Float64() < isc {
			f.state = 2
			enum = enum - 1
			lnum = lnum + 1
		}
	} else {
		if r.Float64() < decay {
			f.state = 0
			lnum = lnum - 1
			gnum = gnum + 1
			if r.Float64() < 0.1 { // Most fluoresced photons leave the cavity
				photons = append(photons, newPhoton(f.xpos, f.ypos, 0))
				cpnum = cpnum + 1
			}
		}
	}
	return f.state
}

func newPhoton(xposition int, yposition int, vel int) *photon {
	p := new(photon)
	p.xpos = xposition
	p.ypos = yposition
	if vel == 0 {
		if r.Float64() < 0.5 {
			p.xvel = 1
		} else {
			p.xvel = -1
		}
	} else {
		p.xvel = vel
	}
	return p
}

func init() {

	photons = make([]*photon, 0)
	fluoros = make([][]fluoro, mediumxdim)
	for i := 0; i < mediumxdim; i++ {
		fluoros[i] = make([]fluoro, mediumydim)
		for j := 0; j < mediumydim; j++ {
			fluoros[i][j].state = 0
			fluoros[i][j].xpos = mediumxpos + i + 1
			fluoros[i][j].ypos = mediumypos + j + 1
		}
	}
	gImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	gImage.Fill(color.RGBA{75, 75, 75, 255})
	eImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	eImage.Fill(color.RGBA{0, 0, 255, 255})
	lImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	lImage.Fill(color.RGBA{255, 75, 75, 255})
	pImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	pImage.Fill(color.RGBA{255, 0, 0, 25})
	hqmImage, _ = ebiten.NewImage(1, cavityydim, ebiten.FilterNearest)
	hqmImage.Fill(color.White)
	lqmImage, _ = ebiten.NewImage(1, cavityydim, ebiten.FilterNearest)
	lqmImage.Fill(color.RGBA{75, 75, 75, 255})
	aomImage, _ = ebiten.NewImage(11, 11, ebiten.FilterNearest)
	aomImage.Fill(color.RGBA{0, 255, 0, 255})
	lpImage, _ = ebiten.NewImage(10, 1, ebiten.FilterNearest)
	lpImage.Fill(color.RGBA{255, 0, 0, 255})
	txtImage, _ = ebiten.NewImage(screenWidth, 32, ebiten.FilterNearest)
	msg := "'Q' to quit   'R' to reset   'Left' to activate automatic Q-modulation\n'Right' to modulate Q manually   'Up' for high Q   'Down' for low Q"
	ebitenutil.DebugPrint(txtImage, msg)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Reset()
}

func drawhqMirrors(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	opts.GeoM.Translate(cavityxpos, cavityypos)
	screen.DrawImage(hqmImage, opts)
	opts.GeoM.Translate(cavityxdim+1, 0)
	screen.DrawImage(hqmImage, opts)
	qval = highqval
	return nil
}

func drawlqMirrors(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	opts.GeoM.Translate(cavityxpos, cavityypos)
	screen.DrawImage(lqmImage, opts)
	opts.GeoM.Translate(cavityxdim+1, 0)
	screen.DrawImage(hqmImage, opts)
	qval = lowqval
	return nil
}

func drawMedium(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	opts.GeoM.Translate(mediumxpos, mediumypos)
	for i := 0; i < mediumxdim; i++ {
		for j := 0; j < mediumydim; j++ {
			state := fluoros[i][j].updateStates()
			if state == 0 {
				screen.DrawImage(gImage, opts)
			} else if state == 1 {
				screen.DrawImage(eImage, opts)
			} else {
				screen.DrawImage(lImage, opts)
			}
			opts.GeoM.Translate(0, 1)
		}
		opts.GeoM.Translate(1, -mediumydim)
	}
	return nil
}

func updatePhoton(i int) (int, int) {
	p := photons[i]
	// This is a bit convluted, but minimizes the comparisons made as each region is exclusive
	if p.xpos >= mediumxpos {
		if p.xpos < mediumxpos+mediumxdim && p.ypos >= mediumypos && p.ypos < mediumypos+mediumydim { // In the medium
			f := &fluoros[p.xpos-mediumxpos][p.ypos-mediumypos]
			if f.state == 2 {
				if r.Float64() < cross { // Stimulated emission
					f.state = 0
					lnum = lnum - 1
					gnum = gnum + 1
					newx := p.xpos + r.Intn(3) - 1
					newy := p.ypos + r.Intn(3) - 1
					if newy < mediumypos || newy > mediumypos+mediumydim-1 {
						newy = p.ypos
					}
					photons = append(photons, newPhoton(newx, newy, p.xvel))
					cpnum = cpnum + 1
				}
			}
		} else if p.xpos > cavityxpos+cavityxdim { // At the cavity end-mirror
			p.xvel = -p.xvel
		}
	} else if p.xpos == cavityxpos { // At the output coupler
		if r.Float64() < qval { // Reflected
			p.xvel = -p.xvel
			//if i == 0 {
			//	fmt.Printf("photon 1 framenum = %+v\n", framenum)
			//}
		} else { // Transmitted
			cpnum = cpnum - 1
			lpnum = lpnum + 1
		}
	} else if p.xpos == 0 { // Off the screen
		// Fast way to remove the ith element of photons
		photons[i] = photons[len(photons)-1]
		photons[len(photons)-1] = nil
		photons = photons[:len(photons)-1]
		lpnum = lpnum - 1
		if i < len(photons) {
			return updatePhoton(i) // Update the photon that took this place
		}
		return 0, 0
	}
	p.xpos = p.xpos + p.xvel
	return p.xpos, p.ypos
}

func drawPhotons(screen *ebiten.Image) error {
	for i := 0; i < len(photons); i++ {
		xpos, ypos := updatePhoton(i) // This function is separate to allow recursion when dealing with a photon leaving the screen
		opts.GeoM.Reset()
		opts.GeoM.Translate(float64(xpos), float64(ypos))
		screen.DrawImage(pImage, opts)
	}
	return nil
}

func drawAOM(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	opts.GeoM.Translate(cavityxpos-5, cavityypos+cavityydim+1)
	screen.DrawImage(aomImage, opts)
	return nil
}

func drawGraphs(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	height := float64(gnum) / 6
	opts.GeoM.Scale(10.0, height)
	opts.GeoM.Translate(334.0, 290.0-height)
	screen.DrawImage(gImage, opts)
	opts.GeoM.Reset()
	height = float64(enum) / 6
	opts.GeoM.Scale(10.0, height)
	opts.GeoM.Translate(345.0, 290.0-height)
	screen.DrawImage(eImage, opts)
	opts.GeoM.Reset()
	height = float64(lnum) / 6
	opts.GeoM.Scale(10.0, height)
	opts.GeoM.Translate(356.0, 290.0-height)
	screen.DrawImage(lImage, opts)
	opts.GeoM.Reset()
	height = math.Sqrt(float64(lpnum)) * 4
	opts.GeoM.Scale(1.0, height)
	opts.GeoM.Translate(70.0, 290.0-height)
	screen.DrawImage(lpImage, opts)
	opts.GeoM.Reset()
	height = math.Sqrt(float64(cpnum)) * 1.5
	opts.GeoM.Scale(1.0, height)
	opts.GeoM.Translate(220, 290.0-height)
	screen.DrawImage(lpImage, opts)
	return nil
}
func resetFluoros() {
	for i := 0; i < mediumxdim; i++ {
		for j := 0; j < mediumydim; j++ {
			fluoros[i][j].state = 0
		}
	}
	photons = make([]*photon, 0)
	cpnum = 0
	lpnum = 0
	gnum = mediumxdim * mediumydim
	enum = 0
	lnum = 0
}

func update(screen *ebiten.Image) error {
	framenum = framenum + 1
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		os.Exit(0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		resetFluoros()
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		autoq = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		autoq = false
	}
	//if ebiten.IsKeyPressed(ebiten.KeyDown) {
	// qval = lowqval
	//} else {
	// qval = highqval
	//}
	if autoq {
		drawAOM(screen)
		if framenum%cavityframes > 50 {
			highq = false
		} else {
			highq = true
		}
	} else {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			highq = true
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			highq = false
		}
	}
	if cpnum > 50000 { // Failsafe for runaway lasing
		//fmt.Println("Number of cavity photons indicates a runaway process")
		os.Exit(1)
	}
	//if ebiten.CurrentFPS() < 10 {
	//	fmt.Println("FPS dropped too low")
	// os.Exit(1)
	//}
	if highq {
		drawhqMirrors(screen)
	} else {
		drawlqMirrors(screen)
	}
	opts.GeoM.Reset()
	screen.DrawImage(txtImage, opts)
	drawMedium(screen)
	drawPhotons(screen)
	drawGraphs(screen)
	//msg := fmt.Sprintf("TPS: %0.2f FPS: %0.2f cpnum: %d lpnum: %d gnum: %d enum: %d lnum: %d", ebiten.CurrentTPS(), ebiten.CurrentFPS(), cpnum, lpnum, gnum, enum, lnum)
	//ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Laser Cavity Simulator"); err != nil {
		log.Fatal(err)
	}
}
