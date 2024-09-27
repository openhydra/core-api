package privileges

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("privileges test", func() {
	Describe("SetFullAccess test", func() {
		It("should return a map of all modules with full access", func() {
			p := DefaultPrivilegeProvider{}
			result := p.SetFullAccess()
			Expect(result).To(Equal(map[string]uint64{
				"indexView":         3,
				"courseStudentView": 1,
				"model":             31,
				"role":              31,
				"group":             31,
				"course":            63,
				"dataset":           31,
				"user":              63,
				"setting":           31,
				"deviceStudentView": 31,
				"device":            63,
				"rag":               127,
				"flavor":            31,
			}))
		})
	})
	Describe("CanAccess test", func() {
		It("should return true if the user has the required permission", func() {
			p := DefaultPrivilegeProvider{}
			result, _ := p.CanAccess(map[string]uint64{
				"course":  31,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "course", 1)
			Expect(result).To(BeTrue())
		})

		It("should return false if the required permission over range", func() {
			p := DefaultPrivilegeProvider{}
			result, _ := p.CanAccess(map[string]uint64{
				"course":  31,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "course", 32)
			Expect(result).To(BeFalse())
		})

		It("should return false if the user does not have the required permission", func() {
			p := DefaultPrivilegeProvider{}
			result, _ := p.CanAccess(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "course", PermissionCourseCreate)
			Expect(result).To(BeFalse())
		})

		It("should be error due to no module found", func() {
			p := DefaultPrivilegeProvider{}
			_, err := p.CanAccess(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "test", PermissionCourseCreate)
			Expect(err).To(HaveOccurred())
		})
		It("should be access denied due to no permission found for group module", func() {
			p := DefaultPrivilegeProvider{}
			canAccess, err := p.CanAccess(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
				"group":   0,
			}, "group", 1)
			Expect(err).To(BeNil())
			Expect(canAccess).To(BeFalse())
		})
		It("should be access granted due to permission found for group module", func() {
			p := DefaultPrivilegeProvider{}
			canAccess, err := p.CanAccess(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
				"group":   31,
			}, "group", 1)
			Expect(err).To(BeNil())
			Expect(canAccess).To(BeTrue())
		})
	})
	Describe("ModulePermission test", func() {
		It("should return a map of all modules with their permissions", func() {
			p := DefaultPrivilegeProvider{}
			result, _ := p.ModulePermission(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "course")
			Expect(result).To(Equal(map[string]bool{
				"view_page":                  true,
				"list":                       true,
				"create":                     false,
				"update":                     false,
				"delete":                     false,
				"manage_other_user_resource": false,
				http.MethodPost:              false,
				http.MethodPut:               false,
				http.MethodPatch:             false,
				http.MethodDelete:            false,
				http.MethodGet:               true,
			}))
		})
		It("should be error due to no module found", func() {
			p := DefaultPrivilegeProvider{}
			_, err := p.ModulePermission(map[string]uint64{
				"course":  PermissionCourseViewPage | PermissionCourseList,
				"dataset": 31,
				"device":  31,
				"user":    31,
				"role":    31,
				"rag":     63,
			}, "test")
			Expect(err).To(HaveOccurred())
		})
	})
})
