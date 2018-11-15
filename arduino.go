package tooling

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
)

type ArduinoEnvironment struct {
	ToolsDirectory string
	Boards         *PropertiesMap
	Platform       *PropertiesMap
}

func NewArduinoEnvironment() (ae *ArduinoEnvironment) {
	return &ArduinoEnvironment{}
}

func (ae *ArduinoEnvironment) Locate(candidate string) error {
	dir, err := searchForTools(candidate)
	if err != nil {
		return err
	}

	ae.ToolsDirectory = dir

	boardsPath := path.Join(ae.ToolsDirectory, "boards.txt")
	boards, err := NewPropertiesMapFromFile(boardsPath)
	if err != nil {
		log.Fatalf("Error: Unable to open %s (%v)", boardsPath, err)
	}

	platformPath := path.Join(ae.ToolsDirectory, "platform.txt")
	platform, err := NewPropertiesMapFromFile(platformPath)
	if err != nil {
		log.Fatalf("Error: Unable to open %s (%v)", platformPath, err)
	}

	ae.Boards = boards
	ae.Platform = platform

	return nil
}

func searchForTools(candidate string) (string, error) {
	if candidate != "" {
		return candidate, nil
	}

	exec, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Dir(exec))
	if err != nil {
		log.Fatal(err)
	}

	home := os.Getenv("HOME")

	candidates := []string{
		filepath.Join(home, ".fk/tools"),
		filepath.Join(dir, "tools"),
		filepath.Join(filepath.Dir(dir), "lib/flasher"),
		filepath.Join(filepath.Dir(dir), "lib/fkflash"),
		"./tools",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			return p, nil
		}
	}

	return "", fmt.Errorf("Unable to find tools, looked in %v", candidates)
}
