package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mod/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	r := gin.Default()

	// Configuração do middleware CORS
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	db.InitDB()

	r.POST("/person", addPerson)
	r.GET("/person/:id", getPerson)
	r.GET("/person/all", addPersonAll)
	r.POST("/person/:id/video", addVideo)
	r.GET("/person/:id/videos", listVideos)
	r.GET("/videos/:id/:filename", getVideo)
	r.GET("/person/:id/image", getImage) // Nova rota para obter a imagem
	r.GET("/videos/all", listAllVideos)

	r.Run(":8580")
}

func addPerson(c *gin.Context) {
	name := c.PostForm("name")
	image, err := c.FormFile("image")

	if name == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and image are required"})
		return
	}

	person := db.Person{Name: name}
	result, err := db.AddPerson(&person)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add person"})
		return
	}

	id := result.InsertedID.(primitive.ObjectID).Hex()
	personFolder := filepath.Join("uploads", id)
	os.MkdirAll(personFolder, os.ModePerm)

	filename := filepath.Base(image.Filename)
	imagePath := filepath.Join(personFolder, filename)
	c.SaveUploadedFile(image, imagePath)

	update := bson.M{"image_path": imagePath}
	_, err = db.UpdatePerson(id, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update person"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "name": person.Name, "image_path": imagePath})
}

func getPerson(c *gin.Context) {
	id := c.Param("id")
	person, err := db.GetPersonByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": person.ID, "name": person.Name, "image_path": person.ImagePath})
}

func addVideo(c *gin.Context) {
	id := c.Param("id")
	person, err := db.GetPersonByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	video, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video is required"})
		return
	}

	videoFolder := filepath.Join("videos", id)
	err = os.MkdirAll(videoFolder, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create directory"})
		return
	}

	videoFiles, err := os.ReadDir(videoFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read directory"})
		return
	}
	videoNumber := len(videoFiles) + 1

	// Salvar o arquivo de vídeo com áudio
	videoFilename := fmt.Sprintf("%d%s", videoNumber, ".mp4")
	videoPath := filepath.Join(videoFolder, videoFilename)
	err = c.SaveUploadedFile(video, videoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save video"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         person.ID,
		"video_path": videoPath,
		"filename":   videoFilename,
	})
}

func listVideos(c *gin.Context) {
	id := c.Param("id")
	person, err := db.GetPersonByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}

	videoFolder := filepath.Join("videos", id)
	if _, err := os.Stat(videoFolder); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No videos found"})
		return
	}

	videoFiles, _ := os.ReadDir(videoFolder)
	var videoUrls []string
	for _, file := range videoFiles {
		videoUrls = append(videoUrls, fmt.Sprintf("%s/videos/%s/%s", c.Request.Host, id, file.Name()))
	}

	c.JSON(http.StatusOK, gin.H{"id": person.ID, "videos": videoUrls})
}

func getVideo(c *gin.Context) {
	id := c.Param("id")
	filename := c.Param("filename")
	videoFolder := filepath.Join("videos", id)
	videoPath := filepath.Join(videoFolder, filename)
	c.File(videoPath)
}

func getImage(c *gin.Context) {
	id := c.Param("id")

	imageFolder := filepath.Join("uploads", id)
	imageFiles, err := os.ReadDir(imageFolder)
	if err != nil || len(imageFiles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	imagePath := filepath.Join(imageFolder, imageFiles[0].Name())
	c.File(imagePath)
}

func addPersonAll(c *gin.Context) {
	data, err := db.GetPersonAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	fmt.Println(data)
	c.JSON(http.StatusOK, data)
}

func listAllVideos(c *gin.Context) {
	videoRootFolder := "videos"

	if _, err := os.Stat(videoRootFolder); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No videos found"})
		return
	}

	allVideos := make(map[string][]string)
	persons, err := os.ReadDir(videoRootFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read directory"})
		return
	}

	for _, person := range persons {
		if !person.IsDir() {
			continue
		}
		personID := person.Name()
		personVideoFolder := filepath.Join(videoRootFolder, personID)

		videoFiles, err := os.ReadDir(personVideoFolder)
		if err != nil {
			continue
		}

		var videoUrls []string
		for _, file := range videoFiles {
			videoUrls = append(videoUrls, fmt.Sprintf("%s/videos/%s/%s", c.Request.Host, personID, file.Name()))
		}
		if len(videoUrls) > 0 {
			allVideos[personID] = videoUrls
		}
	}

	c.JSON(http.StatusOK, gin.H{"videos": allVideos})
}
