package gallery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/mudler/LocalAI/core/config"
	"github.com/mudler/LocalAI/pkg/downloader"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

// Installs a model from the gallery
func InstallModelFromGallery(galleries []config.Gallery, name string, basePath string, req GalleryModel, downloadStatus func(string, string, string, float64), enforceScan bool) error {

	applyModel := func(model *GalleryModel) error {
		name = strings.ReplaceAll(name, string(os.PathSeparator), "__")

		var config Config

		if len(model.URL) > 0 {
			var err error
			config, err = GetGalleryConfigFromURL(model.URL, basePath)
			if err != nil {
				return err
			}
			config.Description = model.Description
			config.License = model.License
		} else if len(model.ConfigFile) > 0 {
			// TODO: is this worse than using the override method with a blank cfg yaml?
			reYamlConfig, err := yaml.Marshal(model.ConfigFile)
			if err != nil {
				return err
			}
			config = Config{
				ConfigFile:  string(reYamlConfig),
				Description: model.Description,
				License:     model.License,
				URLs:        model.URLs,
				Name:        model.Name,
				Files:       make([]File, 0), // Real values get added below, must be blank
				// Prompt Template Skipped for now - I expect in this mode that they will be delivered as files.
			}
		} else {
			return fmt.Errorf("invalid gallery model %+v", model)
		}

		installName := model.Name
		if req.Name != "" {
			installName = req.Name
		}

		// Copy the model configuration from the request schema
		config.URLs = append(config.URLs, model.URLs...)
		config.Icon = model.Icon
		config.Files = append(config.Files, req.AdditionalFiles...)
		config.Files = append(config.Files, model.AdditionalFiles...)

		// TODO model.Overrides could be merged with user overrides (not defined yet)
		if err := mergo.Merge(&model.Overrides, req.Overrides, mergo.WithOverride); err != nil {
			return err
		}

		if err := InstallModel(basePath, installName, &config, model.Overrides, downloadStatus, enforceScan); err != nil {
			return err
		}

		return nil
	}

	models, err := AvailableGalleryModels(galleries, basePath)
	if err != nil {
		return err
	}

	model := FindGalleryElement(models, name, basePath)
	if model == nil {
		return fmt.Errorf("no model found with name %q", name)
	}

	return applyModel(model)
}

type GalleryElement interface {
	SetGallery(gallery config.Gallery)
	SetInstalled(installed bool)
	GetName() string
	GetGallery() config.Gallery
}

func FindGalleryElement[T GalleryElement](models []T, name string, basePath string) T {
	var model T
	name = strings.ReplaceAll(name, string(os.PathSeparator), "__")

	if !strings.Contains(name, "@") {
		for _, m := range models {
			if strings.EqualFold(m.GetName(), name) {
				model = m
				break
			}
		}

	} else {
		for _, m := range models {
			if strings.EqualFold(name, fmt.Sprintf("%s@%s", m.GetGallery().Name, m.GetName())) {
				model = m
				break
			}
		}
	}

	return model
}

// List available models
// Models galleries are a list of yaml files that are hosted on a remote server (for example github).
// Each yaml file contains a list of models that can be downloaded and optionally overrides to define a new model setting.
func AvailableGalleryModels(galleries []config.Gallery, basePath string) (GalleryModels, error) {
	var models []*GalleryModel

	// Get models from galleries
	for _, gallery := range galleries {
		galleryModels, err := getGalleryModels[*GalleryModel](gallery, basePath)
		if err != nil {
			return nil, err
		}
		models = append(models, galleryModels...)
	}

	return models, nil
}

// List available backends
func AvailableBackends(galleries []config.Gallery, basePath string) (GalleryBackends, error) {
	var models []*GalleryBackend

	// Get models from galleries
	for _, gallery := range galleries {
		galleryModels, err := getGalleryModels[*GalleryBackend](gallery, basePath)
		if err != nil {
			return nil, err
		}
		models = append(models, galleryModels...)
	}

	return models, nil
}

func findGalleryURLFromReferenceURL(url string, basePath string) (string, error) {
	var refFile string
	uri := downloader.URI(url)
	err := uri.DownloadWithCallback(basePath, func(url string, d []byte) error {
		refFile = string(d)
		if len(refFile) == 0 {
			return fmt.Errorf("invalid reference file at url %s: %s", url, d)
		}
		cutPoint := strings.LastIndex(url, "/")
		refFile = url[:cutPoint+1] + refFile
		return nil
	})
	return refFile, err
}

func getGalleryModels[T GalleryElement](gallery config.Gallery, basePath string) ([]T, error) {
	var models []T = []T{}

	if strings.HasSuffix(gallery.URL, ".ref") {
		var err error
		gallery.URL, err = findGalleryURLFromReferenceURL(gallery.URL, basePath)
		if err != nil {
			return models, err
		}
	}
	uri := downloader.URI(gallery.URL)

	err := uri.DownloadWithCallback(basePath, func(url string, d []byte) error {
		return yaml.Unmarshal(d, &models)
	})
	if err != nil {
		if yamlErr, ok := err.(*yaml.TypeError); ok {
			log.Debug().Msgf("YAML errors: %s\n\nwreckage of models: %+v", strings.Join(yamlErr.Errors, "\n"), models)
		}
		return models, err
	}

	// Add gallery to models
	for _, model := range models {
		model.SetGallery(gallery)
		// we check if the model was already installed by checking if the config file exists
		// TODO: (what to do if the model doesn't install a config file?)
		if _, err := os.Stat(filepath.Join(basePath, fmt.Sprintf("%s.yaml", model.GetName()))); err == nil {
			model.SetInstalled(true)
		}
	}
	return models, nil
}
