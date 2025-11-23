package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seaweedfs/seaweedfs/weed/admin/dash"
	"github.com/seaweedfs/seaweedfs/weed/admin/view/app"
	"github.com/seaweedfs/seaweedfs/weed/admin/view/layout"
	"github.com/seaweedfs/seaweedfs/weed/iam/policy"
	"github.com/seaweedfs/seaweedfs/weed/s3api/policy_engine"
)

type IAMPolicyHandlers struct {
	adminServer *dash.AdminServer
}

func NewIAMPolicyHandlers(adminServer *dash.AdminServer) *IAMPolicyHandlers {
	return &IAMPolicyHandlers{
		adminServer: adminServer,
	}
}

func (h *IAMPolicyHandlers) ShowIAMPolicies(c *gin.Context) {
	// Debug log
	fmt.Printf("DEBUG: ShowIAMPolicies called for path: %s\n", c.Request.URL.Path)
	
	// Get policies data from the server
	data := h.getIAMPoliciesData(c)

	// Convert to PoliciesData for template
	templateData := dash.PoliciesData{
		Username:      data.Username,
		Policies:      convertToIAMPolicies(data.Policies),
		TotalPolicies: data.TotalPolicies,
		LastUpdated:   data.LastUpdated,
	}

	// Render HTML template
	c.Header("Content-Type", "text/html")
	policiesComponent := app.IAMPolicies(templateData)
	layoutComponent := layout.Layout(c, policiesComponent)
	err := layoutComponent.Render(c.Request.Context(), c.Writer)
	if err != nil {
		fmt.Printf("DEBUG: Template render error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template: " + err.Error()})
		return
	}
}

func (h *IAMPolicyHandlers) GetIAMPolicies(c *gin.Context) {
	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy engine not available"})
		return
	}

	store := policyEngine.GetStore()
	if store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy store not available"})
		return
	}

	policyNames, err := store.ListPolicies(context.Background(), "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list IAM policies"})
		return
	}

	policies := make(map[string]*policy.PolicyDocument)
	for _, name := range policyNames {
		policyDoc, err := store.GetPolicy(context.Background(), "", name)
		if err == nil {
			policies[name] = policyDoc
		}
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (h *IAMPolicyHandlers) CreateIAMPolicy(c *gin.Context) {
	var req dash.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy engine not available"})
		return
	}

	store := policyEngine.GetStore()
	if store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy store not available"})
		return
	}

	policyDoc := convertFromEnginePolicy(req.Document)

	err := store.StorePolicy(context.Background(), "", req.Name, policyDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create IAM policy"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "IAM policy created using policy engine",
		"policy":  req.Name,
	})
}

func (h *IAMPolicyHandlers) GetIAMPolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}

	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy engine not available"})
		return
	}

	store := policyEngine.GetStore()
	if store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy store not available"})
		return
	}

	policyDoc, err := store.GetPolicy(context.Background(), "", policyName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "IAM policy not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"policy": convertToEnginePolicy(policyDoc)})
}

func (h *IAMPolicyHandlers) UpdateIAMPolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}

	var req dash.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy engine not available"})
		return
	}

	store := policyEngine.GetStore()
	if store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy store not available"})
		return
	}

	policyDoc := convertFromEnginePolicy(req.Document)

	err := store.StorePolicy(context.Background(), "", policyName, policyDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update IAM policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IAM policy updated using policy engine",
		"policy":  policyName,
	})
}

func (h *IAMPolicyHandlers) DeleteIAMPolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Policy name is required"})
		return
	}

	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy engine not available"})
		return
	}

	store := policyEngine.GetStore()
	if store == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy store not available"})
		return
	}

	err := store.DeletePolicy(context.Background(), "", policyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete IAM policy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IAM policy deleted using policy engine",
		"policy":  policyName,
	})
}

func (h *IAMPolicyHandlers) ValidateIAMPolicy(c *gin.Context) {
	var req struct {
		Document policy_engine.PolicyDocument `json:"document" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Document.Version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "Policy version is required"})
		return
	}

	if len(req.Document.Statement) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "Policy must have at least one statement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Policy document is valid"})
}

func (h *IAMPolicyHandlers) getIAMPoliciesData(c *gin.Context) dash.IAMPoliciesData {
	username := c.GetString("username")
	if username == "" {
		username = "admin"
	}

	policiesData := dash.IAMPoliciesData{
		Username:    username,
		Policies:    make(map[string]policy_engine.PolicyDocument),
		LastUpdated: time.Now(),
	}

	// Get policies from policy engine
	policyEngine := h.adminServer.GetPolicyEngine()
	if policyEngine != nil {
		store := policyEngine.GetStore()
		if store != nil {
			policyNames, err := store.ListPolicies(context.Background(), "")
			if err == nil {
				for _, name := range policyNames {
					policyDoc, err := store.GetPolicy(context.Background(), "", name)
					if err == nil {
						policiesData.Policies[name] = convertToEnginePolicy(policyDoc)
					}
				}
			}
		}
	}

	policiesData.TotalPolicies = len(policiesData.Policies)
	return policiesData
}

func convertToIAMPolicies(policies map[string]policy_engine.PolicyDocument) []dash.IAMPolicy {
	result := make([]dash.IAMPolicy, 0, len(policies))
	for name, policy := range policies {
		result = append(result, dash.IAMPolicy{
			Name:         name,
			Document:     policy,
			DocumentJSON: "",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		})
	}
	return result
}

func convertFromEnginePolicy(engineDoc policy_engine.PolicyDocument) *policy.PolicyDocument {
	statements := make([]policy.Statement, len(engineDoc.Statement))
	for i, stmt := range engineDoc.Statement {
		statements[i] = policy.Statement{
			Sid:       stmt.Sid,
			Effect:    string(stmt.Effect),
			Principal: stmt.Principal,
			Action:    stmt.Action.Strings(),
			Resource:  stmt.Resource.Strings(),
			Condition: convertConditions(stmt.Condition),
		}
	}

	return &policy.PolicyDocument{
		Version:   engineDoc.Version,
		Statement: statements,
	}
}

func convertToEnginePolicy(policyDoc *policy.PolicyDocument) policy_engine.PolicyDocument {
	if policyDoc == nil {
		return policy_engine.PolicyDocument{}
	}

	statements := make([]policy_engine.PolicyStatement, len(policyDoc.Statement))
	for i, stmt := range policyDoc.Statement {
		var principal *policy_engine.StringOrStringSlice
		if stmt.Principal != nil {
			if p, ok := stmt.Principal.(*policy_engine.StringOrStringSlice); ok {
				principal = p
			}
		}

		statements[i] = policy_engine.PolicyStatement{
			Sid:       stmt.Sid,
			Effect:    policy_engine.PolicyEffect(stmt.Effect),
			Principal: principal,
			Action:    policy_engine.NewStringOrStringSlice(stmt.Action...),
			Resource:  policy_engine.NewStringOrStringSlice(stmt.Resource...),
			Condition: convertConditionsToEngine(stmt.Condition),
		}
	}

	return policy_engine.PolicyDocument{
		Version:   policyDoc.Version,
		Statement: statements,
	}
}

func convertConditions(conditions policy_engine.PolicyConditions) map[string]map[string]interface{} {
	if conditions == nil {
		return nil
	}
	result := make(map[string]map[string]interface{})
	for key, value := range conditions {
		// Convert map[string]StringOrStringSlice to map[string]interface{}
		convertedValue := make(map[string]interface{})
		for k, v := range value {
			convertedValue[k] = v.Strings()
		}
		result[key] = convertedValue
	}
	return result
}

func convertConditionsToEngine(conditions map[string]map[string]interface{}) policy_engine.PolicyConditions {
	if conditions == nil {
		return nil
	}
	result := make(policy_engine.PolicyConditions)
	for key, value := range conditions {
		// Convert map[string]interface{} to map[string]StringOrStringSlice
		convertedValue := make(map[string]policy_engine.StringOrStringSlice)
		for k, v := range value {
			if strSlice, ok := v.([]string); ok {
				convertedValue[k] = policy_engine.NewStringOrStringSlice(strSlice...)
			} else if str, ok := v.(string); ok {
				convertedValue[k] = policy_engine.NewStringOrStringSlice(str)
			} else {
				// Convert to string as fallback
				convertedValue[k] = policy_engine.NewStringOrStringSlice(fmt.Sprintf("%v", v))
			}
		}
		result[key] = convertedValue
	}
	return result
}
