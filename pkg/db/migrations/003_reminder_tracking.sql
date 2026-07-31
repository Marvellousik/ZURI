-- Migration 003: Add last_reminded_at to support persistent reminder cadence

ALTER TABLE memory_record
ADD COLUMN IF NOT EXISTS last_reminded_at TIMESTAMPTZ;
