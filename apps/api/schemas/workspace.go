package schemas

import "time"

// Workspace roles. An owner manages the workspace's keys, alerts and source
// maps; a member reads its logs and keeps saved queries.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// InstanceWorkspaceID is the workspace that held every row before tenancy:
// the instance operators' own logs, and where the legacy ingest token writes.
const InstanceWorkspaceID int64 = 1

// Workspace is the tenant every log entry, key, saved query, alert rule and
// source map belongs to. PersonalOf names the account a personal workspace
// was created for; Slug names a workspace the collector routes apps into.
// PlanExempt workspaces are the operators' own and skip the plan limits.
type Workspace struct {
	ID         int64     `json:"id" gorm:"column:id;primaryKey"`
	Name       string    `json:"name" gorm:"column:name;not null"`
	Slug       *string   `json:"slug,omitempty" gorm:"column:slug;uniqueIndex"`
	PersonalOf *int64    `json:"personal_of,omitempty" gorm:"column:personal_of;uniqueIndex"`
	PlanExempt bool      `json:"plan_exempt" gorm:"column:plan_exempt;not null;default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Workspace) TableName() string { return "workspaces" }

// WorkspaceMember grants one account a role in one workspace.
type WorkspaceMember struct {
	WorkspaceID int64     `json:"workspace_id" gorm:"column:workspace_id;primaryKey"`
	UserID      int64     `json:"user_id" gorm:"column:user_id;primaryKey"`
	Role        string    `json:"role" gorm:"column:role;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (WorkspaceMember) TableName() string { return "workspace_members" }
