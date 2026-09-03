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

	project, err := service.CreateProject(ctx, "  Payments  ", "  Payments platform  ", "  Shared work  ")
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "P-1" || project.Slug != "payments" || project.Title != "Payments platform" ||
		project.Description != "Shared work" || project.Archived {
		t.Fatalf("project=%+v", project)
	}
	second, err := service.CreateProject(ctx, "billing", "Billing", "")
	if err != nil || second.ID != "P-2" {
		t.Fatalf("second project=%+v err=%v", second, err)
	}
	bySlug, err := service.ResolveProject(ctx, "payments")
	if err != nil || bySlug.ID != project.ID {
		t.Fatalf("resolved by slug=%+v err=%v", bySlug, err)
	}
	if _, err := service.ResolveProject(ctx, "unknown"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("unknown project code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.CreateProject(ctx, "Not A Slug", "Title", ""); domain.ErrorCode(err) !=
		domain.DomainErrorCodeInvalidSlug {
		t.Fatalf("invalid slug code=%s", domain.ErrorCode(err))
	}
	if _, err := service.CreateProject(ctx, "titleless", "  ", ""); domain.ErrorCode(err) !=
		domain.DomainErrorCodeInvalidTitle {
		t.Fatalf("missing title code=%s", domain.ErrorCode(err))
	}
}

// The public ID prefix decides the kind, and a bare slug resolves to a feature
// before a project so the two independent namespaces stay predictable.
func TestGetNodeResolvesProjectsByPublicIDAndFallsBackFromFeatureSlugs(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "shared", "Shared", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateFeature(ctx, "shared", "Shared feature", "", ""); err != nil {
		t.Fatal(err)
	}
	node, err := service.GetNode(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := node.(domain.Project); !ok || got.ID != project.ID {
		t.Fatalf("project node=%#v", node)
	}
	node, err = service.GetNode(ctx, "shared")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := node.(domain.Feature); !ok || got.Slug != "shared" {
		t.Fatalf("colliding slug node=%#v, want the feature", node)
	}
	if _, err := service.GetNode(ctx, "P-9"); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing project node code=%s err=%v", domain.ErrorCode(err), err)
	}
}

func TestFeatureProjectMembershipIsOptionalAndReversible(t *testing.T) {
	ctx := context.Background()
	_, service := openTestService(t)
	project, err := service.CreateProject(ctx, "payments", "Payments", "")
	if err != nil {
		t.Fatal(err)
	}
	// A slug is accepted wherever a public project ID is, and the stored value
	// is always the public ID.
	feature, err := service.CreateFeature(ctx, "checkout", "Checkout", "", "payments")
	if err != nil || feature.ProjectID != project.ID {
		t.Fatalf("created feature=%+v err=%v", feature, err)
	}
	unaffiliated, err := service.CreateFeature(ctx, "search", "Search", "", "")
	if err != nil || unaffiliated.ProjectID != "" {
		t.Fatalf("unaffiliated feature=%+v err=%v", unaffiliated, err)
	}
	empty := ""
	detached, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, nil, &empty)
	if err != nil || detached.ProjectID != "" {
		t.Fatalf("detached feature=%+v err=%v", detached, err)
	}
	// An omitted membership leaves the current one alone.
	renamed, err := service.UpdateFeature(ctx, feature.ID, nil, stringPointer("Renamed"), nil, nil, nil, nil)
	if err != nil || renamed.ProjectID != "" {
		t.Fatalf("renamed feature=%+v err=%v", renamed, err)
	}
	reattached, err := service.UpdateFeature(ctx, feature.ID, nil, nil, nil, nil, nil, &project.ID)
	if err != nil || reattached.ProjectID != project.ID {
		t.Fatalf("reattached feature=%+v err=%v", reattached, err)
	}
	missing := "P-9"
	if _, err := service.UpdateFeature(
		ctx, feature.ID, nil, nil, nil, nil, nil, &missing,
	); domain.ErrorCode(err) != domain.DomainErrorCodeNotFound {
		t.Fatalf("missing project code=%s err=%v", domain.ErrorCode(err), err)
	}
	if _, err := service.CreateFeature(
		ctx, "orphan", "Orphan", "", "P-9",
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
	if fixture.project, err = service.CreateProject(ctx, "payments", "Payments", ""); err != nil {
		t.Fatal(err)
	}
	if fixture.feature, err = service.CreateFeature(ctx, "checkout", "Checkout", "", "payments"); err != nil {
		t.Fatal(err)
	}
	if fixture.blocker, err = service.CreateTask(
		ctx, fixture.feature.ID, "Blocker", "", domain.TaskKindPR, "",
	); err != nil {
		t.Fatal(err)
	}
	if fixture.blocked, err = service.CreateTask(
		ctx, fixture.feature.ID, "Blocked", "", domain.TaskKindPR, "",
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
	_, taskErr := service.CreateTask(ctx, f.feature.ID, "New", "", domain.TaskKindManual, "")
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
	_, featureErr := service.UpdateFeature(ctx, f.feature.ID, nil, stringPointer("Renamed"), nil, nil, nil, nil)
	empty := ""
	_, membershipErr := service.UpdateFeature(ctx, f.feature.ID, nil, nil, nil, nil, nil, &empty)
	_, projectErr := service.UpdateProject(ctx, f.project.ID, nil, stringPointer("Renamed"), nil, nil)
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
	if _, err := fixture.service.UpdateProject(ctx, fixture.project.ID, nil, nil, nil, &archived); err != nil {
		t.Fatal(err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 1 || !snapshot.Projects[0].Archived {
		t.Fatalf("projects=%+v", snapshot.Projects)
	}
	// The feature is presented as read-only without being archived itself.
	if snapshot.Features[0].Archived || !snapshot.Features[0].ReadOnly {
		t.Fatalf("feature=%+v, want read-only but not archived", snapshot.Features[0])
	}
	for name, err := range fixture.refusedWrites(ctx) {
		if domain.ErrorCode(err) != domain.DomainErrorCodeArchivedReadOnly {
			t.Errorf("%s: code=%s err=%v, want archived_read_only", name, domain.ErrorCode(err), err)
		}
	}
	// Lifting the archive is the one update the project accepts, and it restores
	// writes to everything inside it.
	active := false
	if _, err := fixture.service.UpdateProject(ctx, fixture.project.ID, nil, nil, nil, &active); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.CreateTask(
		ctx, fixture.feature.ID, "After", "", domain.TaskKindManual, "",
	); err != nil {
		t.Fatalf("write after unarchiving: %v", err)
	}
}

func TestArchivedFeatureRefusesWritesButStaysDeletable(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	archived := true
	if _, err := fixture.service.UpdateFeature(
		ctx, fixture.feature.ID, nil, nil, nil, nil, &archived, nil,
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
		// The project itself is still active, so its own writes stay allowed.
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
	// Discarding archived work is the exception that keeps the archive usable.
	if err := fixture.service.DeleteFeature(ctx, fixture.feature.ID, true); err != nil {
		t.Fatalf("delete archived feature: %v", err)
	}
}

// A feature with no project and no archive of its own is not read-only, which
// is the third case the derivation has to get right.
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

func TestProjectDeleteReleasesFeaturesAndRemovesOnlyItsOwnDocuments(t *testing.T) {
	ctx := context.Background()
	fixture := newReadOnlyFixture(t)
	if err := fixture.service.DeleteProject(ctx, fixture.project.ID, false); domain.ErrorCode(err) !=
		domain.DomainErrorCodeReferencesExist {
		t.Fatalf("delete without cascade code=%s err=%v", domain.ErrorCode(err), err)
	}
	// An archived project is still deletable; the release of its features is
	// part of the deletion rather than a forbidden membership change.
	archived := true
	if _, err := fixture.service.UpdateProject(ctx, fixture.project.ID, nil, nil, nil, &archived); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.DeleteProject(ctx, fixture.project.ID, true); err != nil {
		t.Fatal(err)
	}
	snapshot, err := fixture.service.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Projects) != 0 || len(snapshot.Features) != 1 {
		t.Fatalf("projects=%+v features=%+v", snapshot.Projects, snapshot.Features)
	}
	feature := snapshot.Features[0]
	if feature.ProjectID != "" || feature.ReadOnly || feature.TaskCount != 2 {
		t.Fatalf("released feature=%+v", feature)
	}
	for _, document := range snapshot.Documents {
		if document.ID == fixture.projectDocument.ID {
			t.Fatalf("project document survived the cascade: %+v", document)
		}
	}
	if len(snapshot.Documents) != 3 {
		t.Fatalf("documents=%+v, want the feature, task, and plan documents", snapshot.Documents)
	}
}

func stringPointer(value string) *string { return &value }
