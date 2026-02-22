package assessment

import (
	"context"
	"errors"
	"time"

	assessmentpb "github.com/amnayem/skillforge/shared/pb/assessment"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrAssessmentNotFound = errors.New("assessment not found")
	ErrSubmissionNotFound = errors.New("submission not found")
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateAssessment(ctx context.Context, a *assessmentpb.Assessment) (*assessmentpb.Assessment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var createdAt, updatedAt time.Time
	query := `
		INSERT INTO assessments (tenant_id, lesson_id, title, description, passing_score_percentage)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query, a.TenantId, a.LessonId, a.Title, a.Description, a.PassingScorePercentage).
		Scan(&a.Id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	a.CreatedAt = timestamppb.New(createdAt)
	a.UpdatedAt = timestamppb.New(updatedAt)

	for i, q := range a.Questions {
		qQuery := `
			INSERT INTO questions (assessment_id, type, text, correct_answer, points, position)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`
		var qID string
		err = tx.QueryRow(ctx, qQuery, a.Id, q.Type, q.Text, q.CorrectAnswer, q.Points, i).Scan(&qID)
		if err != nil {
			return nil, err
		}
		a.Questions[i].Id = qID
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

func (r *PostgresRepository) GetAssessment(ctx context.Context, id string) (*assessmentpb.Assessment, error) {
	query := `
		SELECT id, tenant_id, lesson_id, title, description, passing_score_percentage, created_at, updated_at
		FROM assessments
		WHERE id = $1
	`
	var a assessmentpb.Assessment
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.Id, &a.TenantId, &a.LessonId, &a.Title, &a.Description, &a.PassingScorePercentage, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssessmentNotFound
		}
		return nil, err
	}
	a.CreatedAt = timestamppb.New(createdAt)
	a.UpdatedAt = timestamppb.New(updatedAt)

	qQuery := `
		SELECT id, type, text, correct_answer, points
		FROM questions
		WHERE assessment_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.Query(ctx, qQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var q assessmentpb.Question
		err := rows.Scan(&q.Id, &q.Type, &q.Text, &q.CorrectAnswer, &q.Points)
		if err != nil {
			return nil, err
		}
		a.Questions = append(a.Questions, &q)
	}

	return &a, nil
}

func (r *PostgresRepository) SubmitAssessment(ctx context.Context, studentID string, req *assessmentpb.SubmitAssessmentRequest) (*assessmentpb.SubmissionResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Fetch assessment questions for grading
	qQuery := `SELECT id, correct_answer, points FROM questions WHERE assessment_id = $1`
	rows, err := tx.Query(ctx, qQuery, req.AssessmentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := make(map[string]struct {
		correct string
		points  int32
	})
	var totalPoints int32 = 0
	for rows.Next() {
		var id, correct string
		var points int32
		if err := rows.Scan(&id, &correct, &points); err != nil {
			return nil, err
		}
		questions[id] = struct {
			correct string
			points  int32
		}{correct, points}
		totalPoints += points
	}
	rows.Close()

	if len(questions) == 0 {
		return nil, ErrAssessmentNotFound
	}

	var earnedPoints int32 = 0
	answerRecords := make([]struct {
		qid           string
		ans           string
		pointsAwarded int32
		isCorrect     bool
	}, 0)

	for _, ans := range req.Answers {
		q, ok := questions[ans.QuestionId]
		if !ok {
			continue
		}
		isCorrect := (ans.AnswerText == q.correct)
		pointsAwarded := int32(0)
		if isCorrect {
			pointsAwarded = q.points
			earnedPoints += pointsAwarded
		}
		answerRecords = append(answerRecords, struct {
			qid           string
			ans           string
			pointsAwarded int32
			isCorrect     bool
		}{ans.QuestionId, ans.AnswerText, pointsAwarded, isCorrect})
	}

	// Fetch passing score
	var passPercentage int32
	err = tx.QueryRow(ctx, `SELECT passing_score_percentage FROM assessments WHERE id = $1`, req.AssessmentId).Scan(&passPercentage)
	if err != nil {
		return nil, err
	}

	achievedPercentage := int32(0)
	if totalPoints > 0 {
		achievedPercentage = (earnedPoints * 100) / totalPoints
	}
	passed := achievedPercentage >= passPercentage

	// Insert submission
	var submissionID string
	sQuery := `
		INSERT INTO submissions (assessment_id, student_id, status, total_points, earned_points, passed, graded_at)
		VALUES ($1, $2, 'graded', $3, $4, $5, NOW())
		RETURNING id
	`
	err = tx.QueryRow(ctx, sQuery, req.AssessmentId, studentID, totalPoints, earnedPoints, passed).Scan(&submissionID)
	if err != nil {
		return nil, err
	}

	// Insert answers
	for _, rec := range answerRecords {
		aQuery := `
			INSERT INTO submission_answers (submission_id, question_id, answer_text, points_awarded, is_correct)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err = tx.Exec(ctx, aQuery, submissionID, rec.qid, rec.ans, rec.pointsAwarded, rec.isCorrect)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &assessmentpb.SubmissionResponse{
		SubmissionId: submissionID,
		Status:       "graded",
	}, nil
}

func (r *PostgresRepository) GetGrade(ctx context.Context, assessmentID, submissionID string) (*assessmentpb.GradeResponse, error) {
	query := `
		SELECT total_points, earned_points, passed, COALESCE(feedback, '')
		FROM submissions
		WHERE id = $1 AND assessment_id = $2
	`
	var res assessmentpb.GradeResponse
	err := r.db.QueryRow(ctx, query, submissionID, assessmentID).Scan(
		&res.TotalPoints, &res.EarnedPoints, &res.Passed, &res.Feedback,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}
	res.SubmissionId = submissionID
	return &res, nil
}
