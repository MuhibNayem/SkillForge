-- +migrate Up
CREATE TABLE certificate_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    background_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_cert_templates_tenant_id ON certificate_templates(tenant_id);

CREATE TABLE certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    student_id UUID NOT NULL,
    course_id UUID NOT NULL,
    template_id UUID NOT NULL REFERENCES certificate_templates(id),
    pdf_url VARCHAR(255) NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(student_id, course_id)
);

CREATE INDEX idx_certs_tenant_id ON certificates(tenant_id);
CREATE INDEX idx_certs_student_id ON certificates(student_id);
CREATE INDEX idx_certs_course_id ON certificates(course_id);

-- +migrate Down
DROP TABLE IF EXISTS certificates;
DROP TABLE IF EXISTS certificate_templates;
