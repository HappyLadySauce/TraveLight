package model

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/HappyLadySauce/TraveLight/pkg/craw"
)

// AutoMigrate migrates all business tables and PostgreSQL features.
// AutoMigrate 迁移全部业务表与 PostgreSQL 特性。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&craw.Attraction{}, &User{}, &Article{}, &Comment{}); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}
	if err := ensureCommentConstraint(db); err != nil {
		return err
	}
	if err := ensureSearchInfrastructure(db); err != nil {
		return err
	}
	return nil
}

func ensureCommentConstraint(db *gorm.DB) error {
	sql := `
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'comments_content_type_check'
	) THEN
		ALTER TABLE comments
		ADD CONSTRAINT comments_content_type_check
		CHECK (content_type IN ('article','attraction'));
	END IF;
END$$;
`
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("ensure comments constraint failed: %w", err)
	}
	return nil
}

func ensureSearchInfrastructure(db *gorm.DB) error {
	searchSQL := []string{
		`ALTER TABLE articles ADD COLUMN IF NOT EXISTS search_vector tsvector;`,
		`ALTER TABLE attractions ADD COLUMN IF NOT EXISTS search_vector tsvector;`,
		`CREATE INDEX IF NOT EXISTS idx_articles_search_vector ON articles USING GIN (search_vector);`,
		`CREATE INDEX IF NOT EXISTS idx_attractions_search_vector ON attractions USING GIN (search_vector);`,
		`CREATE INDEX IF NOT EXISTS idx_attractions_view_count ON attractions (view_count DESC);`,
		`
CREATE OR REPLACE FUNCTION update_articles_search_vector() RETURNS trigger AS $$
BEGIN
	NEW.search_vector := to_tsvector('simple', coalesce(NEW.title,'') || ' ' || coalesce(NEW.content,''));
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`,
		`
CREATE OR REPLACE FUNCTION update_attractions_search_vector() RETURNS trigger AS $$
BEGIN
	NEW.search_vector := to_tsvector('simple', coalesce(NEW.name,'') || ' ' || coalesce(NEW.description,''));
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`,
		`
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_articles_search_vector'
	) THEN
		CREATE TRIGGER trigger_articles_search_vector
		BEFORE INSERT OR UPDATE ON articles
		FOR EACH ROW EXECUTE FUNCTION update_articles_search_vector();
	END IF;
END$$;
`,
		`
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_attractions_search_vector'
	) THEN
		CREATE TRIGGER trigger_attractions_search_vector
		BEFORE INSERT OR UPDATE ON attractions
		FOR EACH ROW EXECUTE FUNCTION update_attractions_search_vector();
	END IF;
END$$;
`,
		`UPDATE articles SET search_vector = to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,'')) WHERE search_vector IS NULL;`,
		`UPDATE attractions SET search_vector = to_tsvector('simple', coalesce(name,'') || ' ' || coalesce(description,'')) WHERE search_vector IS NULL;`,
	}
	for _, statement := range searchSQL {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("ensure search infrastructure failed: %w", err)
		}
	}
	return nil
}
