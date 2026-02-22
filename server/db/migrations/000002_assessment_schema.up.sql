-- Assessment Service Schema
-- Tables: assessments, questions, submissions, submission_answers

CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT DEFAULT '',
    passing_score_percentage INTEGER NOT NULL DEFAULT 70 CHECK (passing_score_percentage BETWEEN 0 AND 100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for finding assessments by lesson
CREATE INDEX idx_assessments_lesson_id ON assessments(lesson_id);

CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('multiple_choice', 'true_false', 'short_answer', 'essay')),
    text TEXT NOT NULL,
    options JSONB DEFAULT '[]', -- Array of strings for MC options
    correct_answer TEXT DEFAULT '', -- Expected answer for auto-grading
    points INTEGER NOT NULL DEFAULT 10 CHECK (points >= 0),
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_questions_assessment_id ON questions(assessment_id);

CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending_review' CHECK (status IN ('pending_review', 'graded')),
    total_points INTEGER NOT NULL DEFAULT 0,
    earned_points INTEGER NOT NULL DEFAULT 0,
    passed BOOLEAN NOT NULL DEFAULT FALSE,
    feedback TEXT DEFAULT '',
    submitted_at TIMESTAMPTZ DEFAULT NOW(),
    graded_at TIMESTAMPTZ
);

CREATE INDEX idx_submissions_assessment_id ON submissions(assessment_id);
CREATE INDEX idx_submissions_student_id ON submissions(student_id);

CREATE TABLE IF NOT EXISTS submission_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer_text TEXT NOT NULL,
    points_awarded INTEGER NOT NULL DEFAULT 0,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    instructor_comment TEXT DEFAULT '',
    UNIQUE(submission_id, question_id)
);

-- Add update triggers (using existing trigger function from 000001)
CREATE TRIGGER set_timestamp_assessments
BEFORE UPDATE ON assessments
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_questions
BEFORE UPDATE ON questions
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();
