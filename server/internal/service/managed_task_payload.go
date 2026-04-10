package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/daemon"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type ManagedExecutionTask struct {
	Task       daemon.Task
	Provider   string
	AuthUserID string
}

func (s *TaskService) BuildManagedExecutionTask(ctx context.Context, task db.AgentTaskQueue) (*ManagedExecutionTask, error) {
	agent, err := s.Queries.GetAgent(ctx, task.AgentID)
	if err != nil {
		return nil, fmt.Errorf("load agent: %w", err)
	}

	provider, err := s.resolveManagedProvider(ctx, task, agent)
	if err != nil {
		return nil, err
	}

	authUserID, err := s.resolveManagedAuthUser(ctx, task, agent)
	if err != nil {
		return nil, err
	}

	result := &ManagedExecutionTask{
		Task: daemon.Task{
			ID:               util.UUIDToString(task.ID),
			AgentID:          util.UUIDToString(task.AgentID),
			RuntimeID:        util.UUIDToString(task.RuntimeID),
			IssueID:          util.UUIDToString(task.IssueID),
			Agent:            &daemon.AgentData{ID: util.UUIDToString(agent.ID), Name: agent.Name, Instructions: agent.Instructions, Skills: convertManagedSkills(s.LoadAgentSkills(ctx, task.AgentID))},
			TriggerCommentID: util.UUIDToString(task.TriggerCommentID),
		},
		Provider:   provider,
		AuthUserID: authUserID,
	}

	if task.IssueID.Valid {
		issue, err := s.Queries.GetIssue(ctx, task.IssueID)
		if err != nil {
			return nil, fmt.Errorf("load issue: %w", err)
		}
		result.Task.WorkspaceID = util.UUIDToString(issue.WorkspaceID)
		result.Task.IssueTitle = issue.Title
		if issue.Description.Valid {
			result.Task.IssueDescription = issue.Description.String
		}
		if ws, err := s.Queries.GetWorkspace(ctx, issue.WorkspaceID); err == nil && len(ws.Repos) > 0 {
			var repos []daemon.RepoData
			if json.Unmarshal(ws.Repos, &repos) == nil {
				result.Task.Repos = repos
			}
		}
		if prior, err := s.Queries.GetLastTaskSession(ctx, db.GetLastTaskSessionParams{
			AgentID: task.AgentID,
			IssueID: task.IssueID,
		}); err == nil && prior.SessionID.Valid {
			result.Task.PriorSessionID = prior.SessionID.String
			if prior.WorkDir.Valid {
				result.Task.PriorWorkDir = prior.WorkDir.String
			}
		}
	}

	if task.ChatSessionID.Valid {
		cs, err := s.Queries.GetChatSession(ctx, task.ChatSessionID)
		if err != nil {
			return nil, fmt.Errorf("load chat session: %w", err)
		}
		result.Task.WorkspaceID = util.UUIDToString(cs.WorkspaceID)
		result.Task.ChatSessionID = util.UUIDToString(cs.ID)
		if ws, err := s.Queries.GetWorkspace(ctx, cs.WorkspaceID); err == nil && len(ws.Repos) > 0 {
			var repos []daemon.RepoData
			if json.Unmarshal(ws.Repos, &repos) == nil {
				result.Task.Repos = repos
			}
		}
		if cs.SessionID.Valid {
			result.Task.PriorSessionID = cs.SessionID.String
		}
		if cs.WorkDir.Valid {
			result.Task.PriorWorkDir = cs.WorkDir.String
		}
		if msgs, err := s.Queries.ListChatMessages(ctx, cs.ID); err == nil {
			for i := len(msgs) - 1; i >= 0; i-- {
				if msgs[i].Role == "user" {
					result.Task.ChatMessage = msgs[i].Content
					break
				}
			}
		}
	}

	return result, nil
}

func (s *TaskService) resolveManagedProvider(ctx context.Context, task db.AgentTaskQueue, agent db.Agent) (string, error) {
	workspaceID := pgtype.UUID{}
	switch {
	case task.IssueID.Valid:
		issue, err := s.Queries.GetIssue(ctx, task.IssueID)
		if err == nil {
			workspaceID = issue.WorkspaceID
		}
	case task.ChatSessionID.Valid:
		cs, err := s.Queries.GetChatSession(ctx, task.ChatSessionID)
		if err == nil {
			workspaceID = cs.WorkspaceID
		}
	}

	if task.LeaseID.Valid {
		if workspaceID.Valid {
			lease, err := s.Queries.GetRuntimeLeaseInWorkspace(ctx, db.GetRuntimeLeaseInWorkspaceParams{
				ID:          task.LeaseID,
				WorkspaceID: workspaceID,
			})
			if err != nil {
				return "", fmt.Errorf("load runtime lease: %w", err)
			}
			if lease.Provider != "" {
				return lease.Provider, nil
			}
		}
	}
	if task.IssueID.Valid {
		issue, err := s.Queries.GetIssue(ctx, task.IssueID)
		if err == nil {
			policy, policyErr := s.loadRuntimePolicy(ctx, issue.WorkspaceID)
			if policyErr == nil && policy.DefaultProvider != "" {
				return policy.DefaultProvider, nil
			}
		}
	}
	if task.ChatSessionID.Valid {
		cs, err := s.Queries.GetChatSession(ctx, task.ChatSessionID)
		if err == nil {
			policy, policyErr := s.loadRuntimePolicy(ctx, cs.WorkspaceID)
			if policyErr == nil && policy.DefaultProvider != "" {
				return policy.DefaultProvider, nil
			}
		}
	}
	if agent.RuntimeMode != "" {
		return agent.RuntimeMode, nil
	}
	return "", fmt.Errorf("no provider resolved for managed task %s", util.UUIDToString(task.ID))
}

func (s *TaskService) resolveManagedAuthUser(ctx context.Context, task db.AgentTaskQueue, agent db.Agent) (string, error) {
	if agent.OwnerID.Valid {
		return util.UUIDToString(agent.OwnerID), nil
	}

	workspaceID := pgtype.UUID{}
	switch {
	case task.IssueID.Valid:
		issue, err := s.Queries.GetIssue(ctx, task.IssueID)
		if err != nil {
			return "", fmt.Errorf("load issue for auth user: %w", err)
		}
		workspaceID = issue.WorkspaceID
	case task.ChatSessionID.Valid:
		cs, err := s.Queries.GetChatSession(ctx, task.ChatSessionID)
		if err != nil {
			return "", fmt.Errorf("load chat session for auth user: %w", err)
		}
		workspaceID = cs.WorkspaceID
	default:
		return "", fmt.Errorf("no workspace context for managed task %s", util.UUIDToString(task.ID))
	}

	members, err := s.Queries.ListMembers(ctx, workspaceID)
	if err != nil {
		return "", fmt.Errorf("list workspace members: %w", err)
	}
	if len(members) == 0 {
		return "", fmt.Errorf("workspace has no members for managed task auth")
	}

	for _, member := range members {
		if member.Role == "owner" || member.Role == "admin" {
			return util.UUIDToString(member.UserID), nil
		}
	}
	return util.UUIDToString(members[0].UserID), nil
}

func convertManagedSkills(skills []AgentSkillData) []daemon.SkillData {
	if len(skills) == 0 {
		return nil
	}
	result := make([]daemon.SkillData, 0, len(skills))
	for _, skill := range skills {
		item := daemon.SkillData{
			Name:    skill.Name,
			Content: skill.Content,
		}
		for _, file := range skill.Files {
			item.Files = append(item.Files, daemon.SkillFileData{
				Path:    file.Path,
				Content: file.Content,
			})
		}
		result = append(result, item)
	}
	return result
}
