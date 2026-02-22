package course_test

import (
	"context"
	"testing"
	"time"

	course "github.com/amnayem/skillforge/services/course"
	"github.com/amnayem/skillforge/shared/pb/coursepb"
)

type mockRepo struct {
	courses map[string]course.CourseRecord
	modules map[string]course.ModuleRecord
	lessons map[string]course.LessonRecord
	nextID  int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		courses: make(map[string]course.CourseRecord),
		modules: make(map[string]course.ModuleRecord),
		lessons: make(map[string]course.LessonRecord),
	}
}

func (m *mockRepo) Create(ctx context.Context, tenantID, instructorID, title, desc, category, difficulty string) (course.CourseRecord, error) {
	m.nextID++
	c := course.CourseRecord{
		ID: "c-" + title, TenantID: tenantID, InstructorID: instructorID,
		Title: title, Description: desc, Category: category, Difficulty: difficulty,
		Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.courses[c.ID] = c
	return c, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (course.CourseRecord, error) {
	c, ok := m.courses[id]
	if !ok {
		return c, course.ErrCourseNotFound
	}
	return c, nil
}

func (m *mockRepo) Update(ctx context.Context, id, title, desc, category, difficulty, thumbnailURL string) (course.CourseRecord, error) {
	c, ok := m.courses[id]
	if !ok {
		return c, course.ErrCourseNotFound
	}
	c.Title = title
	m.courses[id] = c
	return c, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.courses[id]; !ok {
		return course.ErrCourseNotFound
	}
	delete(m.courses, id)
	return nil
}

func (m *mockRepo) List(ctx context.Context, tenantID, category string, page, pageSize int) ([]course.CourseRecord, int, error) {
	var results []course.CourseRecord
	for _, c := range m.courses {
		results = append(results, c)
	}
	return results, len(results), nil
}

func (m *mockRepo) Publish(ctx context.Context, id string) (course.CourseRecord, error) {
	c, ok := m.courses[id]
	if !ok || c.Status != "draft" {
		return c, course.ErrCourseNotFound
	}
	c.Status = "published"
	m.courses[id] = c
	return c, nil
}

func (m *mockRepo) AddModule(ctx context.Context, courseID, title string, position int) (course.ModuleRecord, error) {
	mod := course.ModuleRecord{ID: "m-" + title, CourseID: courseID, Title: title, Position: position}
	m.modules[mod.ID] = mod
	return mod, nil
}

func (m *mockRepo) AddLesson(ctx context.Context, courseID, moduleID, title, lessonType, contentID string, position int) (course.LessonRecord, error) {
	l := course.LessonRecord{ID: "l-" + title, ModuleID: moduleID, Title: title, Type: lessonType, Position: position}
	m.lessons[l.ID] = l
	return l, nil
}

func (m *mockRepo) GetModules(ctx context.Context, courseID string) ([]course.ModuleRecord, error) {
	var results []course.ModuleRecord
	for _, mod := range m.modules {
		if mod.CourseID == courseID {
			results = append(results, mod)
		}
	}
	return results, nil
}

func (m *mockRepo) GetLessons(ctx context.Context, moduleID string) ([]course.LessonRecord, error) {
	var results []course.LessonRecord
	for _, l := range m.lessons {
		if l.ModuleID == moduleID {
			results = append(results, l)
		}
	}
	return results, nil
}

func TestCreateCourse(t *testing.T) {
	h := course.NewHandler(newMockRepo())
	resp, err := h.CreateCourse(context.Background(), &coursepb.CreateCourseRequest{
		Title: "Test Course", Description: "Desc", Category: "AI", Difficulty: "beginner",
		InstructorId: "inst-1", TenantId: "t1",
	})
	if err != nil {
		t.Fatalf("CreateCourse failed: %v", err)
	}
	if resp.Status != "draft" {
		t.Errorf("expected draft, got %s", resp.Status)
	}
}

func TestCreateCourse_EmptyTitle(t *testing.T) {
	h := course.NewHandler(newMockRepo())
	_, err := h.CreateCourse(context.Background(), &coursepb.CreateCourseRequest{})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestPublishCourse(t *testing.T) {
	repo := newMockRepo()
	h := course.NewHandler(repo)
	created, _ := h.CreateCourse(context.Background(), &coursepb.CreateCourseRequest{
		Title: "Pub Course", TenantId: "t1", InstructorId: "i1",
	})
	resp, err := h.PublishCourse(context.Background(), &coursepb.PublishCourseRequest{CourseId: created.CourseId})
	if err != nil {
		t.Fatalf("PublishCourse failed: %v", err)
	}
	if resp.Status != "published" {
		t.Errorf("expected published, got %s", resp.Status)
	}
}

func TestAddModule(t *testing.T) {
	h := course.NewHandler(newMockRepo())
	resp, err := h.AddModule(context.Background(), &coursepb.AddModuleRequest{
		CourseId: "c-1", Title: "Module 1", Order: 1,
	})
	if err != nil {
		t.Fatalf("AddModule failed: %v", err)
	}
	if resp.Title != "Module 1" {
		t.Errorf("expected Module 1, got %s", resp.Title)
	}
}

func TestAddLesson(t *testing.T) {
	h := course.NewHandler(newMockRepo())
	resp, err := h.AddLesson(context.Background(), &coursepb.AddLessonRequest{
		CourseId: "c-1", ModuleId: "m-1", Title: "Intro", Type: "video", Order: 1,
	})
	if err != nil {
		t.Fatalf("AddLesson failed: %v", err)
	}
	if resp.Type != "video" {
		t.Errorf("expected video, got %s", resp.Type)
	}
}
