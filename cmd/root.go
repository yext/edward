package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
	"github.com/yext/edward/output"
	"github.com/yext/edward/services"
	"github.com/yext/edward/updates"
)

var cfgFile string

var edwardClient *edward.Client

var checkUpdateChan chan interface{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "edward",
	Short: "A tool for managing local instances of microservices",
	Long: `Edward is a tool to simplify your microservices development workflow.
Build, start and manage service instances with a single command.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		var dirConfig, err = home.NewConfiguration(edwardHome)
		if err != nil {
			return errors.WithStack(err)
		}

		prefix := "edward"
		if len(args) > 0 {
			prefix = fmt.Sprintf("%s %s", cmd.Use, args[0])
		}

		if redirectLogs {
			log.SetOutput(os.Stdout)
			log.SetPrefix(fmt.Sprintf("%v >", prefix))
			log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		} else {
			log.SetOutput(&lumberjack.Logger{
				Filename:   path.Join(dirConfig.EdwardLogDir, "edward.log"),
				MaxSize:    50, // megabytes
				MaxBackups: 30,
				MaxAge:     1, //days
			})
			log.SetPrefix(fmt.Sprintf("%s > ", prefix))
			log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		}

		log.SetOutput(&lumberjack.Logger{
			Filename:   path.Join(dirConfig.EdwardLogDir, "edward.log"),
			MaxSize:    50, // megabytes
			MaxBackups: 30,
			MaxAge:     1, //days
		})
		log.SetPrefix(fmt.Sprintf("%s > ", prefix))
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

		// Begin logging
		log.Printf("=== Edward v%v ===\n", common.EdwardVersion)
		log.Printf("Args: %v\n", os.Args)
		// Recover from any panic and log appropriately
		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovered from panic:", r)
			}
			log.Printf("=== Exiting ===\n")
		}()

		// Set the default config path
		if configPath == "" {
			var err error
			tmpDirCfg := &home.EdwardConfiguration{}
			err = tmpDirCfg.Initialize()
			if err != nil {
				return errors.WithStack(err)
			}
			configPath, err = config.GetConfigPathFromWorkingDirectory(tmpDirCfg.Dir)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		command := cmd.Use

		if command != "generate" {
			edwardClient, err = edward.NewClientWithConfig(configPath, common.EdwardVersion)
			if err != nil {
				return errors.WithStack(err)
			}
			err = os.Chdir(edwardClient.BasePath())
			if err != nil {
				return errors.WithStack(err)
			}
		} else {
			edwardClient, err = edward.NewClient()
			if err != nil {
				return errors.WithStack(err)
			}
		}

		edwardClient.DirConfig = dirConfig
		edwardClient.Backends, err = buildBackendOverrides()
		if err != nil {
			return errors.WithStack(err)
		}

		// Set service checks to restart the client on sudo as needed
		edwardClient.ServiceChecks = func(sgs []services.ServiceOrGroup) error {
			return errors.WithStack(sudoIfNeeded(sgs))
		}
		// Populate the Edward executable with this binary
		edwardClient.EdwardExecutable = os.Args[0]

		// Let the client know about the log file for starting runners
		edwardClient.LogFile = logFile

		if redirectLogs {
			edwardClient.Follower = output.NewNonLiveFollower()
		}

		if command != "stop" {
			// Check for legacy pidfiles and error out if any are found
			for _, service := range edwardClient.ServiceMap() {
				if _, err := os.Stat(service.GetPidPathLegacy(edwardClient.DirConfig.PidDir)); !os.IsNotExist(err) {
					return errors.New("one or more services were started with an older version of Edward. Please run `edward stop` to stop these instances")
				}
			}
		}

		if command != "run" {
			checkUpdateChan = make(chan interface{})
			go checkUpdateAvailable(edwardClient.DirConfig.Dir, checkUpdateChan)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if checkUpdateChan != nil { //&& !didAutoComplete { // TODO: skip when autocompleting
			updateAvailable, ok := (<-checkUpdateChan).(bool)
			if ok && updateAvailable {
				latestVersion := (<-checkUpdateChan).(string)
				fmt.Printf("A new version of Edward is available (%v), update with:\n\tgo get -u github.com/yext/edward\n", latestVersion)
			}
		}
	},
}

func buildBackendOverrides() (map[string]string, error) {
	var overrides = make(map[string]string)
	for _, backend := range backends {
		separated := strings.Split(backend, ":")
		if len(separated) != 2 {
			return nil, errors.New("backend definition should be of the form '<service>:<backend>'")
		}
		if _, exists := overrides[separated[0]]; exists {
			return nil, fmt.Errorf("multiple backend selections specified for service or group: %s", separated[0])
		}
		overrides[separated[0]] = separated[1]
	}
	return overrides, nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	defaultHome := &home.EdwardConfiguration{}
	err := defaultHome.Initialize()
	if err != nil {
		panic(err)
	}

	logPrefix := "edward"
	if len(os.Args) > 1 {
		logPrefix = fmt.Sprintf("edward %v >", os.Args[1:])
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join(defaultHome.EdwardLogDir, "edward.log"),
		MaxSize:    50, // megabytes
		MaxBackups: 30,
		MaxAge:     1, //days
	})
	log.SetPrefix(logPrefix)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	for _, arg := range os.Args {
		if arg == "--generate-bash-completion" {
			autocompleteServicesAndGroups(defaultHome.Dir)
			return
		}
	}

	if err := RootCmd.Execute(); err != nil {
		log.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

var configPath string
var redirectLogs bool
var logFile string
var edwardHome string
var backends []string

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&logFile, "logfile", "", "Write logs to `PATH`")
	RootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Use service configuration file at `PATH`")
	RootCmd.PersistentFlags().BoolVar(&redirectLogs, "redirect_logs", false, "Redirect edward logs to the console")
	RootCmd.PersistentFlags().StringVar(&edwardHome, "edward_home", "", "")
	RootCmd.PersistentFlags().StringSliceVarP(&backends, "backend", "b", nil, "Choose a specific backend for a service or group, of the form '<service>:<backend name>'")
	err := RootCmd.PersistentFlags().MarkHidden("redirect_logs")
	if err != nil {
		panic(err)
	}
	err = RootCmd.PersistentFlags().MarkHidden("logfile")
	if err != nil {
		panic(err)
	}
	err = RootCmd.PersistentFlags().MarkHidden("edward_home")
	if err != nil {
		panic(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println("initConfig: error finding home dir:", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra-start" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cobra-start")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func checkUpdateAvailable(homeDir string, checkUpdateChan chan interface{}) {
	defer close(checkUpdateChan)
	updateAvailable, latestVersion, err := updates.UpdateAvailable("yext", "edward", common.EdwardVersion, filepath.Join(homeDir, ".cache/version"))
	if err != nil {
		log.Println("Error checking for updates:", err)
		return
	}

	checkUpdateChan <- updateAvailable
	if updateAvailable {
		checkUpdateChan <- latestVersion
	}
}
