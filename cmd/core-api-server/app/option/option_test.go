package option

import (
	"core-api/cmd/core-api-server/app/config"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var _ = Describe("option test", func() {
	var option *Option
	BeforeEach(func() {
		option = &Option{}
		// create a config file in /tmp/core-api-server-config.yaml
		defaultConfig := config.DefaultConfig()
		defaultConfig.AuthConfig.Keystone.Username = ""
		defaultConfig.AuthConfig.Keystone.Password = "new-password"
		result, err := yaml.Marshal(defaultConfig)
		Expect(err).To(BeNil())
		// write to a file
		err = os.WriteFile("/tmp/core-api-server-config.yaml", result, 0644)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		serverConfig = nil
	})

	Describe("binding test", func() {
		It("when binding flags", func() {
			cmd := &cobra.Command{
				Use:  "test",
				Long: "test",
			}
			fs := cmd.PersistentFlags()
			option.BindFlags(fs)
			cmd.SetArgs([]string{"--config", "/tmp/core-api-server-config.yaml"})
			cmd.Execute()
			Expect(option.ConfigPath).To(Equal("/tmp/core-api-server-config.yaml"))
		})
	})

	Describe("generate config test", func() {
		It("should be expected", func() {
			option.ConfigPath = "/tmp/core-api-server-config.yaml"
			config, err := option.GenerateConfig(false)
			Expect(err).To(BeNil())
			// keep default value if specified value is empty in config file
			Expect(config.AuthConfig.Keystone.Username).To(Equal("admin"))
			// if not empty then we should use it
			Expect(config.AuthConfig.Keystone.Password).To(Equal("new-password"))
			Expect(config.CoreApiConfig.DisableAuth).To(BeFalse())
			Expect(config.CoreApiConfig.Port).To(Equal("8080"))
		})
		It("should be error with config file not exists", func() {
			option.ConfigPath = "/tmp/core-api-server-config-not-exists.yaml"
			_, err := option.GenerateConfig(false)
			Expect(err).NotTo(BeNil())
		})
	})
})
