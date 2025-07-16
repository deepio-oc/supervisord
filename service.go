package main

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceCommand install/uninstall/start/stop supervisord service
type ServiceCommand struct {
}

var serviceCommand ServiceCommand

type supervisordService struct{}

func (s *supervisordService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	
	status <- svc.Status{State: svc.StartPending}
	
	// Start supervisord
	done := make(chan bool)
	go s.run(done)
	
	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				done <- true
				return false, 0
			}
		}
	}
}

func (s *supervisordService) run(done chan bool) {
	// Change to the directory where the executable is located
	// This ensures supervisord can find its config file
	if exePath, err := os.Executable(); err == nil {
		if exeDir := filepath.Dir(exePath); exeDir != "" {
			os.Chdir(exeDir)
		}
	}
	
	// Start supervisord in a separate goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Supervisord service panicked: %v", r)
			}
		}()
		runServer()
	}()
	
	// Wait for stop signal
	<-done
}

// Execute implement Execute() method defined in flags.Commander interface, executes the given command
func (sc ServiceCommand) Execute(args []string) error {
	if len(args) == 0 {
		showUsage()
		return nil
	}

	action := args[0]
	switch action {
	case "install":
		return installService()
	case "uninstall":
		return uninstallService()
	case "start":
		return startService()
	case "stop":
		return stopService()

	default:
		showUsage()
	}

	return nil
}

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		log.Error("Failed to get executable path: ", err)
		fmt.Println("Failed to get executable path: ", err)
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		log.Error("Failed to connect to service manager: ", err)
		fmt.Println("Failed to connect to service manager: ", err)
		return err
	}
	defer m.Disconnect()

	// Prepare service arguments
	serviceArgs := []string{}
	if options.Configuration != "" {
		serviceArgs = append(serviceArgs, "--configuration="+options.Configuration)
	}
	if options.EnvFile != "" {
		serviceArgs = append(serviceArgs, "--env-file="+options.EnvFile)
	}

	s, err := m.CreateService("go-supervisord", exePath, mgr.Config{
		DisplayName: "go-supervisord",
		Description: "Supervisord service in golang",
	}, serviceArgs...)
	if err != nil {
		log.Error("Failed to install service go-supervisord: ", err)
		fmt.Println("Failed to install service go-supervisord: ", err)
		return err
	}
	defer s.Close()

	fmt.Println("Succeed to install service go-supervisord")
	return nil
}

func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		log.Error("Failed to connect to service manager: ", err)
		fmt.Println("Failed to connect to service manager: ", err)
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService("go-supervisord")
	if err != nil {
		log.Error("Failed to open service go-supervisord: ", err)
		fmt.Println("Failed to open service go-supervisord: ", err)
		return err
	}
	defer s.Close()

	// Stop service if running
	s.Control(svc.Stop)

	err = s.Delete()
	if err != nil {
		log.Error("Failed to uninstall service go-supervisord: ", err)
		fmt.Println("Failed to uninstall service go-supervisord: ", err)
		return err
	}

	fmt.Println("Succeed to uninstall service go-supervisord")
	return nil
}

func startService() error {
	m, err := mgr.Connect()
	if err != nil {
		log.Error("Failed to connect to service manager: ", err)
		fmt.Println("Failed to connect to service manager: ", err)
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService("go-supervisord")
	if err != nil {
		log.Error("Failed to open service go-supervisord: ", err)
		fmt.Println("Failed to open service go-supervisord: ", err)
		return err
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		log.Error("Failed to start service: ", err)
		fmt.Println("Failed to start service: ", err)
		return err
	}

	fmt.Println("Succeed to start service go-supervisord")
	return nil
}

func stopService() error {
	m, err := mgr.Connect()
	if err != nil {
		log.Error("Failed to connect to service manager: ", err)
		fmt.Println("Failed to connect to service manager: ", err)
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService("go-supervisord")
	if err != nil {
		log.Error("Failed to open service go-supervisord: ", err)
		fmt.Println("Failed to open service go-supervisord: ", err)
		return err
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		log.Error("Failed to stop service: ", err)
		fmt.Println("Failed to stop service: ", err)
		return err
	}

	fmt.Println("Succeed to stop service go-supervisord")
	return nil
}



func showUsage() {
	fmt.Println("usage: supervisord service install/uninstall/start/stop")
}

func init() {
	parser.AddCommand("service",
		"install/uninstall/start/stop service",
		"install/uninstall/start/stop service",
		&serviceCommand)
}
