-- Rename 'plugins' table to 'images' for clarity
-- Images are templates/build artifacts, VMs are instances created from images

-- Rename plugins table to images
ALTER TABLE plugins RENAME TO images;

-- Rename plugin_artifacts table to image_artifacts
ALTER TABLE plugin_artifacts RENAME TO image_artifacts;

-- Rename plugin_name column to image_name in image_artifacts
ALTER TABLE image_artifacts RENAME COLUMN plugin_name TO image_name;

-- Drop old index and create new one with updated naming
DROP INDEX IF EXISTS idx_plugin_artifacts_plugin_kind;
CREATE INDEX IF NOT EXISTS idx_image_artifacts_image_kind ON image_artifacts(image_name, version, kind);
