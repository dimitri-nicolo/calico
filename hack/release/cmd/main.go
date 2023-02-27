package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"

	"github.com/projectcalico/calico/hack/release/pkg/builder"
)

var (
	create, publish, newBranch, meta bool
	dir                              string
	imgOverrides                     string
	versionsFile                     string
)

func init() {
	flag.BoolVar(&create, "create", false, "Create a release from the current commit")
	flag.BoolVar(&publish, "publish", false, "Publish the release built from the current tag")
	flag.BoolVar(&newBranch, "new-branch", false, "Create a new release branch from master")
	flag.BoolVar(&meta, "metadata", false, "Product release metadata")

	flag.StringVar(&dir, "dir", "./", "Directory to place build metadata in")

	// These flags are needed for metadata building in hashreleases, because some repos aren't in the monorepo yet and use different
	// versions for hashreleases.
	flag.StringVar(&imgOverrides, "img-overrides", "", "Comma-separated list of name+version overides. e.g., img-overrides=calico-node:4.16,kube-controllers:3.12")
	flag.StringVar(&versionsFile, "versions-file", "", "Path to a versions file from which to get image override information")

	flag.Parse()
}

func main() {
	// Create a releaseBuilder to use.
	r := builder.NewReleaseBuilder(&builder.RealCommandRunner{})

	if meta {
		overrides := strings.Split(imgOverrides, ",")
		if versionsFile != "" {
			overrides = loadVersionFile(versionsFile)
		}
		configureLogging("metadata.log")
		err := r.BuildMetadata(dir, overrides...)
		if err != nil {
			logrus.WithError(err).Error("Failed to produce release metadata")
			os.Exit(1)
		}
		return
	}

	if create {
		configureLogging("release-build.log")
		err := r.BuildRelease()
		if err != nil {
			logrus.WithError(err).Error("Failed to create Calico release")
			os.Exit(1)
		}
		return
	}

	if publish {
		configureLogging("release-publish.log")
		err := r.PublishRelease()
		if err != nil {
			logrus.WithError(err).Error("Failed to publish Calico release")
			os.Exit(1)
		}
		return
	}

	if newBranch {
		configureLogging("cut-release-branch.log")
		err := r.NewBranch()
		if err != nil {
			logrus.WithError(err).Error("Failed to create new release branch")
			os.Exit(1)
		}
		return
	}

	logrus.Fatalf("No command specified")
}

func configureLogging(filename string) {
	// Set up logging to both stdout as well as a file.
	writers := []io.Writer{os.Stdout, &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 10,
	}}
	logrus.SetOutput(io.MultiWriter(writers...))
}

// loads a versions yaml file and returns the images within, in name=version format.
func loadVersionFile(filename string) []string {
	imgs := []string{}
	bs, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	// Local helpers for unmarshalling yaml contents.
	type Version struct {
		Components map[string]interface{} `yaml:"components"`
	}
	type VersionsYAML []Version
	versions := VersionsYAML{}

	err = yaml.Unmarshal(bs, &versions)
	if err != nil {
		panic(err)
	}
	for _, c := range versions[0].Components {
		comp := c.(map[string]interface{})
		image := comp["image"]
		version := comp["version"]
		if image != nil && version != nil {
			image := strings.TrimPrefix(image.(string), "tigera/")
			imgs = append(imgs, fmt.Sprintf("%s=%s", image, version))
			logrus.Infof("Found image in versions file: %s:%s", image, version)
		}
	}

	return imgs
}
