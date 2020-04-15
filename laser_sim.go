package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
)

const (
	screenWidth  = 1250
	screenHeight = 500
	cavityxdim   = 601                                    // The cavity x dimension
	cavityydim   = 201                                    // The cavity y dimension
	cavityxpos   = 100                                    // The cavity x position
	cavityypos   = (screenHeight - cavityydim) / 2        // The cavity y position
	mediumxdim   = 121                                    // The lasing medium x dimension
	mediumydim   = 81                                     // The lasing medium y dimension
	mediumxpos   = cavityxpos + (cavityxdim-mediumxdim)/2 // The medium x position
	mediumypos   = cavityypos + (cavityydim-mediumydim)/2 // The medium y position
)

var (
	pump    int           // Pump intensity
	isc     int           // Rate of inter-system crossing
	decay   int           // Rate of spontaneous decay from the lasing state
	cross   int           // Optical cross-section of the fluorophores
	photons []photon      // All the photons currently in the simulation
	fluoros [][]fluoro    // 2-d array of the lasing medium fluorophore molecules
	qval    int           // Qualtiy factor of the cavity (reflectivity of end-mirrors)
	gnum    int           // The number of fluorophores in the ground state
	enum    int           // The number of fluorophores in the excited state
	lnum    int           // The number of fluorophores in the lasing state
	gImage  *ebiten.Image // Sprite for ground state fluorophores
	eImage  *ebiten.Image // Sprite for excited state fluorophores
	lImage  *ebiten.Image // Sprite for lasing state fluorophores
	pImage  *ebiten.Image // Sprite for photons
	opts    = &ebiten.DrawImageOptions{}
)

type photon struct {
	xpos  int
	ypos  int
	group int
}

type fluoro struct {
	state int // 0 => ground state, 1 => excited state, 2 => lasing state
}

func (f *fluoro) updateStates() int {
	if f.state == 0 {
		if rand.Intn(1000) < pump {
			f.state = 1
		}
	} else if f.state == 1 {
		if rand.Intn(1000) < pump {
			f.state = 0
		} else if rand.Intn(1000) < isc {
			f.state = 2
		}
	} else {
		if rand.Intn(1000) < decay {
			f.state = 0
			// TODO: create a photon
		}
	}
	return f.state
}

func init() {
	fluoros = make([][]fluoro, mediumxdim)
	for i := 0; i < mediumxdim; i++ {
		fluoros[i] = make([]fluoro, mediumydim)
		for j := 0; j < mediumydim; j++ {
			fluoros[i][j].state = 0
		}
	}
	gnum = len(fluoros) * len(fluoros[0])
	fmt.Printf("gnum = %+v\n", gnum)
	fmt.Printf("mediumxdim = %+v\n", mediumxdim)
	fmt.Printf("mediumydim = %+v\n", mediumydim)
	gImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	gImage.Fill(color.White)
	eImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	eImage.Fill(color.RGBA{0, 0, 255, 255})
	lImage, _ = ebiten.NewImage(1, 1, ebiten.FilterNearest)
	lImage.Fill(color.RGBA{255, 0, 0, 255})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Reset()
	pump = 50
	isc = 5
	decay = 5
	cross = 50
}

func drawMedium(screen *ebiten.Image) error {
	opts.GeoM.Reset()
	opts.GeoM.Translate(mediumxpos, mediumypos)
	for i := 0; i < mediumxdim; i++ {
		opts.GeoM.Translate(1, 0)
		for j := 0; j < mediumydim; j++ {
			opts.GeoM.Translate(0, 1)
			state := fluoros[i][j].updateStates()
			if state == 0 {
				screen.DrawImage(gImage, opts)
			} else if state == 1 {
				screen.DrawImage(eImage, opts)
			} else {
				screen.DrawImage(lImage, opts)
			}
		}
		opts.GeoM.Translate(0, -mediumydim)
	}
	return nil
}

func resetFluoros() {
	for i := 0; i < mediumxdim; i++ {
		for j := 0; j < mediumydim; j++ {
			fluoros[i][j].state = 0
		}
	}
}

func update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		os.Exit(0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		resetFluoros()
	}
	drawMedium(screen)
	msg := "Test Screen"
	ebitenutil.DebugPrint(screen, msg)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 1, "Lasing Threshold Simulator"); err != nil {
		log.Fatal(err)
	}
}
