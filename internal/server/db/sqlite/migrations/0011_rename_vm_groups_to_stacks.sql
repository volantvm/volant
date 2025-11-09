-- Rename vm_groups to stacks for terminology unification
ALTER TABLE vm_groups RENAME TO stacks;

-- Update the vms foreign key column name for clarity
-- Note: SQLite doesn't support renaming columns in older versions,
-- so we keep group_id for backward compatibility but document it refers to stacks
