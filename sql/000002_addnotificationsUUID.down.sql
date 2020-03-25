ALTER TABLE notifications
    DROP COLUMN UUID;

ALTER TABLE notifications
    DROP INDEX notifications_uuid_uindex;
