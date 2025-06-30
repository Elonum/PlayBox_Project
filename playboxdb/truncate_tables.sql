-- 1) Очистка всех пользователей
BEGIN;

TRUNCATE TABLE users
  RESTART IDENTITY
  CASCADE;

COMMIT;
