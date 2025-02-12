package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/o3willard-AI/SSSonector/internal/errors"
	"github.com/o3willard-AI/SSSonector/internal/preflight"
)

var (
	configPath = flag.String("config", "/etc/sssonector/config.yaml", "Path to config file")
	mode       = flag.String("mode", "", "Operating mode (server/client)")
	fix        = flag.Bool("fix", false, "Attempt to fix issues automatically")
	format     = flag.String("format", "text", "Output format (text/json/markdown)")
	quiet      = flag.Bool("quiet", false, "Suppress non-error output")
	verbose    = flag.Bool("verbose", false, "Show detailed check information")
	checkOnly  = flag.String("check", "", "Run specific check only")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Initialize error handler
	errorHandler := errors.NewHandler(logger)
	registerErrorHandlers(errorHandler)

	// Create pre-flight options
	opts := &preflight.Options{
		ConfigPath: *configPath,
		Mode:       *mode,
		Fix:        *fix,
		Format:     *format,
		Quiet:      *quiet,
		Verbose:    *verbose,
		CheckOnly:  *checkOnly,
	}

	// Create and run pre-flight checker
	checker := preflight.NewChecker(opts, errorHandler)
	report, err := checker.RunChecks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pre-flight check failed: %v\n", err)
		os.Exit(1)
	}

	// Print report
	if !*quiet {
		if err := report.Print(*format); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print report: %v\n", err)
			os.Exit(1)
		}
	}

	// Exit with status code based on report
	if !report.Passed() {
		os.Exit(1)
	}
}

// registerErrorHandlers registers error handlers and fixes
func registerErrorHandlers(h *errors.Handler) {
	// Configuration errors
	h.RegisterFix(errors.ErrConfigSyntax, errors.Fix{
		Description: "Fix YAML syntax in configuration file",
		Command:     "yamllint -f parsable --fix /etc/sssonector/config.yaml",
		Automatic:   false,
		Risk:        "Low",
	})

	h.RegisterFix(errors.ErrConfigPermission, errors.Fix{
		Description: "Fix configuration file permissions",
		Command:     "sudo chmod 640 /etc/sssonector/config.yaml && sudo chown root:sssonector /etc/sssonector/config.yaml",
		Automatic:   true,
		Risk:        "Low",
	})

	// System errors
	h.RegisterFix(errors.ErrSystemCapability, errors.Fix{
		Description: "Add required capabilities to binary",
		Command:     "sudo setcap cap_net_admin,cap_net_raw+ep /usr/local/bin/sssonector",
		Automatic:   true,
		Risk:        "Medium",
	})

	h.RegisterFix(errors.ErrSystemGroup, errors.Fix{
		Description: "Add current user to sssonector group",
		Command:     "sudo usermod -a -G sssonector $USER",
		Automatic:   true,
		Risk:        "Low",
		Requires:    []string{"Session restart"},
	})

	// Resource errors
	h.RegisterFix(errors.ErrResourceBusy, errors.Fix{
		Description: "Kill processes using required resources",
		Command:     "sudo lsof -t -i :8443 | xargs kill",
		Automatic:   false,
		Risk:        "High",
		Requires:    []string{"Process verification"},
	})

	h.RegisterFix(errors.ErrResourcePermission, errors.Fix{
		Description: "Fix TUN device permissions",
		Command:     "sudo chmod 660 /dev/net/tun && sudo chown root:sssonector /dev/net/tun",
		Automatic:   true,
		Risk:        "Low",
	})

	// Network errors
	h.RegisterFix(errors.ErrNetworkPort, errors.Fix{
		Description: "Release port 8443",
		Command:     "sudo fuser -k 8443/tcp",
		Automatic:   false,
		Risk:        "High",
		Requires:    []string{"Process verification"},
	})

	h.RegisterFix(errors.ErrNetworkInterface, errors.Fix{
		Description: "Delete existing TUN interface",
		Command:     "sudo ip link delete tun0",
		Automatic:   true,
		Risk:        "Medium",
	})

	// Security errors
	h.RegisterFix(errors.ErrSecurityPermission, errors.Fix{
		Description: "Fix certificate permissions",
		Command:     "sudo chmod 600 /etc/sssonector/certs/*.key && sudo chmod 644 /etc/sssonector/certs/*.crt",
		Automatic:   true,
		Risk:        "Low",
	})
}
