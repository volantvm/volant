-- Create stack_vms junction table
-- Links stacks to their VMs (many-to-many relationship)

CREATE TABLE IF NOT EXISTS stack_vms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    stack_id INTEGER NOT NULL,
    vm_id TEXT NOT NULL,
    role TEXT, -- Optional: role of VM in the stack (e.g., 'web', 'db', 'cache')
    ordinal INTEGER DEFAULT 0, -- Order within the stack for startup/shutdown
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (stack_id) REFERENCES stacks(id) ON DELETE CASCADE,
    FOREIGN KEY (vm_id) REFERENCES vms(id) ON DELETE CASCADE,
    UNIQUE(stack_id, vm_id)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_stack_vms_stack_id ON stack_vms(stack_id);
CREATE INDEX idx_stack_vms_vm_id ON stack_vms(vm_id);
CREATE INDEX idx_stack_vms_ordinal ON stack_vms(stack_id, ordinal);