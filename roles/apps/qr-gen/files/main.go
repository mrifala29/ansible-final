package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/gofiber/fiber/v2"

	qrcode "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

type DataRequest struct {
	StringQr string `json:"stringQr"`
	Produk   string `json:"produk"`
	Expire   string `json:"expire"`
}

const URL_main = "http://127.0.0.1:3000" // Ganti dengan URL server

func main() {
	app := fiber.New()

	app.Static("/public", "./public")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("server running!!!")
	})

	app.Post("/generate-qr", func(c *fiber.Ctx) error {

		req := new(DataRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Cannot parse JSON"})
		}

		if req.StringQr == "" {
			return c.Status(400).JSON(fiber.Map{"error": "StringQr cannot be empty"})
		}
		if req.Produk == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Produk cannot be empty"})
		}
		if req.Expire == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Expire cannot be empty"})
		}

		// ==== GENERATE QR + LOGO ====

		logoPath := "./logo_dana.png"
		tempQR := fmt.Sprintf("./public/temp_%d.png", time.Now().UnixNano())

		writer, err := standard.New(
			tempQR,
			standard.WithQRWidth(50),                // ukuran QR
			standard.WithBorderWidth(100),           // border
			standard.WithLogoImageFilePNG(logoPath), // logo
		)
		if err != nil {
			log.Println("Writer error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to init writer"})
		}

		qr, err := qrcode.New(req.StringQr)
		if err != nil {
			log.Println("QR error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create QR object"})
		}

		if err := qr.Save(writer); err != nil {
			log.Println("QR Save error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save QR image"})
		}

		// ==== LOAD FILE PNG ====

		qrImg, err := imaging.Open(tempQR) // Tetap gunakan kode Anda, ini sudah benar
		if err != nil {
			log.Println("imaging error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to open QR image"})
		}

		// ==== CANVAS (BAGIAN YANG DIUBAH) ====

		qrSize := qrImg.Bounds().Dx() // Lebar QR (misal 500px)

		// Tentukan tinggi untuk setiap bagian
		textHeightTop := 300 // Ruang untuk 'produk'

		// Ruang untuk 'Expire', tapi 0 jika 'Expire' kosong
		textHeightBottom := 20
		if req.Expire != "" {
			textHeightBottom = 180 // Beri ruang 150px jika ada
		}

		// Hitung total kanvas
		canvasW := qrSize
		canvasH := textHeightTop + qrSize + textHeightBottom // TINGGI TOTAL

		dc := gg.NewContext(canvasW, canvasH)
		dc.SetRGB(1, 1, 1) // Latar belakang putih
		dc.Clear()
		dc.SetRGB(0, 0, 0) // Warna teks hitam

		// 1. Gambar Teks ATAS (produk)
		if err := dc.LoadFontFace("./fonts/Poppins.ttf", 150); err == nil {
			// Y-pos: Di tengah blok atas (misal: 100/2 = 50)
			dc.DrawStringWrapped(
				req.Produk,
				float64(canvasW)/2,            // X: Tengah
				(float64(textHeightTop)/2)+50, // Y: Tengah blok atas
				0.5, 0.5,                      // Anchor di tengah
				float64(canvasW)-10,
				1.5,
				gg.AlignCenter,
			)
		}

		// 2. Gambar QR (DI TENGAH)
		// Y-pos: Tepat di bawah blok teks atas
		dc.DrawImage(qrImg, 0, textHeightTop)

		// 3. Gambar Teks BAWAH (Expire)
		if req.Expire != "" { // Hanya gambar jika ada
			if err := dc.LoadFontFace("./fonts/Poppins.ttf", 130); err == nil {
				// Y-pos: Di tengah blok bawah
				// (Mulai dari textHeightTop + qrSize, lalu ambil setengahnya textHeightBottom)
				yBottomCenter := (float64(textHeightTop+qrSize) + float64(textHeightBottom)/2) - 50
				dc.DrawStringWrapped(
					fmt.Sprintf("Berlaku Sampai %s", req.Expire),
					float64(canvasW)/2, // X: Tengah
					yBottomCenter,      // Y: Tengah blok bawah
					0.5, 0.5,           // Anchor di tengah
					float64(canvasW)-10,
					1.5,
					gg.AlignCenter,
				)
			}
		}

		// ==== SAVE FINAL ====

		filename := fmt.Sprintf("Payment_QR_%d.png", time.Now().UnixNano())
		finalPath := "./public/" + filename

		if err := dc.SavePNG(finalPath); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to save final image"})
		}

		// Hapus temp
		os.Remove(tempQR)

		url := URL_main + "/public/" + filename

		return c.JSON(fiber.Map{
			"message": "QR generated successfully!",
			"url":     url,
		})
	})

	app.Delete("/delete-qr/:filename", func(c *fiber.Ctx) error {
		filename := c.Params("filename")
		if filename == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Filename is required"})
		}

		filePath := "./public/" + filename
		if err := os.Remove(filePath); err != nil {
			log.Println("Delete error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to delete file"})
		}

		return c.JSON(fiber.Map{"message": "File deleted successfully"})
	})

	log.Println("Server running on :3000")
	log.Fatal(app.Listen(":3000"))
}
