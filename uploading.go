package tooling

import (
	"fmt"
	"log"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.bug.st/serial.v1"
)

type UploadOptions struct {
	Arduino     *ArduinoEnvironment
	Board       string
	Binary      string
	Port        string
	SkipTouch   bool
	FlashOffset int
	Verbose     bool
	Verify      bool
	Quietly     bool
}

func getPortsMap() map[string]bool {
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	m := make(map[string]bool)
	for _, p := range ports {
		m[p] = true
	}
	return m
}

func toKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func diffPortMaps(before map[string]bool, after map[string]bool) (added []string, removed []string) {
	added = make([]string, 0)
	removed = make([]string, 0)

	for p, _ := range after {
		if _, ok := before[p]; !ok {
			added = append(added, p)
		}
	}

	for p, _ := range before {
		if _, ok := after[p]; !ok {
			removed = append(removed, p)
		}
	}

	return
}

type PortDiscoverer struct {
	before map[string]bool
}

type Port struct {
	Name string
}

func NewPortDiscoveror() *PortDiscoverer {
	return &PortDiscoverer{
		before: getPortsMap(),
	}
}

func (pd *PortDiscoverer) List() []*Port {
	ports := make([]*Port, 0)

	for name, _ := range getPortsMap() {
		ports = append(ports, &Port{Name: name})
	}

	return ports
}

func (pd *PortDiscoverer) Discover() string {
	s := time.Now()

	for {
		after := getPortsMap()

		added, removed := diffPortMaps(pd.before, after)

		log.Printf("%v -> %v | %v %v\n", toKeys(pd.before), toKeys(after), removed, added)

		if len(added) > 0 {
			return added[0]
		}

		time.Sleep(500 * time.Millisecond)

		if time.Since(s) > 5*time.Second {
			break
		}

		pd.before = after
	}

	return ""
}

func listPorts() {
	ports := getPortsMap()
	log.Printf("Ports: %v", toKeys(ports))
}

func getPlatformKey() string {
	if runtime.GOOS == "darwin" {
		return "macosx"
	}
	return runtime.GOOS
}

func Upload(options *UploadOptions) error {
	board := options.Arduino.Boards.ToSubtree(options.Board)
	tools := options.Arduino.Platform.ToSubtree("tools")
	tool, _ := board.Lookup("upload.tool", make(map[string]string))

	// Force version 18 of bossac.
	if tool == "bossac" {
		tool = "bossac18"
	}

	u := board.Merge(tools.ToSubtree(tool))

	original := u.Properties["cmd"]
	if runtime.GOOS == "darwin" {
		u.Properties["cmd"] = original + "_osx"
	} else {
		if runtime.GOARCH == "arm" {
			u.Properties["cmd"] = original + "_linux_arm"
		} else {
			u.Properties["cmd"] = original + "_linux"
		}
	}

	port := options.Port
	if port == "" {
		port = NewPortDiscoveror().Discover()
		if port == "" {
			return fmt.Errorf("No port")
		}
	} else {
		use1200bpsTouch := board.ToBool("upload.use_1200bps_touch")

		if !options.SkipTouch && use1200bpsTouch {
			log.Printf("Performing 1200bps touch...")

			mode := &serial.Mode{
				BaudRate: 1200,
			}
			p, err := serial.Open(port, mode)
			if err != nil {
				listPorts()
				log.Fatalf("Error: Touch failed on (%s): %v", port, err)
			}
			p.SetDTR(false)
			p.SetRTS(true)
			p.Close()

			port = NewPortDiscoveror().Discover()
			if port == "" {
				if options.Port == "" {
					return fmt.Errorf("No port")
				}
				port = options.Port
			}
		}
	}

	log.Printf("Using port %s\n", port)

	if options.Verbose {
		u.Properties["upload.verbose"] = u.Properties["upload.params.verbose"]
	} else {
		u.Properties["upload.verbose"] = ""
	}
	if options.Verify {
		u.Properties["upload.verify"] = u.Properties["upload.params.verify"]
	} else {
		u.Properties["upload.verify"] = ""
	}
	u.Properties["upload.offset"] = fmt.Sprintf("%d", options.FlashOffset)
	u.Properties["runtime.tools.bossac-1.6.1-arduino.path"] = options.Arduino.ToolsDirectory
	u.Properties["runtime.tools.bossac-1.8.0-48-gb176eee.path"] = options.Arduino.ToolsDirectory
	u.Properties["serial.port.file"] = path.Base(port)
	u.Properties["build.path"] = path.Dir(options.Binary)
	u.Properties["build.project_name"] = strings.Replace(path.Base(options.Binary), path.Ext(options.Binary), "", -1)

	line, _ := options.Arduino.Platform.Lookup(fmt.Sprintf("tools.%s.upload.pattern", tool), u.Properties)

	log.Printf(line)

	if err := ExecuteAndPipeCommandLine(line, "upload | ", options.Quietly); err != nil {
		return err
	}

	return nil
}
