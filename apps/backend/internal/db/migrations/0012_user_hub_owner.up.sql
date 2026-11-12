-- 0012_user_hub_owner — marks the primary operator account tied to existing hub data.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS owns_hub_data boolean NOT NULL DEFAULT false;
