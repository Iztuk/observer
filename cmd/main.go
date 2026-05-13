package main

import (
	"cf-observer/internal/audit"
	"cf-observer/internal/config"
	"cf-observer/internal/daemon"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printRootUsage()
		os.Exit(1)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		printRootUsage()
		return
	}

	cmdArg := os.Args[1]
	switch cmdArg {
	// NOTE: Run the daemon in the background when the project is ready for deployment
	case "init":
		initCmd := flag.NewFlagSet("init", flag.ExitOnError)
		resetConfig := initCmd.Bool("reset-config", false, "Overwrite existing config file")
		resetDB := initCmd.Bool("reset-db", false, "Delete and recreate the database")
		_ = initCmd.Parse(os.Args[2:])

		if !*resetConfig && !*resetDB {
			err := config.InitConfigDir(*resetConfig)
			if err != nil {
				log.Fatal(err)
			}

			configDir, err := config.ConfigDir()
			if err != nil {
				log.Fatal(err)
			}

			dbPath := configDir + "/cf-observer.db"
			err = audit.InitDatabase(dbPath, *resetDB)
			if err != nil {
				log.Fatal(err)
			}

		} else {
			if *resetConfig {
				err := config.InitConfigDir(*resetConfig)
				if err != nil {
					log.Fatal(err)
				}
			}

			configDir, err := config.ConfigDir()
			if err != nil {
				log.Fatal(err)
			}

			dbPath := configDir + "/cf-observer.db"
			if *resetDB {
				err = audit.InitDatabase(dbPath, *resetDB)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

	case "start":
		startCmd := flag.NewFlagSet("start", flag.ExitOnError)
		_ = startCmd.Parse(os.Args[2:])

		hosts, err := config.LoadConfigFile()
		if err != nil {
			log.Fatal(err)
		}

		err = daemon.RunDaemon(hosts)
		if err != nil {
			log.Fatal(err)
		}
	case "stop":
	default:
		printRootUsage()
		os.Exit(1)
	}

}

func printRootUsage() {
	fmt.Println("Usage: cf-observer <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init      Initialize the config directory and default configuration")
	fmt.Println("  start     Start the observer daemon")
	fmt.Println("  stop      Stop the observer daemon")
	fmt.Println()
	fmt.Println("Run 'cf-observer <command> -h' for command-specific help")
}
