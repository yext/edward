package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

var logger *log.Logger

var checkUpdateChan chan interface{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "edward",
	Short: "A tool for managing local instances of microservices",
	Long: `Edward is a tool to simplify your microservices development workflow.
Build, start and manage service instances with a single command.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if redirectLogs {
			logger = log.New(os.Stdout, fmt.Sprintf("%v >", os.Args), log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
		}
		// Begin logging
		logger.Printf("=== Edward v%v ===\n", common.EdwardVersion)
		logger.Printf("Args: %v\n", os.Args)

		// Set the default config path
		if configPath == "" {
			var err error
			configPath, err = config.GetConfigPathFromWorkingDirectory()
			if err != nil {
				return errors.WithStack(err)
			}
		}

		command := cmd.Use

		var err error
		if command != "generate" {
			edwardClient, err = edward.NewClientWithConfig(configPath, common.EdwardVersion, logger)
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

		// Set service checks to restart the client on sudo as needed
		edwardClient.ServiceChecks = func(sgs []services.ServiceOrGroup) error {
			return errors.WithStack(sudoIfNeeded(sgs))
		}
		edwardClient.Logger = logger
		// Populate the Edward executable with this binary
		edwardClient.EdwardExecutable = os.Args[0]

		if redirectLogs {
			edwardClient.Follower = output.NewNonLiveFollower()
		}

		if command != "stop" {
			// Check for legacy pidfiles and error out if any are found
			for _, service := range edwardClient.ServiceMap() {
				if _, err := os.Stat(service.GetPidPathLegacy()); !os.IsNotExist(err) {
					return errors.New("one or more services were started with an older version of Edward. Please run `edward stop` to stop these instances")
				}
			}
		}

		if command != "run" {
			checkUpdateChan = make(chan interface{})
			go checkUpdateAvailable(checkUpdateChan)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		defer logger.Printf("=== Exiting ===\n")
		if checkUpdateChan != nil { //&& !didAutoComplete { // TODO: skip when autocompleting
			updateAvailable, ok := (<-checkUpdateChan).(bool)
			if ok && updateAvailable {
				latestVersion := (<-checkUpdateChan).(string)
				fmt.Printf("A new version of Edward is available (%v), update with:\n\tgo get -u github.com/yext/edward\n", latestVersion)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	if err := home.EdwardConfig.Initialize(); err != nil {
		fmt.Printf("%+v", err)
	}

	logPrefix := "edward"
	if len(os.Args) > 1 {
		logPrefix = fmt.Sprintf("edward %v >", os.Args[1:])
	}
	logger = log.New(&lumberjack.Logger{
		Filename:   filepath.Join(home.EdwardConfig.EdwardLogDir, "edward.log"),
		MaxSize:    50, // megabytes
		MaxBackups: 30,
		MaxAge:     1, //days
	}, logPrefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	for _, arg := range os.Args {
		if arg == "--generate-bash-completion" {
			autocompleteServicesAndGroups(logger)
			return
		}
	}

	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var configPath string
var redirectLogs bool

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Use service configuration file at `PATH`")
	RootCmd.PersistentFlags().BoolVar(&redirectLogs, "redirect_logs", false, "Redirect edward logs to the console")
	err := RootCmd.PersistentFlags().MarkHidden("redirect_logs")
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
			fmt.Println(err)
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

func checkUpdateAvailable(checkUpdateChan chan interface{}) {
	defer close(checkUpdateChan)
	updateAvailable, latestVersion, err := updates.UpdateAvailable("yext", "edward", common.EdwardVersion, filepath.Join(home.EdwardConfig.Dir, ".cache/version"), logger)
	if err != nil {
		logger.Println("Error checking for updates:", err)
		return
	}

	checkUpdateChan <- updateAvailable
	if updateAvailable {
		checkUpdateChan <- latestVersion
	}
}
