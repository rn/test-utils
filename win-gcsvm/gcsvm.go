package main

import (
	"flag"

	opengcs "github.com/Microsoft/opengcs/client"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	imgPath := flag.String("dir", "C:\\Program Files\\Linux Containers", "Directory with initrd.img and bootx64.efi")
	name := flag.String("name", "", "Name of the VM (default a UUID v4)")
	cmdLine := flag.String("cmdline", "console=ttyS0", "Kernel command line arguments")

	flag.Parse()

	if *name == "" {
		*name = uuid.NewV4().String()
	}

	logrus.SetLevel(logrus.DebugLevel)

	opts := []string{
		"lcow.kirdpath=" + *imgPath,
		"lcow.bootparameters=" + *cmdLine,
	}
	cfg := &opengcs.Config{}
	if err := cfg.GenerateDefault(opts); err != nil {
		logrus.Fatalf("GenerateDefault() failed: %v", err)
	}
	cfg.Name = *name

	if err := cfg.Validate(); err != nil {
		logrus.Fatalf("Validate() failed: %v", err)
	}

	logrus.Infof("Create VM: %s", *name)
	if err := cfg.StartUtilityVM(); err != nil {
		logrus.Fatalf("StartUtilityVM() failed: %v", err)
	}
}
