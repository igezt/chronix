ALTER TABLE scheduled_tasks
ADD COLUMN recurrence_interval INTEGER,
ADD COLUMN recurrence_limit INTEGER,
ADD COLUMN recurrence_count INTEGER DEFAULT 0;