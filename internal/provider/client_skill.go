// Copyright groundcover 2026
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/groundcover-com/groundcover-sdk-go/pkg/client/agent"
	"github.com/groundcover-com/groundcover-sdk-go/pkg/models"
	gcsdktransport "github.com/groundcover-com/groundcover-sdk-go/pkg/transport"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const terraformProviderUserAgent = "terraform-provider-groundcover"

func skillRequestOptions() []agent.ClientOption {
	return []agent.ClientOption{
		gcsdktransport.WithHeadersOverride(http.Header{
			"User-Agent": []string{terraformProviderUserAgent},
		}),
	}
}

func (c *SdkClientWrapper) CreateSkill(ctx context.Context, req *models.AgentSkillRequest) (*models.AgentSkillDetail, error) {
	tflog.Debug(ctx, "Executing SDK Call: Create Skill")
	params := agent.NewAgentCreateSkillParams().WithContext(ctx).WithTimeout(defaultTimeout).WithBody(req)
	resp, err := c.sdkClient.Agent.AgentCreateSkill(params, nil, skillRequestOptions()...)
	if err != nil {
		return nil, handleApiError(ctx, err, "CreateSkill", skillRequestName(req))
	}
	if resp == nil || resp.Payload == nil || resp.Payload.Skill == nil {
		return nil, errors.New("create skill response payload was nil")
	}
	return resp.Payload.Skill, nil
}

func (c *SdkClientWrapper) GetSkill(ctx context.Context, id string) (*models.AgentSkillDetail, error) {
	tflog.Debug(ctx, "Executing SDK Call: Get Skill", map[string]any{"id": id})
	params := agent.NewAgentGetSkillParams().WithContext(ctx).WithTimeout(defaultTimeout).WithSkillID(id)
	resp, err := c.sdkClient.Agent.AgentGetSkill(params, nil, skillRequestOptions()...)
	if err != nil {
		return nil, handleApiError(ctx, err, "GetSkill", id)
	}
	if resp == nil || resp.Payload == nil || resp.Payload.Skill == nil {
		return nil, errors.New("get skill response payload was nil")
	}
	return resp.Payload.Skill, nil
}

func (c *SdkClientWrapper) UpdateSkill(ctx context.Context, id string, req *models.AgentSkillRequest) (*models.AgentSkillDetail, error) {
	tflog.Debug(ctx, "Executing SDK Call: Update Skill", map[string]any{"id": id})
	params := agent.NewAgentUpdateSkillParams().WithContext(ctx).WithTimeout(defaultTimeout).WithSkillID(id).WithBody(req)
	resp, err := c.sdkClient.Agent.AgentUpdateSkill(params, nil, skillRequestOptions()...)
	if err != nil {
		return nil, handleApiError(ctx, err, "UpdateSkill", skillRequestName(req))
	}
	if resp == nil || resp.Payload == nil || resp.Payload.Skill == nil {
		return nil, errors.New("update skill response payload was nil")
	}
	return resp.Payload.Skill, nil
}

func (c *SdkClientWrapper) DeleteSkill(ctx context.Context, id string) error {
	tflog.Debug(ctx, "Executing SDK Call: Delete Skill", map[string]any{"id": id})
	params := agent.NewAgentDeleteSkillParams().WithContext(ctx).WithTimeout(defaultTimeout).WithSkillID(id)
	_, err := c.sdkClient.Agent.AgentDeleteSkill(params, nil, skillRequestOptions()...)
	if err == nil {
		return nil
	}
	mappedErr := handleApiError(ctx, err, "DeleteSkill", id)
	if errors.Is(mappedErr, ErrNotFound) {
		return nil
	}
	return mappedErr
}

func skillRequestName(req *models.AgentSkillRequest) string {
	if req == nil || req.Name == nil {
		return "<unknown>"
	}
	return *req.Name
}
