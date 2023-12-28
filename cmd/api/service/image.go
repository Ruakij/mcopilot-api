package service

import (
	"log"
	"os"
	"time"

	"git.ruekov.eu/ruakij/mcopilot-api/lib/advancedmap"
)

type ImageService struct {
	ImageStore *advancedmap.AdvancedMap[string, []byte]
}

var ImageServiceSingleton *ImageService

// Init creates a new AdvancedMap with a 30-minute time limit and loads the files from the given path
func (service *ImageService) Init(path string) *ImageService {
	service.ImageStore = advancedmap.NewAdvancedMap[string, []byte](30*time.Minute, 0)

	// Create the directory if it does not exist
	err := os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Load from disk
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if !file.IsDir() {
			// Read the file content as bytes
			data, err := os.ReadFile(path + "/" + file.Name())
			if err != nil {
				log.Println(err)
				continue
			}

			// Get the file info
			info, err := file.Info()
			if err != nil {
				log.Println(err)
				continue
			}

			// Check if the file was accessed within the time limit
			if time.Since(info.ModTime()) <= service.ImageStore.TimeLimit {
				// Store the file content with the file name as the key
				service.ImageStore.Put(file.Name(), data)
			}
		}
	}

	return service
}

// GetImage returns the image data from the ImageStore with the given key
// If the image is not in the ImageStore, it tries to load it from the disk
func (service *ImageService) GetImage(key string) ([]byte, bool) {
	// Check if the image is in the ImageStore
	data, ok := service.ImageStore.Get(key)
	if ok {
		return data, ok
	}

	// Try to load the image from the disk
	data, err := os.ReadFile(key)
	if err != nil {
		return nil, false
	}

	// Store the image in the ImageStore
	service.ImageStore.Put(key, data)

	// Update the file access time
	err = os.Chtimes(key, time.Now(), time.Now())
	if err != nil {
		log.Println(err)
	}

	return data, true
}

// PutImage adds a new image to the ImageStore with the given key and data
// and writes the data to the disk
func (service *ImageService) PutImage(key string, data []byte) error {
	// Store the image in the ImageStore
	service.ImageStore.Put(key, data)

	// Write the data to the disk
	err := os.WriteFile(key, data, 0644)
	if err != nil {
		return err
	}

	// Update the file access time
	err = os.Chtimes(key, time.Now(), time.Now())
	if err != nil {
		log.Println(err)
	}

	return nil
}

// DeleteImage removes an image from the ImageStore with the given key
// and deletes the file from the disk
func (service *ImageService) DeleteImage(key string) {
	// Remove the image from the ImageStore
	service.ImageStore.Remove(key)

	// Delete the file from the disk
	os.Remove(key)
}
