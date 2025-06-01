ALTER TABLE scheduled_tasks
ADD COLUMN recurrence_interval INTERVAL,
ADD COLUMN recurrence_limit INTEGER,
ADD COLUMN recurrence_count INTEGER DEFAULT 0;