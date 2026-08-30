ALTER TABLE sticker DROP CONSTRAINT IF EXISTS sticker_sid_pack_fk;
DROP INDEX IF EXISTS sticker_image_hash_idx;
ALTER TABLE sticker DROP COLUMN IF EXISTS image_hash;
DROP TABLE IF EXISTS sticker_pack;
