package internal

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/wesovilabs/beyond/helper"
	"github.com/wesovilabs/beyond/logger"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	defaultTargetDir      = ".beyond"
	defaultBeyondSettings = "beyond.toml"
)

func load(settingsPath string) *Settings {
	settings := Settings{}

	if _, err := os.Stat(settingsPath); err == nil {
		if _, err := toml.DecodeFile(settingsPath, &settings); err != nil {
			logger.Errorf(err.Error())
		}
	}

	return &settings
}

// Settings Beyond settings
type Settings struct {
	Path        string
	Project     string
	Output      string
	Pkg         string
	Excludes    []string
	ExcludeDirs map[string]bool
	Verbose     bool
	Work        bool
}

// BeyondSettingFromCommandLine returns the BeyondSettings from the command line args
func BeyondSettingFromCommandLine(args []string) *Settings {
	var path, project, outputDir, pkg, settingsPath string

	pwd, err := os.Getwd()

	if err != nil {
		panic(err)
	}

	var verbose, work bool

	flag.StringVar(&project, "project", "", "project name")
	flag.StringVar(&path, "path", pwd, "path")
	flag.StringVar(&settingsPath, "config", filepath.Join(path, defaultBeyondSettings), "beyond.tml path")
	flag.StringVar(&outputDir, "output", "", "output directory")
	flag.StringVar(&pkg, "package", "", "relative path to the main package")
	flag.BoolVar(&verbose, "verbose", false, "print info level logs to stdout")
	flag.BoolVar(&work, "work", false, "print the name of the temporary work directory and do not delete it when exiting")
	flag.Parse()

	settings := load(settingsPath)

	settings.updateWithFlags(args, project, path, outputDir, pkg, verbose, work)

	return settings
}

func takePackage(args []string) string {
	for i := range args {
		arg := args[i]
		if arg == "build" || arg == "generate" || arg == "run" {
			if len(args) > i+1 {
				file := args[i+1]
				return filepath.Dir(file)
			}
		}
	}

	return ""
}

func (settings *Settings) withProject(path, project string) {
	if project != "" {
		settings.Project = project
	} else if settings.Project == "" {
		if module, err := helper.GetModuleName(path); err == nil {
			settings.Project = module
		}
	}
}

func (settings *Settings) withOutputDir(path, outputDir string) {
	if outputDir != "" {
		settings.Output = outputDir
	}

	if settings.Output != "" {
		if outputDir, outErr := filepath.Abs(settings.Output); outErr != nil {
			settings.Output = filepath.Join(path, defaultTargetDir)
		} else {
			settings.Output = outputDir
		}
	} else {
		if targetDir, err := ioutil.TempDir("", "beyond"); err == nil {
			settings.Output = targetDir
		} else {
			settings.Output = filepath.Join(path, defaultTargetDir)
		}
	}
}

func (settings *Settings) withPkg(pkg string, args []string) {
	if pkg != "" {
		settings.Pkg = pkg
	}

	if settings.Pkg == "" {
		settings.Pkg = takePackage(args)
	}
}

func (settings *Settings) withWork(work bool) {
	if work && !settings.Work {
		settings.Work = true
	}
}

func (settings *Settings) withVerbose(verbose bool) {
	if verbose && !settings.Verbose {
		settings.Verbose = true
	}
}

func (settings *Settings) withExcludes() {
	settings.ExcludeDirs = map[string]bool{
		".git": true,
	}

	if settings.Excludes != nil {
		for i := range settings.Excludes {
			if absPath, err := filepath.Abs(settings.Excludes[i]); err == nil {
				settings.ExcludeDirs[absPath] = true
			}
		}
	}

	if outPath, err := filepath.Abs(settings.Output); err == nil {
		settings.ExcludeDirs[outPath] = true
	}
}

func (settings *Settings) updateWithFlags(args []string, project, path, outputDir, pkg string, verbose, work bool) {
	settings.withProject(path, project)
	settings.withOutputDir(path, outputDir)
	settings.Path = path
	settings.withPkg(pkg, args)
	settings.withWork(work)
	settings.withVerbose(verbose)
	settings.withExcludes()
}

// RemoveBeyondArguments removes beyond arguments from the list of arguments
func RemoveBeyondArguments(input []string) []string {
	out := make([]string, 0)
	argsIndex := make(map[int]bool)

	for i, arg := range input {
		switch arg {
		case "--config", "--project", "--output", "--path", "--package":
			argsIndex[i] = true
			argsIndex[i+1] = true
		case "--verbose", "--work":
			argsIndex[i] = true

			if len(input) > i+1 {
				if input[i+1] == "true" || input[i+1] == "false" {
					argsIndex[i+1] = true
				}
			}
		}
	}

	for i := range input {
		if !argsIndex[i] {
			out = append(out, input[i])
		}
	}

	return out
}
