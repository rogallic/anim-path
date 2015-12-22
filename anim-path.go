package main

import (
	"fmt"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"encoding/xml"
	"io/ioutil"
	"unicode/utf8"
)

type Config struct {
	OutWidth, OutHeight int
	Speed               float64 // pixels per frame
	DashLen             float64
	framesInDay			int
}

type Camera struct {
	Frame       int
	OutWidth    int
	OutHeight   int
	CanvasWidth   int
	CanvasHeight  int
	X, Y, Scale float64
}

func NewCamera(config Config) *Camera {
	var c Camera = Camera{}
	c.Scale = 1
	c.OutWidth = config.OutWidth
	c.OutHeight = config.OutHeight
	c.X = float64(c.OutWidth) / 2
	c.Y = float64(c.OutHeight) / 2
	return &c
}

func (c *Camera) ToPoint(config Config, p Point) {
	var camW float64 = float64(c.OutWidth);
	var camH float64 = float64(c.OutHeight);
	var cnvW float64 = float64(c.CanvasWidth);
	var cnvH float64 = float64(c.CanvasHeight);
	var px float64 = p.X
	var py float64 = p.Y
	if px + camW / 2 > cnvW {
		px = cnvW - camW/2
	}
	if px - camW / 2 < 0 {
		px = camW / 2
	}
	if py + camH / 2 > cnvH {
		py = cnvH - camH/2
	}
	if py - camH / 2 < 0 {
		py = camH / 2
	}
	var dx float64 = px - c.X
	var dy float64 = py - c.Y
	c.X += dx / (100 / config.Speed)
	c.Y += dy / (100 / config.Speed)
}

func (c *Camera) CropAndWriteFrame(dest *image.RGBA) {
	var w float64 = float64(c.OutWidth)
	var h float64 = float64(c.OutHeight)
	var x float64 = c.X
	var y float64 = c.Y
	if x < w/2 {
		x = w / 2
	} else if x > float64(c.CanvasWidth)-w/2 {
		x = float64(c.CanvasWidth) - w/2
	}
	if y < h/2 {
		y = h / 2
	} else if y > float64(c.CanvasHeight)-h/2 {
		y = float64(c.CanvasHeight) - h/2
	}
	var rect = image.Rect(int(Round(x-w/2)), int(Round(y-h/2)), int(Round(x+w/2)), int(Round(y+h/2)))
	var cropped image.Image = dest.SubImage(rect)
	draw2dimg.SaveToPngFile(fmt.Sprintf("results/%v.png", c.Frame), cropped)
}

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

type Point struct {
	X float64
	Y float64
	stopType string
}

type AnimPath struct {
	Points          []Point
	DashLen         float64
	PathLen         float64
	MaxX            float64
	MaxY            float64
	Speed           float64 // count unscaled pixels
	DrawedLen       float64
	DrawedDashes    int
	CurOnDash       bool
	CurLin          int
	CurLinLen       float64
	CurLinDrawedLen float64
	CurLinDx        float64
	CurLinDy        float64
	FocusX          float64
	FocusY          float64
}

func NewAnimPath(config Config, Points []Point) *AnimPath {
	var p AnimPath = AnimPath{}
	p.Points = Points
	p.Speed = config.Speed
	p.DashLen = config.DashLen
	p.PathLen = p.getPathLen()
	p.MaxX, p.MaxY = p.getMax()
	p.setCurLin(0)
	return &p
}

func (p *AnimPath) getPathLen() float64 {
	if p.Points == nil || len(p.Points) < 2 {
		return 0
	}
	var dx float64
	var dy float64
	var sum float64 = 0
	var x = p.Points[0].X
	var y = p.Points[0].Y
	for _, point := range p.Points {
		dx = point.X - x
		dy = point.Y - y
		sum += math.Sqrt(dx*dx + dy*dy)
		x = point.X
		y = point.Y
	}
	return sum
}

func (p *AnimPath) getMax() (float64, float64) {
	var MaxX float64
	var MaxY float64
	for _, point := range p.Points {
		if point.X > MaxX {
			MaxX = point.X
		}
		if point.Y > MaxY {
			MaxY = point.Y
		}
	}
	return MaxX, MaxY
}

func (p *AnimPath) DrawNextFrame(gc *draw2dimg.GraphicContext) {
	var startFrameDrawedLen float64 = p.DrawedLen
	var endFrameDrawedLen float64 = startFrameDrawedLen + p.Speed
	var lineStart Point
	var ePortion float64
	var sPortion float64
	for p.DrawedLen < endFrameDrawedLen {
		if p.CurLin+1 >= len(p.Points) {
			p.DrawedLen = p.PathLen
			return
		}
		lineStart = p.Points[p.CurLin]
		var drawedDashes = p.DrawedLen / p.DashLen
		var toDashChangeLen float64 = p.DashLen * (float64(p.DrawedDashes+1) - drawedDashes)
		var toChangeLen float64
		var needNextLIne bool = false
		var needChangeDash bool = false
		if toDashChangeLen > p.CurLinLen-p.CurLinDrawedLen {
			needNextLIne = true
			toChangeLen = p.CurLinLen - p.CurLinDrawedLen
		} else {
			needChangeDash = true
			toChangeLen = toDashChangeLen
		}
		if toChangeLen > endFrameDrawedLen-p.DrawedLen {
			needChangeDash = false
			needNextLIne = false
			toChangeLen = endFrameDrawedLen - p.DrawedLen
		}
		sPortion = p.CurLinDrawedLen / p.CurLinLen
		ePortion = (p.CurLinDrawedLen + toChangeLen) / p.CurLinLen
		if p.CurOnDash {
			gc.MoveTo(lineStart.X+p.CurLinDx*sPortion, lineStart.Y+p.CurLinDy*sPortion)
			gc.LineTo(lineStart.X+p.CurLinDx*ePortion, lineStart.Y+p.CurLinDy*ePortion)
			gc.FillStroke()
		}
		p.CurLinDrawedLen += toChangeLen
		p.DrawedLen += toChangeLen
		if needNextLIne {
			p.setCurLin(p.CurLin + 1)
		}
		if needChangeDash {
			p.CurOnDash = !p.CurOnDash
			p.DrawedDashes++
		}
	}
	p.FocusX = lineStart.X + p.CurLinDx*ePortion
	p.FocusY = lineStart.Y + p.CurLinDy*ePortion
}

func (p *AnimPath) setCurLin(ind int) {
	p.CurLin = ind
	if ind+1 < len(p.Points) {
		p.CurLinDrawedLen = 0
		var lineStart Point = p.Points[p.CurLin]
		var lineEnd Point = p.Points[p.CurLin+1]
		var dx = lineEnd.X - lineStart.X
		var dy = lineEnd.Y - lineStart.Y
		p.CurLinLen = math.Sqrt(dx*dx + dy*dy)
		p.CurLinDx = dx
		p.CurLinDy = dy
	}
}

func DrawNextFrame(config Config, c *Camera, p *AnimPath, dest *image.RGBA, gc *draw2dimg.GraphicContext) bool {
	if p.DrawedLen >= p.PathLen-1 {
		return false
	}
	c.Frame++
	if c.Frame%10 == 0 {
		fmt.Println("frame #", c.Frame)
	}
	p.DrawNextFrame(gc)
	if c.Frame == 1 {
		c.X = p.FocusX
		c.Y = p.FocusY
	}
	if p.CurLin < len(p.Points) {
		c.ToPoint(config, Point{p.FocusX, p.FocusY, ""})
	}
	c.CropAndWriteFrame(dest)
	return true
}

func main() {
	var config Config = Config{320, 240, 3, 7, 0}
	var c = NewCamera(config)

	var err error

	var b []byte
    b, err = ioutil.ReadFile("path.svg")
    if err != nil {
    	fmt.Println(err)
    	return
    }
    svg := &Svg{}
    err = xml.Unmarshal(b, svg)
    if err != nil {
        fmt.Println(err)
        return
    }
    var maxLenPath int = -1
    for i, path := range svg.Paths {
    	if maxLenPath == -1 || len(path.D) >= len(svg.Paths[maxLenPath].D) {
    		maxLenPath = i
    	}
    }
    if maxLenPath == -1 {
    	fmt.Println("Not found path in path.svg")
    	return
    }
    var path Path = svg.Paths[maxLenPath]

	var points []Point = getPathPoints(path.D, 0, 0	)
	var p = NewAnimPath(config, points)

	var imgfile io.Reader
	imgfile, err = os.Open("./map.png")
	if err != nil {
		fmt.Println("Can not open map.png")
		return
	}
	if imgfile == nil {
		fmt.Println("Empty map.png")
		return
	}
	var img image.Image
	img, err = png.Decode(imgfile)
	if err != nil {
		fmt.Println("Can not decode map.png")
		return
	}
	if img == nil {
		fmt.Println("Empty map.png")
		return
	}
	imgfile, err = os.Open("./map.png")
	if err != nil {
		fmt.Println("Can not open map.png")
		return
	}
	if imgfile == nil {
		fmt.Println("Empty map.png")
		return
	}
	var imgConf image.Config
	imgConf, err = png.DecodeConfig(imgfile)
	if err != nil {
		fmt.Println("Can not decode config map.png")
		return
	}
	var destW int = maxInt(c.OutWidth, int(p.MaxX), imgConf.Width)
	var destH int = maxInt(c.OutHeight, int(p.MaxY), imgConf.Height)
	c.CanvasWidth = destW
	c.CanvasHeight = destH
	var dest *image.RGBA = image.NewRGBA(image.Rect(-destW, -destH, destW*2, destH*2))
	draw2dimg.DrawImage(img, dest, draw2d.NewIdentityMatrix(), 0, 0)
	var gc *draw2dimg.GraphicContext = draw2dimg.NewGraphicContext(dest)
	gc.SetStrokeColor(color.RGBA{0x66, 0x00, 0xff, 0xff})
	gc.SetLineWidth(5)

	for DrawNextFrame(config, c, p, dest, gc) {
	}
}

func maxInt(vls ...int) int {
	var max int = vls[0]
	for _, v := range vls {
		if v > max {
			max = v
		}
	}
	return max
}

type Svg struct {
    XMLName xml.Name `xml:"svg"`
    Paths   []Path  `xml:"g>path"`
}
type Path struct {
	D string `xml:"d,attr"`
	Transform string `xml:"transform,attr,omitempty"`
	Style string `xml:"style,attr,omitempty"`
	Id string `xml:"id,attr,omitempty"`
}

func getPathPoints(str string, marginX, marginY float64) []Point {
	var strPoints []string = strings.Split(str, " ")
	var points []Point = make([]Point, len(strPoints))
	var prevX float64
	var prevY float64
	var minX float64
	var minY float64
	var strPointsLen int = len(strPoints);
	var l int = 0 // length of points
	var absoluteCoordsMode bool = false
	var i int = 0
	for ; i < strPointsLen; i++ {
		var s string = strPoints[i]
		if len(s) == 1 {
			var r rune
			r, _ = utf8.DecodeRuneInString(s)
			if r > 64 {
				if s == "S" || s == "C" || s == "s" || s == "c" {
					i += 2
				}
				if r > 96 {
					absoluteCoordsMode = false
				} else {
					absoluteCoordsMode = true
				}
				continue
			}
		}
		var dxdy []string = strings.Split(s, ",")
		var dx float64
		dx, _ = strconv.ParseFloat(dxdy[0], 64)
		var dy float64
		dy, _ = strconv.ParseFloat(dxdy[1], 64)
		if absoluteCoordsMode {
			prevX = dx
			prevY = dy
		} else {
			prevX += dx
			prevY += dy
		}
		var point Point = Point{prevX, prevY, ""}
		if point.X < minX {
			minX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		points[l] = point
		l++
	}
	var retPoints []Point = make([]Point, l)
	for i = 0; i < l; i++ {
		var point = points[i]
		retPoints[i] = Point{point.X/* - minX*/ + marginX, point.Y/* - minY*/ + marginY, ""}
	}
	return retPoints
}