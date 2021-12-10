package wmsstaticmap

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/twpayne/go-geom"
)

//Size : can either be Integer for the long side, or a struct consistant of width and height (both integers)
type Size struct {
	Width, Height int
}

type dim struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UnmarshalJSON : Own Json Unmarshal to check if type is int or struct
func (d *Size) UnmarshalJSON(data []byte) error {
	var x int
	if json.Unmarshal(data, &x) == nil {
		*d = Size{Width: x, Height: 1} // if size is int, return only width and calculate later
		return nil
	}
	var sd dim

	if err := json.Unmarshal(data, &sd); err != nil {
		return err
	}
	*d = Size(sd)
	return nil
}

// FetchMap : fetch a map from url and layer with bounds in EPSG:4326 and return a png with size
func FetchMap(url string, layer string, bounds *geom.MultiPoint, size Size, params map[string]string) (result string, err error) {
	if bounds.SRID() != 4326 {
		return "", errors.New("NO VALID SRID")
	}
	center := (bounds.Bounds().Max(1) + bounds.Bounds().Min(1)) / 2
	vh := (bounds.Bounds().Max(1) - bounds.Bounds().Min(1)) * 111                                      // Lattitude distance is 111km
	vw := (bounds.Bounds().Max(0) - bounds.Bounds().Min(0)) * 111.325 * math.Cos(math.Pi*(center)/180) // Longitude distance is 111,325km * cos(latitude)
	ratio := vw / vh
	if size.Height < 2 {
		size.Height = int(float64(size.Width) / ratio)
	}
	if size.Width < size.Height {
		size.Width, size.Height = size.Height, size.Width
	}
	paramString := ""
	if len(params) > 0 {
		allParams := make([]string, 0, len(params))
		for k, v := range params {
			allParams = append(allParams, k+"="+v)
		}
		paramString = "&" + strings.Join(allParams, "&")
	}
	call := fmt.Sprintf("%s?service=WMS&version=1.1.0&request=GetMap&layers=%s&bbox=%f,%f,%f,%f&width=%d&height=%d&srs=EPSG:4326&format=image/png%s", url, layer, bounds.Bounds().Min(0), bounds.Bounds().Min(1), bounds.Bounds().Max(0), bounds.Bounds().Max(1), size.Width, size.Height, paramString)
	log.Printf("%+v", call)
	response, err := http.Get(call)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	image := response.Body
	buf, err := ioutil.ReadAll(image)
	if err != nil {
		return "", err
	}

	defer image.Close()

	imgBase64Str := base64.StdEncoding.EncodeToString(buf)

	return imgBase64Str, nil
	//return fmt.Sprintf("Ratio: %f, %s, %s, %+v, %v", ratio, url, layer, bounds.Bounds(), size)
}

// GetXYfromLonLat
func GetXYfromLonLat(bounds *geom.MultiPoint, size Size, point [2]float64) (coordinates [2]int) {

	left := bounds.Bounds().Min(0)
	right := bounds.Bounds().Max(0)
	bottom := bounds.Bounds().Max(1)
	top := bounds.Bounds().Min(1)
	widthdegree := right - left
	heightdegree := bottom - top
	widthpoint := point[0] - left
	heightpoint := bottom - point[1]
	coordinates[0] = int(float64(size.Width) / widthdegree * widthpoint)
	coordinates[1] = int(float64(size.Height) / heightdegree * heightpoint)
	return coordinates
}
