-- Migration 004: Make originating_commit nullable with a conditional CHECK constraint

ALTER TABLE memory_record
ALTER COLUMN originating_commit DROP NOT NULL;

ALTER TABLE memory_record
ADD CONSTRAINT chk_originating_commit_by_source
CHECK (
    CASE
        WHEN source_type = 'pr_merge'
            THEN originating_commit IS NOT NULL
        ELSE
            TRUE
    END
);
