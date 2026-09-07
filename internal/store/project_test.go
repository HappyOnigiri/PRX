package store_test

import (
	"context"
	"testing"

	"github.com/HappyOnigiri/PRX/internal/app"
	"github.com/HappyOnigiri/PRX/internal/domain"
)

func TestProjectAllocatesPublicIDsAndNormalizesItsValues(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)

	project, err := service.CreateProject(ctx, "  Payments platform  ", "  Shared work  ")
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "P-1" || project.Title != "Payments platform" ||
		project.Description != "Shared work" || project.Archived {
		t.Fatalf("project=%+v", project)
	}
	second, err := service.CreateProject(ctx, "Billing", "")
	if err != nil || second.ID != "P-2" {
		t.Fatalf("second project=%+v err=%v", second, err)
	}
	byID, err := service.ResolveProject(ctx, project.ID)
	if err != nil || byID.Title != project.Title {
		t.Fatalf("resolved by ID=%+v err=%v", byID, err)
	}
	if _, err := service.ResolveProject(ctx, "P-9"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("unknown project code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.CreateProject(ctx, "  ", ""); domain.ErrorCode(err) !=
		domain.DomainErrorCodeInvalidTitle {
		t.Fatalf("missing title code=%s", domain.ErrorCode(err))
	}
}

// The public ID prefix decides the kind, so a project and a feature never
// compete for the same operand.
func TestGetNodeResolvesProjectsAndFeaturesByPublicID(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Shared", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Shared feature", "", project.ID)
	if err != nil {
		t.Fatal(err)
	}
	node, err := service.GetNode(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := node.(domain.Project); !ok || got.ID != project.ID {
		t.Fatalf("project node=%#v", node)
	}
	node, err = service.GetNode(ctx, feature.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := node.(domain.Feature); !ok || got.ID != feature.ID {
		t.Fatalf("feature node=%#v", node)
	}
	if _, err := service.GetNode(ctx, "P-9"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing project node code=%s err=%v", domain.ErrorCode(err), err)
	}
}

// Membership is required and can only move from one project to another, so the
// empty request that used to detach a feature is refused.
func TestFeatureProjectMembershipIsRequiredAndMovable(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "Payments", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateProject(ctx, "Search", "")
	if err != nil {
		t.Fatal(err)
	}
	feature, err := service.CreateFeature(ctx, "Checkout", "", project.ID)
	if err != nil || feature.ProjectID != project.ID {
		t.Fatalf("created feature=%+v err=%v", feature, err)
	}
	if _, err := service.CreateFeature(
		ctx, "Orphan", "", "",
	); domain.ErrorCode(err) != domain.DomainErrorCodeInvalidParent {
		t.Fatalf("feature without a project code=%s err=%v", domain.ErrorCode(err), err)
	}
	empty := ""
	if _, err := service.UpdateFeature(
		ctx, feature.ID, domain.FeatureUpdate{ProjectID: &empty},
	); domain.ErrorCode(err) != domain.DomainErrorCodeInvalidParent {
		t.Fatalf("detach code=%s err=%v", domain.ErrorCode(err), err)
	}
	moved, err := service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{ProjectID: &second.ID})
	if err != nil || moved.ProjectID != second.ID {
		t.Fatalf("moved feature=%+v err=%v", moved, err)
	}
	// 所属を省略した場合は現在の所属をそのまま保つ。
	renamed, err := service.UpdateFeature(ctx, feature.ID, domain.FeatureUpdate{Title: stringPointer("Renamed")})
	if err != nil || renamed.ProjectID != second.ID {
		t.Fatalf("renamed feature=%+v err=%v", renamed, err)
	}
	missing := "P-9"
	if _, err := service.UpdateFeature(
		ctx, feature.ID, domain.FeatureUpdate{ProjectID: &missing},
	); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing project code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.CreateFeature(
		ctx, "Orphan", "", "P-9",
	); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing project on create code=%s", domain.ErrorCode(err))
	}
}

// readOnlyFixture is one project holding one feature with every kind of
// contained record, so archiving either container exercises the whole guard.
type readOnlyFixture struct {
	service         *app.Service
	project         domain.Project
	feature         domain.Feature
	blocker         domain.Task
	blocked         domain.Task
	projectDocument domain.Document
	featureDocument domain.Document
	taskDocument    domain.Document
}

func newReadOnlyFixture(t *testing.T) readOnlyFixture {
	t.Helper()
	ctx := context.Background()
	_, service := openTestService(t)
	fixture := readOnlyFixture{service: service}
	var err error
	if fixture.project, err = service.CreateProject(ctx, "Payments", ""); err != nil {
		t.Fatal(err)
	}
	if fixture.feature, err = service.CreateFeature(ctx, "Checkout", "", fixture.project.ID); err != nil {
		t.Fatal(err)
	}
	if fixture.blocker, err = service.CreateTask(
		ctx, fixture.feature.ID, "Blocker", "", "",
	); err != nil {
		t.Fatal(err)
	}
	if fixture.blocked, err = service.CreateTask(
		ctx, fixture.feature.ID, "Blocked", "", "",
	); err != nil {
		t.Fatal(err)
	}
	if _, err = service.AddDependency(ctx, fixture.blocker.ID, fixture.blocked.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.AttachPullRequest(ctx, fixture.blocker.ID, "https://github.com/acme/api/pull/1"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.UpsertImplementationPlan(
		ctx, fixture.blocker.ID, domain.Document{Kind: domain.DocumentKindMarkdown, Content: "# Plan"},
	); err != nil {
		t.Fatal(err)
	}
	for _, document := range []struct {
		target *domain.Document
		parent domain.DocumentParent
	}{
		{target: &fixture.projectDocument, parent: domain.DocumentParent{ProjectID: fixture.project.ID}},
		{target: &fixture.featureDocument, parent: domain.DocumentParent{FeatureID: fixture.feature.ID}},
		{target: &fixture.taskDocument, parent: domain.DocumentParent{TaskID: fixture.blocked.ID}},
	} {
		if *document.target, err = service.AddDocument(
			ctx, document.parent, domain.DocumentKindURL, "Reference", "https://example.com/reference", "", false,
		); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

// refusedWrites is every operation the read-only barrier must reject, named so
// a failure says which one slipped through.
func (f readOnlyFixture) refusedWrites(ctx context.Context) map[string]error {
	service := f.service
	_, dependencyErr := service.AddDependency(ctx, f.blocker.ID, f.blocked.ID)
	_, taskErr := service.CreateTask(ctx, f.feature.ID, "New", "", "")
	_, updateTaskErr := service.UpdateTask(ctx, f.blocker.ID, stringPointer("Renamed"), nil, nil, nil)
	_, pullRequestErr := service.AttachPullRequest(ctx, f.blocked.ID, "https://github.com/acme/api/pull/2")
	_, planErr := service.UpsertImplementationPlan(
		ctx, f.blocked.ID, domain.Document{Kind: domain.DocumentKindMarkdown, Content: "# Later"},
	)
	_, projectDocumentErr := service.AddDocument(
		ctx, domain.DocumentParent{ProjectID: f.project.ID},
		domain.DocumentKindURL, "Late", "https://example.com/late", "", false,
	)
	_, featureDocumentErr := service.AddDocument(
		ctx, domain.DocumentParent{FeatureID: f.feature.ID},
		domain.DocumentKindURL, "Late", "https://example.com/late", "", false,
	)
	_, taskDocumentErr := service.AddDocument(
		ctx, domain.DocumentParent{TaskID: f.blocked.ID},
		domain.DocumentKindURL, "Late", "https://example.com/late", "", false,
	)
	_, updateDocumentErr := service.UpdateDocument(ctx, f.taskDocument.ID, stringPointer("Renamed"), nil, nil)
	_, updateFeatureDocumentErr := service.UpdateDocument(
		ctx, f.featureDocument.ID, stringPointer("Renamed"), nil, nil,
	)
	_, updateProjectDocumentErr := service.UpdateDocument(
		ctx, f.projectDocument.ID, stringPointer("Renamed"), nil, nil,
	)
	_, featureErr := service.UpdateFeature(ctx, f.feature.ID, domain.FeatureUpdate{Title: stringPointer("Renamed")})
	empty := ""
	_, membershipErr := service.UpdateFeature(ctx, f.feature.ID, domain.FeatureUpdate{ProjectID: &empty})
	_, projectErr := service.UpdateProject(ctx, f.project.ID, domain.ProjectUpdate{Title: stringPointer("Renamed")})
	return map[string]error{
		"add dependency":           dependencyErr,
		"remove dependency":        service.RemoveDependency(ctx, f.blocker.ID, f.blocked.ID),
		"create task":              taskErr,
		"update task":              updateTaskErr,
		"delete task":              service.DeleteTask(ctx, f.blocked.ID, true),
		"attach pull request":      pullRequestErr,
		"detach pull request":      service.DetachPullRequest(ctx, f.blocker.ID),
		"upsert plan":              planErr,
		"delete plan":              service.DeleteImplementationPlan(ctx, f.blocker.ID),
		"add project document":     projectDocumentErr,
		"add feature document":     featureDocumentErr,
		"add task document":        taskDocumentErr,
		"update task document":     updateDocumentErr,
		"update feature document":  updateFeatureDocumentErr,
		"update project document":  updateProjectDocumentErr,
		"delete task document":     service.DeleteDocument(ctx, f.taskDocument.ID),
		"delete feature document":  service.DeleteDocument(ctx, f.featureDocument.ID),
		"delete project document":  service.DeleteDocument(ctx, f.projectDocument.ID),
		"update feature":           featureErr,
		"change project membershp": membershipErr,
		"update project":           projectErr,
	}
}

func TestArchivedProjectRefusesEveryWriteInsideIt(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	archived := true
	if _, err := fixture.service.UpdateProject(
		ctx, fixture.project.ID, domain.ProjectUpdate{Archived: &archived},
	); err != nil {
		t.Fatal(err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 1 || !snapshot.Projects[0].Archived {
		t.Fatalf("projects=%+v", snapshot.Projects)
	}
	// feature 自体はアーカイブされないまま、読み取り専用として提示される。
	if snapshot.Features[0].Archived || !snapshot.Features[0].ReadOnly {
		t.Fatalf("feature=%+v, want read-only but not archived", snapshot.Features[0])
	}
	for name, err := range fixture.refusedWrites(ctx) {
		if domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
			t.Errorf("%s: code=%s err=%v, want archived_read_only", name, domain.ErrorCode(err), err)
		}
	}
	// アーカイブ解除だけはプロジェクトが受け付ける更新で、
	// 中身すべてへの書き込みを回復させる。
	active := false
	if _, err := fixture.service.UpdateProject(
		ctx, fixture.project.ID, domain.ProjectUpdate{Archived: &active},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.CreateTask(
		ctx, fixture.feature.ID, "After", "", "",
	); err != nil {
		t.Fatalf("write after unarchiving: %v", err)
	}
}

func TestArchivedFeatureRefusesWritesButStaysDeletable(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	archived := true
	if _, err := fixture.service.UpdateFeature(
		ctx, fixture.feature.ID, domain.FeatureUpdate{Archived: &archived},
	); err != nil {
		t.Fatal(err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Features[0].Archived || !snapshot.Features[0].ReadOnly {
		t.Fatalf("feature=%+v", snapshot.Features[0])
	}
	for name, err := range fixture.refusedWrites(ctx) {
		// プロジェクト自体はまだ active なので、自身への書き込みは許可される。
		if name == "update project" || name == "add project document" || name == "update project document" ||
			name == "delete project document" {
			if err != nil {
				t.Errorf("%s: err=%v, want the active project to accept the write", name, err)
			}
			continue
		}
		if domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
			t.Errorf("%s: code=%s err=%v, want archived_read_only", name, domain.ErrorCode(err), err)
		}
	}
	// アーカイブ済みの作業を捨てられることが、アーカイブを実用に保つ例外。
	if err := fixture.service.DeleteFeature(ctx, fixture.feature.ID, true); err != nil {
		t.Fatalf("delete archived feature: %v", err)
	}
}

// The archived flag moves in either direction past the barrier, so an archive
// command stays idempotent and a feature whose own flag was cleared inside an
// archived project can be put back the way it was.
func TestArchivedFlagMovesInBothDirectionsPastTheBarrier(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	archived, active := true, false
	setProject := func(value *bool) error {
		_, err := fixture.service.UpdateProject(ctx, fixture.project.ID, domain.ProjectUpdate{Archived: value})
		return err
	}
	setFeature := func(value *bool) error {
		_, err := fixture.service.UpdateFeature(ctx, fixture.feature.ID, domain.FeatureUpdate{Archived: value})
		return err
	}
	for _, step := range []struct {
		name string
		err  error
	}{
		{"archive the project", setProject(&archived)},
		{"archive the already archived project", setProject(&archived)},
		{"archive the feature inside the archived project", setFeature(&archived)},
		{"unarchive the feature inside the archived project", setFeature(&active)},
		{"re-archive the feature inside the archived project", setFeature(&archived)},
	} {
		if step.err != nil {
			t.Errorf("%s: %v", step.name, step.err)
		}
	}

	// 通るのはフラグだけ。他の変更と一緒に解除するのは、
	// アーカイブ済みの入れ物への書き込みに変わりない。
	if _, err := fixture.service.UpdateFeature(
		ctx, fixture.feature.ID, domain.FeatureUpdate{Title: stringPointer("Renamed"), Archived: &active},
	); domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
		t.Errorf("unarchive with a title change: code=%s err=%v, want archived_read_only",
			domain.ErrorCode(err), err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Features[0].Archived || !snapshot.Features[0].ReadOnly {
		t.Fatalf("feature=%+v, want archived and read-only", snapshot.Features[0])
	}
}

// A feature that is archived nowhere, neither on its own nor through its
// project, is not read-only; that is the third case the derivation has to get
// right.
func TestActiveFeatureInActiveProjectIsNotReadOnly(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Features[0].ReadOnly {
		t.Fatalf("feature=%+v, want writable", snapshot.Features[0])
	}
}

// A cascade empties the project: its own documents and the features it holds,
// with everything those features own. A feature cannot survive its project,
// because it belongs to one.
func TestProjectDeleteRemovesItsDocumentsAndTheFeaturesItHolds(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	if err := fixture.service.DeleteProject(ctx, fixture.project.ID, false); domain.ErrorCode(err) !=
		domain.DomainErrorCodeReferencesExist {
		t.Fatalf("delete without cascade code=%s err=%v", domain.ErrorCode(err), err)
	}
	// アーカイブ済みプロジェクトも削除できる。削除こそが、
	// アーカイブした作業を最終的に捨てる手段。
	archived := true
	if _, err := fixture.service.UpdateProject(
		ctx, fixture.project.ID, domain.ProjectUpdate{Archived: &archived},
	); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.DeleteProject(ctx, fixture.project.ID, true); err != nil {
		t.Fatal(err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 0 || len(snapshot.Features) != 0 ||
		len(snapshot.Tasks) != 0 || len(snapshot.Dependencies) != 0 ||
		len(snapshot.Documents) != 0 {
		t.Fatalf(
			"projects=%+v features=%+v tasks=%+v dependencies=%+v documents=%+v",
			snapshot.Projects, snapshot.Features, snapshot.Tasks,
			snapshot.Dependencies, snapshot.Documents,
		)
	}
}

func stringPointer(value string) *string { return &value }
