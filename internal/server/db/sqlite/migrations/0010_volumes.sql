-- Add persistent volume support for Volant VMs
-- Enables data persistence across VM restarts

-- Create volumes table
CREATE TABLE volumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,  -- ext4|squashfs|bind
    size_gb INTEGER NOT NULL,
    persistent BOOLEAN DEFAULT 1,
    host_path TEXT NOT NULL,
    backup_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create volume_mounts table
CREATE TABLE volume_mounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    volume_id INTEGER NOT NULL,
    vm_name TEXT NOT NULL,
    mount_point TEXT NOT NULL,
    read_only BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (volume_id) REFERENCES volumes(id) ON DELETE CASCADE,
    UNIQUE(vm_name, mount_point)  -- One mount per path per VM
);

-- Create indexes for efficient lookups
CREATE INDEX idx_volume_mounts_volume ON volume_mounts(volume_id);
CREATE INDEX idx_volume_mounts_vm ON volume_mounts(vm_name);
