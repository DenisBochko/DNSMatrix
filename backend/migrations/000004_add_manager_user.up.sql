-- 000004_add_manager_user.up.sql

INSERT INTO sso.users (id, username, email, password, role, confirmed)
VALUES ('20f696af-32d4-49bf-97be-81926123c7a8',
        'Manager',
        'manager@gmail.com',
        '$2a$10$pVV0q0XYeUDbRUHBHKmhs.LWoxSb0goGDnUia.HZKCHfSM6JNeBQC',
        'manager',
        true);
