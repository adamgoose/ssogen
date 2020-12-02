package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/adamgoose/ssogen/lib"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "ssogen {start-url}",
	Short: "Produces a valid ~/.aws/config file for your given SSO grants.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		region := viper.GetString("region")
		sess, err := session.NewSession(&aws.Config{
			Region: &region,
		})
		if err != nil {
			return err
		}

		c := &lib.Configurator{Session: sess}

		if err := c.RegisterClient(viper.GetString("client-name")); err != nil {
			return err
		}
		if err := c.StartDeviceAuthorization(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Login at %s\n", *c.Device.VerificationUriComplete)

		if err := <-c.WaitForToken(
			viper.GetDuration("poll-interval"),
			viper.GetDuration("poll-timeout"),
		); err != nil {
			return err
		}

		if err := c.LoadRoles(); err != nil {
			return err
		}

		c.WriteConfig(os.Stdout)

		return nil
	},
}

// Execute executes the command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	godotenv.Load()
	viper.AutomaticEnv()

	rootCmd.Flags().String("region", "us-east-2", "AWS Region to use for SSO and generated configuration")
	viper.BindPFlag("region", rootCmd.Flags().Lookup("region"))
	rootCmd.Flags().String("client-name", "sso-configurator", "Client name to use when registering with SSO OIDC")
	viper.BindPFlag("client-name", rootCmd.Flags().Lookup("client-name"))
	rootCmd.Flags().Duration("poll-interval", 5*time.Second, "Token Polling interval")
	viper.BindPFlag("poll-interval", rootCmd.Flags().Lookup("poll-interval"))
	rootCmd.Flags().Duration("poll-timeout", 5*time.Minute, "Token Polling timeout")
	viper.BindPFlag("poll-timeout", rootCmd.Flags().Lookup("poll-timeout"))
}
