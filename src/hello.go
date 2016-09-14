// hello.go
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	//"unicode/utf8"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/golang/freetype/truetype"
	"github.com/satori/go.uuid"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Image struct {
	Id   string
	Text string
	Path string
}

type CreateRequest struct {
	Id          string `json:"id"`
	To          string `json:"to"`
	From        string `json:"from"`
	Message     string `json:"message"`
	MessageType string `json:"messagetype"`
	FontSize    string `json:"fontsize"`
	TextColor   string `json:"textcolor"`
}

type CreateResponse struct {
	Id           string `json:"id"`
	ImageSrc     string `json:"imageSrc"`
	ImageViewUrl string `json:"imageViewUrl"`
}

type DeleteRequest struct {
	Id string `json:"id"`
}

var (
	width      = flag.Int("width", 720, "width of the image in pixels")
	height     = flag.Int("height", 1280, "height of the image in pixels")
	dpi        = flag.Float64("dpi", 72, "screen resolution in Dots Per Inch")
	fontfile   = flag.String("fontfile", "./fonts/Merged-SFUI-NotoDevanagari-Emoji.ttf", "filename of the ttf font")
	hinting    = flag.String("hinting", "none", "none | full")
	size       = flag.Float64("size", 24, "font size in points")
	footersize = flag.Float64("footersize", 30, "font size for footer in points")
	spacing    = flag.Float64("spacing", 1.5, "line spacing (e.g. 2 means double spaced)")
	wonb       = flag.Bool("whiteonblack", false, "white text on a black background")
	footerurl  = flag.String("footerurl", "Made using - www.desistickers.azurewebsites.net", "Url of the site")
)

var insightsClient appinsights.TelemetryClient

func checkErrorAndPanic(err error) {
	if err != nil {
		fmt.Printf("WriteFileJson ERROR: %+v", err)
		panic(err)
	}
}

func loadImage(id string) (*Image, error) {
	filename := "./stickers/" + id + ".txt"
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Printf("ReadFile ERROR: %+v", err)
		return nil, err
	}

	image := Image{}
	err = json.Unmarshal(data, &image)
	checkErrorAndPanic(err)

	// Create the image object from the contents of the file
	//text := "sample text"
	//path := "/stickers/samplePath.jpg"

	return &image, nil
}

func getImageData(id string) ([]byte, error) {
	filename := "./stickers/" + id + ".jpg"
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Printf("ReadFile ERROR: %+v", err)
		return nil, err
	}

	return data, nil
}

func saveImage(img *Image) error {
	filename := "./stickers/" + img.Id + ".txt"
	data, err := json.Marshal(img)
	checkErrorAndPanic(err)

	return ioutil.WriteFile(filename, data, 0644)
}

func deleteImage(id string) error {
	filename := "./stickers/" + id + ".jpg"

	// Remove the file from the disk
	err := os.Remove(filename)
	if err != nil {
		return err
	}

	return nil
}

func renderTemplate(w http.ResponseWriter, p *Image, templateName string) {
	t, _ := template.ParseFiles(templateName)
	t.Execute(w, p)
}

func createNewImageId() string {
	//id := "abcd"
	newUuid := uuid.NewV1()
	id := newUuid.String()
	// Implement the method to create a new guid based string
	return id
}

//
// Image rendering methods
//

func createFont(fontFileName string) (*truetype.Font, error) {
	// Read the font data.
	fontBytes, err := ioutil.ReadFile(fontFileName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return f, err
}

func createNewImageWithBackground(imgW int, imgH int, filePath string) *image.RGBA {
	// Render background on the image buffer
	start := time.Now()

	var imageBackground image.Image
	var jpegErr error

	if filePath != "" {
		reader, err := os.Open(filePath)
		checkErrorAndPanic(err)

		imageBackground, jpegErr = jpeg.Decode(reader)
		checkErrorAndPanic(jpegErr)
	} else {
		imageBackground = image.NewUniform(color.RGBA{0xff, 0xff, 0xff, 0xff})
	}

	elapsed := time.Since(start)
	insightsClient.TrackMetric("CreateBaseImage", float32(elapsed.Seconds()))

	if *wonb {
		imageBackground = image.Black
	}

	rgba := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(rgba, rgba.Bounds(), imageBackground, image.ZP, draw.Src)

	return rgba
}

func getTopMargin(totalLineHeight int, availableHeight int) int {
	topMargin := 0
	if totalLineHeight >= availableHeight {
		topMargin = 10
	} else {
		topMargin = (availableHeight - totalLineHeight) / 2
	}

	return topMargin
}

func generateMessage(to string, from string, text string) []string {
	message := ""
	// Analyze the text and start drawing
	if to != "" {
		message = to + ",\n \n"
	}

	if text != "" {
		message += text
	}

	if from != "" {
		message += "\n \n" + from
	}

	lines := strings.Split(message, "\n")

	return lines
}

func breakLineToFit(d *font.Drawer, line string, imageWidth int, marginH int) []string {

	reflowedLines := []string{}

	lineWidth := d.MeasureString(line).Round()

	if (lineWidth + marginH) >= imageWidth {

		newLine := ""
		remainingLine := ""
		cutReached := false

		lines := strings.Split(line, " ")
		for _, line := range lines {
			lenNewLine := d.MeasureString(newLine).Round()
			lenLine := d.MeasureString(line).Round()

			if (lenNewLine+lenLine+marginH) >= imageWidth || cutReached == true {
				remainingLine += line + " "
				cutReached = true
			} else {
				newLine += line + " "
			}
		}

		if newLine == "" {
			reflowedLines = append(reflowedLines, line)
			return reflowedLines
		}

		reflowedLines = append(reflowedLines, newLine)
		splitLines := breakLineToFit(d, remainingLine, imageWidth, marginH)
		reflowedLines = append(reflowedLines, splitLines...)
	} else {
		reflowedLines = append(reflowedLines, line)
	}

	//newlines := []string{line}
	return reflowedLines
}

func reflowText(lines []string, d *font.Drawer, imageWidth int, marginH int) []string {
	finalLines := []string{}

	// Trim the lines to fit the width of the image as per the font size
	for _, line := range lines {
		lineWidth := d.MeasureString(line).Round()

		if (lineWidth + marginH) >= imageWidth {
			splitLines := breakLineToFit(d, line, imageWidth, marginH)

			finalLines = append(finalLines, splitLines...)
		} else {
			finalLines = append(finalLines, line)
		}
	}

	return finalLines
}

func reformatHindiGlyph(lines []string) []string {

	var maatra rune = 'ि'
	newLines := []string{}

	for _, line := range lines {
		newLine := ""
		words := strings.Split(line, " ")

		for _, word := range words {
			newWord := ""
			// Check if the word contains the "i" vowel
			if strings.ContainsRune(word, maatra) {
				var lastRune rune
				for i, runeD := range word {
					//fmt.Printf("%c %d\n", runeD, i)

					//interchange the order of "i" vowel
					if runeD == maatra {
						index := strings.IndexRune(word, lastRune)
						size := 3
						newWord = word[:index] + string(runeD) + string(lastRune) + word[i+size:]
						//fmt.Println(word)
						//fmt.Println(newWord)
					}
					lastRune = runeD
				}
			} else {
				newWord = word
			}

			// Append the word to the new line
			if newLine == "" {
				newLine = newWord
			} else {
				newLine += " " + newWord
			}
		}

		// Append the new lines to the lines
		newLines = append(newLines, newLine)
	}

	return newLines
}

func createImage(id string, to string, from string, text string, messageType string, fontSize float64, textColor string) (*Image, error) {
	// Create an image using the input and save it locally
	//path := baseImageName

	imgW, imgH := *width, *height

	// Create the font
	f, err := createFont(*fontfile)
	checkErrorAndPanic(err)

	// Create a foreground color
	fg := image.Black
	if *wonb {
		fg = image.White
	}

	if textColor == "White" {
		fg = image.White
	} else if textColor == "Black" {
		fg = image.Black
	} else if textColor == "Red" {
		fg = image.NewUniform(color.RGBA{0xff, 0x00, 0x00, 0xff})
	} else if textColor == "Blue" {
		fg = image.NewUniform(color.RGBA{0x00, 0x00, 0xff, 0xff})
	} else if textColor == "Yellow" {
		fg = image.NewUniform(color.RGBA{0x00, 0xff, 0xff, 0xff})
	}

	filePath := "baseImages/7.jpg"
	if messageType == "Happy Birthday" {
		filePath = "baseImages/5.jpg"
	} else if messageType == "Happy Anniversary" {
		filePath = "baseImages/1.jpg"
	} else if messageType == "Festival" {
		filePath = "baseImages/6.jpg"
	} else if messageType == "Inspirational" {
		filePath = "baseImages/8.jpg"
	} else if messageType == "Relationship" {
		filePath = "baseImages/9.jpg"
	} else if messageType == "Quote" {
		filePath = "baseImages/3.jpg"
	}

	//Create new image with background
	rgba := createNewImageWithBackground(imgW, imgH, filePath)

	// Create a font drawer
	d := createDrawer(rgba, fg, f, fontSize, *dpi, font.HintingNone)

	// Generate the final text
	lines := generateMessage(to, from, text)

	// Reflow the text to fit the image width
	reflowLines := reflowText(lines, d, imgW, 20)
	finalLines := reformatHindiGlyph(reflowLines)

	lineHeight := int(math.Ceil(1.1 * fontSize * (*dpi / 72)))
	totalHeight := len(finalLines) * lineHeight
	y := lineHeight + getTopMargin(totalHeight, *height)

	// Render the final message
	for _, renderLine := range finalLines {
		d.Dot = fixed.Point26_6{
			X: (fixed.I(imgW) - d.MeasureString(renderLine)) / 2,
			Y: fixed.I(y),
		}
		d.DrawString(renderLine)
		y += lineHeight
	}

	// Render the watermark (url) in the footer
	renderWatermark(rgba, fg, f, imgW, imgH)

	// Save the image to disk
	filename := "./stickers/" + id + ".jpg"
	saveImageFile(filename, rgba)

	return &Image{Id: id, Text: text, Path: filename}, nil
}

func renderWatermark(rgba draw.Image, fg image.Image, f *truetype.Font, imgW int, imgH int) {
	d := createDrawer(rgba, fg, f, *footersize, *dpi, font.HintingNone)

	// Render the final set of lines
	renderLine := *footerurl
	y := imgH - 10

	d.Dot = fixed.Point26_6{
		X: (fixed.I(imgW) - d.MeasureString(renderLine)) / 2,
		Y: fixed.I(y),
	}
	d.DrawString(renderLine)
}

func createDrawer(rgba draw.Image, fg image.Image, f *truetype.Font, fontSize float64, dpi float64, h font.Hinting) *font.Drawer {
	d := &font.Drawer{
		Dst: rgba,
		Src: fg,
		Face: truetype.NewFace(f, &truetype.Options{
			Size:    fontSize,
			DPI:     dpi,
			Hinting: h,
		}),
	}

	return d
}

func saveImageFile(filename string, buffer *image.RGBA) {

	start := time.Now()

	// Save that RGBA image to disk.
	outFile, err := os.Create(filename)
	checkErrorAndPanic(err)

	defer outFile.Close()
	b := bufio.NewWriter(outFile)

	err = jpeg.Encode(b, buffer, nil)
	//encoder := png.Encoder{CompressionLevel: png.BestCompression}
	//err = encoder.Encode(b, buffer)
	checkErrorAndPanic(err)

	err = b.Flush()
	checkErrorAndPanic(err)

	elapsed := time.Since(start)

	insightsClient.TrackMetric("SaveImage", float32(elapsed.Seconds()))
}

//
// Handlers
//

func createHandler(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	decoder := json.NewDecoder(r.Body)
	var inputData CreateRequest

	err := decoder.Decode(&inputData)
	checkErrorAndPanic(err)

	id := inputData.Id
	to := inputData.To
	from := inputData.From
	text := inputData.Message
	messageType := inputData.MessageType
	fontSize := inputData.FontSize
	textColor := inputData.TextColor

	id = createNewImageId()

	// Create a new image
	fSize, intErr := strconv.ParseFloat(fontSize, 64)
	checkErrorAndPanic(intErr)

	image, err := createImage(id, to, from, text, messageType, fSize, textColor)
	checkErrorAndPanic(err)

	imageSrc := "/image/" + image.Id + "?t=" + time.Now().String()
	imageViewUrl := "http://desistickers.azurewebsites.net/view/" + image.Id

	response := CreateResponse{Id: image.Id, ImageSrc: imageSrc, ImageViewUrl: imageViewUrl}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)

	// Write back json response
	json.NewEncoder(w).Encode(response)

	elapsed := time.Since(start)

	insightsClient.TrackMetric("CreateHandler", float32(elapsed.Seconds()))
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var inputData DeleteRequest

	err := decoder.Decode(&inputData)
	checkErrorAndPanic(err)

	id := inputData.Id

	if id == "" {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
	}

	// Save the image on disk
	err = deleteImage(id)
	checkErrorAndPanic(err)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/view/"):]

	// Redirect to home page if the image was not found
	if id == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	image := &Image{Id: id}

	// Render the view page
	renderTemplate(w, image, "view.html")
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/image/"):]

	// Find the image and load if it exists on the server
	buffer, err := getImageData(id)

	// Redirect to home page if the image was not found
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
	}

	w.Header().Set("Content-Type", "image/jpg")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer)))

	if _, err := w.Write(buffer); err != nil {
		fmt.Println("unable to write image.")
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[1:])
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func messageTypesHandler(w http.ResponseWriter, r *http.Request) {

	response := []string{"Happy Birthday", "Happy Anniversary", "Festival", "Inspirational", "Relationship", "Quote", "Basic"}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	// Write back json response
	json.NewEncoder(w).Encode(response)
}

func fontSizesHandler(w http.ResponseWriter, r *http.Request) {

	response := []int{10, 12, 14, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	// Write back json response
	json.NewEncoder(w).Encode(response)
}

func textColorsHandler(w http.ResponseWriter, r *http.Request) {

	response := []string{"Black", "White", "Red", "Blue", "Yellow"}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	// Write back json response
	json.NewEncoder(w).Encode(response)
}

func main() {
	flag.Parse()

	port := os.Getenv("HTTP_PLATFORM_PORT")
	fmt.Println(port)

	insightsClient = appinsights.NewTelemetryClient("1a87e58b-3a57-4ddd-ad15-e5bc3c55694f")
	insightsClient.TrackEvent("ServerRunning")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/static/", staticHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/create/", createHandler)
	http.HandleFunc("/delete/", deleteHandler)

	http.HandleFunc("/MessageTypes", messageTypesHandler)
	http.HandleFunc("/FontSizes", fontSizesHandler)
	http.HandleFunc("/TextColors", textColorsHandler)

	http.ListenAndServe(":"+os.Getenv("HTTP_PLATFORM_PORT"), nil)
}
