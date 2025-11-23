package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seaweedfs/seaweedfs/weed/admin/dash"
	"github.com/seaweedfs/seaweedfs/weed/admin/view/app"
	"github.com/seaweedfs/seaweedfs/weed/admin/view/layout"
	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/iam/integration"
	"github.com/seaweedfs/seaweedfs/weed/iam/policy"
)

// RoleHandlers contains all the HTTP handlers for role management
type RoleHandlers struct {
	adminServer *dash.AdminServer
}

// NewRoleHandlers creates a new instance of RoleHandlers
func NewRoleHandlers(adminServer *dash.AdminServer) *RoleHandlers {
	return &RoleHandlers{
		adminServer: adminServer,
	}
}

// ShowRoles renders the roles management page
func (h *RoleHandlers) ShowRoles(c *gin.Context) {
	// Get roles data from the server
	rolesData := h.getRolesData(c)

	// Render HTML template
	c.Header("Content-Type", "text/html")
	rolesComponent := app.Roles(rolesData)
	layoutComponent := layout.Layout(c, rolesComponent)
	err := layoutComponent.Render(c.Request.Context(), c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template: " + err.Error()})
		return
	}
}

// GetRoles returns the list of roles as JSON
func (h *RoleHandlers) GetRoles(c *gin.Context) {
	roles, err := h.adminServer.GetRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get roles: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// CreateRole handles role creation
func (h *RoleHandlers) CreateRole(c *gin.Context) {
	var req dash.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate role name
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role name is required"})
		return
	}

	// Check if role already exists
	existingRole, err := h.adminServer.GetRole(req.Name)
	
	if existingRole != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Role with this name already exists"})
		return
	}

	// Create role definition
	roleDef := &integration.RoleDefinition{
		RoleName:         req.Name,
		RoleArn:          req.RoleArn,
		TrustPolicy:      &req.TrustPolicy,
		AttachedPolicies: req.AttachedPolicies,
		Description:      req.Description,
	}

	// Set default ARN if not provided
	if roleDef.RoleArn == "" {
		roleDef.RoleArn = fmt.Sprintf("arn:aws:iam::role/%s", req.Name)
	}

	// Create the role
	err = h.adminServer.CreateRole(req.Name, roleDef)
	if err != nil {
		glog.Errorf("Failed to create role %s: %v", req.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Role created successfully",
		"role":    req.Name,
	})
}

// GetRole returns a specific role
func (h *RoleHandlers) GetRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role name is required"})
		return
	}

	role, err := h.adminServer.GetRole(roleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role: " + err.Error()})
		return
	}

	if role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, role)
}

// UpdateRole handles role updates
func (h *RoleHandlers) UpdateRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role name is required"})
		return
	}

	var req dash.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Check if role exists
	existingRole, err := h.adminServer.GetRole(roleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing role: " + err.Error()})
		return
	}
	if existingRole == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	// Create role definition with updated values
	roleDef := &integration.RoleDefinition{
		RoleName:         roleName,
		RoleArn:          existingRole.RoleArn,
		TrustPolicy:      &req.TrustPolicy,
		AttachedPolicies: req.AttachedPolicies,
		Description:      req.Description,
	}

	// Update the role
	err = h.adminServer.UpdateRole(roleName, roleDef)
	if err != nil {
		glog.Errorf("Failed to update role %s: %v", roleName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role updated successfully",
		"role":    roleName,
	})
}

// DeleteRole handles role deletion
func (h *RoleHandlers) DeleteRole(c *gin.Context) {
	roleName := c.Param("name")
	if roleName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role name is required"})
		return
	}

	// Check if role exists
	existingRole, err := h.adminServer.GetRole(roleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing role: " + err.Error()})
		return
	}
	if existingRole == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	// Delete the role
	err = h.adminServer.DeleteRole(roleName)
	if err != nil {
		glog.Errorf("Failed to delete role %s: %v", roleName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role deleted successfully",
		"role":    roleName,
	})
}

// ValidateRole validates a trust policy document without saving it
func (h *RoleHandlers) ValidateRole(c *gin.Context) {
	var req struct {
		TrustPolicy policy.PolicyDocument `json:"trust_policy" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Basic validation
	if req.TrustPolicy.Version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trust policy version is required"})
		return
	}

	if len(req.TrustPolicy.Statement) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trust policy must have at least one statement"})
		return
	}

	// Validate each statement for trust policy specific requirements
	for i, statement := range req.TrustPolicy.Statement {
		if statement.Effect != "Allow" && statement.Effect != "Deny" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Statement %d: Effect must be 'Allow' or 'Deny'", i+1),
			})
			return
		}

		if len(statement.Action) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Statement %d: Action is required", i+1),
			})
			return
		}

		// Check for STS actions in trust policy
		hasSTSAction := false
		for _, action := range statement.Action {
			if action == "sts:AssumeRole" || action == "sts:AssumeRoleWithWebIdentity" || action == "sts:AssumeRoleWithCredentials" {
				hasSTSAction = true
				break
			}
		}

		if !hasSTSAction {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Statement %d: Trust policy must contain STS actions (sts:AssumeRole, sts:AssumeRoleWithWebIdentity, or sts:AssumeRoleWithCredentials)", i+1),
			})
			return
		}

		// Principal is required for trust policies
		if statement.Principal == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Statement %d: Principal is required in trust policy", i+1),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "Trust policy document is valid",
	})
}

// getRolesData retrieves roles data from the server
func (h *RoleHandlers) getRolesData(c *gin.Context) dash.RolesData {
	username := c.GetString("username")
	if username == "" {
		username = "admin"
	}

	// Get roles
	roles, err := h.adminServer.GetRoles()
	if err != nil {
		glog.Errorf("Failed to get roles: %v", err)
		// Return empty data on error
		return dash.RolesData{
			Username:    username,
			Roles:       []dash.IAMRole{},
			TotalRoles:  0,
			LastUpdated: time.Now(),
		}
	}

	// Ensure roles is never nil
	if roles == nil {
		roles = []dash.IAMRole{}
	}

	return dash.RolesData{
		Username:    username,
		Roles:       roles,
		TotalRoles:  len(roles),
		LastUpdated: time.Now(),
	}
}
