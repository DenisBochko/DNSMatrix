-- migrations/XXXXXX_create_faqs_table.sql
CREATE TABLE IF NOT EXISTS sso.faqs (
                                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    category VARCHAR(100) NOT NULL,
    "order" INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES sso.users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );

-- Индексы для улучшения производительности
CREATE INDEX IF NOT EXISTS idx_faqs_category ON sso.faqs(category);
CREATE INDEX IF NOT EXISTS idx_faqs_is_active ON sso.faqs(is_active);
CREATE INDEX IF NOT EXISTS idx_faqs_deleted_at ON sso.faqs(deleted_at);
CREATE INDEX IF NOT EXISTS idx_faqs_order ON sso.faqs("order");
CREATE INDEX IF NOT EXISTS idx_faqs_created_at ON sso.faqs(created_at);

-- Уникальный индекс для предотвращения дубликатов
CREATE UNIQUE INDEX IF NOT EXISTS idx_faqs_question_unique
    ON sso.faqs(LOWER(question)) WHERE deleted_at IS NULL;