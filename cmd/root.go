package cmd

import (
	"github.com/oid-explorer/api.oid-explorer.com/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func init() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	cobra.OnInitialize(initEnv)

	rootCMD.Flags().StringP("loglevel", "l", "error", "loglevel")
	rootCMD.Flags().StringP("datasourcename", "d", "", "data sourcename to connect to the SQL database")
	rootCMD.Flags().IntP("port", "p", 9000, "port of the API")

	err := viper.BindPFlag("loglevel", rootCMD.Flags().Lookup("loglevel"))
	if err != nil {
		log.Error().
			AnErr("Error", err).
			Msg("Can't bind flag loglevel")
		return
	}

	err = viper.BindPFlag("datasourcename", rootCMD.Flags().Lookup("datasourcename"))
	if err != nil {
		log.Error().
			AnErr("Error", err).
			Msg("Can't bind flag datasourcename")
		return
	}

	err = viper.BindPFlag("port", rootCMD.Flags().Lookup("port"))
	if err != nil {
		log.Error().
			AnErr("Error", err).
			Msg("Can't bind flag port")
		return
	}
}

func initEnv() {
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetEnvPrefix("API_OID_EXPLORER_COM")
	viper.AutomaticEnv()
}

var rootCMD = &cobra.Command{
	Use:   "api.oid-explorer.com",
	Short: "API implementation of api.oid-explorer.com",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		loglevel, err := zerolog.ParseLevel(viper.GetString("loglevel"))
		if err != nil {
			return errors.New("invalid loglevel set")
		}
		zerolog.SetGlobalLevel(loglevel)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		api.StartAPI()
	},
}

// Execute is the entrypoint for the CLI interface.
func Execute() {
	if err := rootCMD.Execute(); err != nil {
		os.Exit(1)
	}
}
