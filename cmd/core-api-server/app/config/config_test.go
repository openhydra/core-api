package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("DefaultConfig", func() {
		It("should be expected", func() {
			config := DefaultConfig()
			Expect(config.AuthConfig.Keystone.Endpoint).To(Equal("http://localhost:5000"))
			Expect(config.AuthConfig.Keystone.Username).To(Equal("admin"))
			Expect(config.AuthConfig.Keystone.Password).To(Equal("password"))
			Expect(config.AuthConfig.Keystone.DomainId).To(Equal("default"))
			Expect(config.AuthConfig.Keystone.ProjectId).To(Equal("default"))
			Expect(config.AuthConfig.Keystone.TokenKeyInResponse).To(Equal("X-Subject-Token"))
			Expect(config.AuthConfig.Keystone.TokenKeyInRequest).To(Equal("X-Auth-Token"))
		})
	})
})
