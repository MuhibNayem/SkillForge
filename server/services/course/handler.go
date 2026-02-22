package course

import (
	"context"
	"errors"

	"github.com/amnayem/skillforge/shared/pb/coursepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	coursepb.UnimplementedCourseServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateCourse(ctx context.Context, req *coursepb.CreateCourseRequest) (*coursepb.CourseResponse, error) {
	if req.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "title is required")
	}
	c, err := h.repo.Create(ctx, req.TenantId, req.InstructorId, req.Title, req.Description, req.Category, req.Difficulty)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create course: %v", err)
	}
	return courseToProto(c, nil), nil
}

func (h *Handler) GetCourse(ctx context.Context, req *coursepb.GetCourseRequest) (*coursepb.CourseResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	c, err := h.repo.GetByID(ctx, req.CourseId)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, status.Errorf(codes.NotFound, "course not found")
		}
		return nil, status.Errorf(codes.Internal, "get course: %v", err)
	}
	// Eagerly load modules + lessons
	modules, _ := h.repo.GetModules(ctx, c.ID)
	var protoModules []*coursepb.ModuleResponse
	for _, m := range modules {
		lessons, _ := h.repo.GetLessons(ctx, m.ID)
		var protoLessons []*coursepb.LessonResponse
		for _, l := range lessons {
			protoLessons = append(protoLessons, &coursepb.LessonResponse{
				LessonId: l.ID, Title: l.Title, Type: l.Type, ContentId: l.ContentID, Order: int32(l.Position),
			})
		}
		protoModules = append(protoModules, &coursepb.ModuleResponse{
			ModuleId: m.ID, Title: m.Title, Order: int32(m.Position), Lessons: protoLessons,
		})
	}
	return courseToProto(c, protoModules), nil
}

func (h *Handler) UpdateCourse(ctx context.Context, req *coursepb.UpdateCourseRequest) (*coursepb.CourseResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	c, err := h.repo.Update(ctx, req.CourseId, req.Title, req.Description, req.Category, req.Difficulty, req.ThumbnailUrl)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, status.Errorf(codes.NotFound, "course not found")
		}
		return nil, status.Errorf(codes.Internal, "update course: %v", err)
	}
	return courseToProto(c, nil), nil
}

func (h *Handler) DeleteCourse(ctx context.Context, req *coursepb.DeleteCourseRequest) (*coursepb.DeleteCourseResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	if err := h.repo.Delete(ctx, req.CourseId); err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, status.Errorf(codes.NotFound, "course not found")
		}
		return nil, status.Errorf(codes.Internal, "delete course: %v", err)
	}
	return &coursepb.DeleteCourseResponse{Success: true}, nil
}

func (h *Handler) ListCourses(ctx context.Context, req *coursepb.ListCoursesRequest) (*coursepb.ListCoursesResponse, error) {
	courses, total, err := h.repo.List(ctx, req.TenantId, req.Category, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list courses: %v", err)
	}
	resp := &coursepb.ListCoursesResponse{Total: int32(total)}
	for _, c := range courses {
		resp.Courses = append(resp.Courses, courseToProto(c, nil))
	}
	return resp, nil
}

func (h *Handler) PublishCourse(ctx context.Context, req *coursepb.PublishCourseRequest) (*coursepb.CourseResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	c, err := h.repo.Publish(ctx, req.CourseId)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			return nil, status.Errorf(codes.NotFound, "course not found or already published")
		}
		return nil, status.Errorf(codes.Internal, "publish course: %v", err)
	}
	return courseToProto(c, nil), nil
}

func (h *Handler) AddModule(ctx context.Context, req *coursepb.AddModuleRequest) (*coursepb.ModuleResponse, error) {
	if req.CourseId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id is required")
	}
	m, err := h.repo.AddModule(ctx, req.CourseId, req.Title, int(req.Order))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add module: %v", err)
	}
	return &coursepb.ModuleResponse{ModuleId: m.ID, Title: m.Title, Order: int32(m.Position)}, nil
}

func (h *Handler) AddLesson(ctx context.Context, req *coursepb.AddLessonRequest) (*coursepb.LessonResponse, error) {
	if req.CourseId == "" || req.ModuleId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "course_id and module_id are required")
	}
	l, err := h.repo.AddLesson(ctx, req.CourseId, req.ModuleId, req.Title, req.Type, req.ContentId, int(req.Order))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add lesson: %v", err)
	}
	return &coursepb.LessonResponse{LessonId: l.ID, Title: l.Title, Type: l.Type, ContentId: l.ContentID, Order: int32(l.Position)}, nil
}

func courseToProto(c CourseRecord, modules []*coursepb.ModuleResponse) *coursepb.CourseResponse {
	return &coursepb.CourseResponse{
		CourseId:     c.ID,
		Title:        c.Title,
		Description:  c.Description,
		Category:     c.Category,
		Difficulty:   c.Difficulty,
		Status:       c.Status,
		InstructorId: c.InstructorID,
		TenantId:     c.TenantID,
		ThumbnailUrl: c.ThumbnailURL,
		Modules:      modules,
		CreatedAt:    c.CreatedAt.Unix(),
		UpdatedAt:    c.UpdatedAt.Unix(),
	}
}
